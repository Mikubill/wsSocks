package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type wsConn struct {
	*websocket.Conn
}

type webSocket struct {
	hashFunc func(b []byte, seed uint64) []byte
	conn     *wsConn
	lock     sync.Mutex
	id       []byte
	buf      *bytes.Buffer
	b        []byte
	closed   bool
}

const (
	wsAddrLen   = 8
	connAddrLen = 4
	digit       = 8
	wsReadBuf   = 63 * 1024
	wsWriteBuf  = 63 * 1024
)

var (
	flagDial  = []byte("0")
	flagData  = []byte("1")
	flagClose = []byte("2")
	flagLoop  = []byte("3")

	wsKeys     [][]byte
	wsLen      int
	mainServer string
)

func (ws *webSocket) Reader() (err error) {
	addressBuf := make([]byte, connAddrLen)
	hashBuf := make([]byte, digit)
	controlBuf := make([]byte, 1)
	var dataBuf []byte
	for {
		err = ws.Read()
		if err != nil {
			return
		}
		if ws.buf.Len() < connAddrLen+digit {
			log.Warnf("illegal connection %v <-> %v, denied.", ws.conn.LocalAddr(), ws.conn.RemoteAddr())
			_ = ws.conn.Close()
			return
		}

		copy(addressBuf, ws.b)
		copy(controlBuf, ws.b[connAddrLen:])
		copy(hashBuf, ws.b[len(ws.b)-digit:])
		dataBuf = ws.b[connAddrLen+1 : len(ws.b)-digit]

		log.Debugf("frame %x received, len %v", addressBuf, len(dataBuf))
		// verify hmacBlock
		if !validateCode(dataBuf, hashBuf, ws.hashFunc) {
			log.Warnf("invalid hash %v <-> %v, denied.", ws.conn.LocalAddr(), ws.conn.RemoteAddr())
			_ = ws.conn.Close()
			return
		}
		atomic.AddInt64(&downloaded, int64(len(ws.b)))
		if bytes.Equal(controlBuf, flagData) {
			if c, ok := connPool.Load(u32(addressBuf)); ok {
				log.Debugf("data frame %x accepted", addressBuf)
				c := c.(*muxConn)
				_, err = c.pipeW.Write(dataBuf)
				if err != nil {
					_ = c.Close()
				}
			} else {
				log.Debugf("data frame %x accepted, but conn not found", addressBuf)
				_, _ = ws.writeData(addressBuf, flagClose, nil)
			}
		} else if bytes.Equal(controlBuf, flagDial) {
			// server only
			log.Debugf("dial frame %x accepted", addressBuf)
			c := &muxConn{
				id: addressBuf,
				ws: ws,
			}
			c.pipeR, c.pipeW = newPipe()
			// wait until dial finish
			connPool.Store(u32(addressBuf), c)
			host := string(dataBuf)
			go server.dialHandler(host, c)
		} else if bytes.Equal(controlBuf, flagClose) {
			if s, ok := connPool.Load(u32(addressBuf)); ok {
				log.Debugf("close frame %x accepted", addressBuf)
				connPool.Delete(u32(addressBuf))
				s.(*muxConn).closeStuff()
			} else {
				log.Debugf("close frame %x accepted, but conn not found", addressBuf)
			}
		} else if bytes.Equal(controlBuf, flagLoop) {
			_, _ = ws.writeData(addressBuf, flagClose, dataBuf)
		} else {
			log.Warnf("unknown flag: %x", addressBuf)
		}
	}
}

func (ws *webSocket) writeData(prefix, flag, p []byte) (n int, err error) {
	if ws.closed {
		return 0, fmt.Errorf("use of closed websocket")
	}

	err = ws.write(prefix, flag, p)

	if err != nil {
		log.Printf("error writing message with length %v, %v", len(p), err)
		//err = ws.write(prefix, flag, p)
		return
	}
	atomic.AddInt64(&uploaded, int64(len(p)))
	log.Debugf("%v written", int64(len(p)))
	return len(p), nil
}

func (ws *webSocket) write(prefix, flag, p []byte) (err error) {
	ws.lock.Lock()
	//log.Warn(len(prefix), len(flag), len(p), len(generateCode(p, ws.hashFunc)))
	w, err := ws.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	_, err = w.Write(prefix)
	_, err = w.Write(flag)
	_, err = w.Write(p)
	_, err = w.Write(generateCode(p, ws.hashFunc))
	if err != nil {
		return err
	}
	err = w.Close()
	w = nil
	ws.lock.Unlock()
	return
}

func (ws *webSocket) close() {
	log.Warnf("websocket connection closed: %v", u64(ws.id))
	ws.closed = true
	wsPool.Delete(u64(ws.id))
	_ = ws.conn.Close()
}

func startWs(id []byte) (ws *webSocket) {
	var conn *websocket.Conn
	var err error
	newDialer := &websocket.Dialer{
		ReadBufferSize:   wsReadBuf, // Expected average message size
		WriteBufferSize:  wsWriteBuf,
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tlsConfig,
	}
	for {
		conn, _, err = newDialer.Dial(mainServer, map[string][]string{
			"Auth": {hex.EncodeToString(generateCode([]byte("authenticate"), hashWorker))},
			"via":  {hashFlag},
		})
		if err == nil {
			break
		} else {
			log.Warnf("dialing new websocket failed: %s", err.Error())
		}
		time.Sleep(time.Second)
	}
	ws = &webSocket{
		id:       id,
		hashFunc: hashWorker,
		conn:     &wsConn{conn},
		closed:   false,
		buf:      bytes.NewBuffer(make([]byte, 32*1024)),
	}
	taskAdd(func() {
		err := ws.Reader()
		ws.close()
		log.Warnf("websocket connection %v closed", u64(ws.id))
		if err != nil {
			log.Warn(err)
		}
	})
	return
}

func (ws *webSocket) Read() (err error) {
	var r io.Reader
	_, r, err = ws.conn.NextReader()
	if err != nil {
		return err
	}
	ws.buf.Reset()
	ws.buf.Grow(32 * 1024)
	_, err = ws.buf.ReadFrom(r)
	ws.b = ws.buf.Bytes()
	return err
}
