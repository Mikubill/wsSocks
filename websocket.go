package main

import (
	"bytes"
	"encoding/hex"
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
	id       string
	buf      *bytes.Buffer
	b        []byte
	closed   bool
}

const (
	pref  = 3
	digit = 8
)

var (
	flagDial  = []byte("0")
	flagData  = []byte("1")
	flagClose = []byte("2")
	flagLoop  = []byte("3")
)

func (ws *webSocket) Reader() (err error) {
	addressBuf := make([]byte, pref)
	hashBuf := make([]byte, digit)
	controlBuf := make([]byte, 1)
	var dataBuf []byte
	for {
		err = ws.Read()
		if err != nil {
			return
		}
		if ws.buf.Len() < pref+digit {
			log.Warnf("illegal connection %v <-> %v, denied.", ws.conn.LocalAddr(), ws.conn.RemoteAddr())
			_ = ws.conn.Close()
			return
		}

		copy(addressBuf, ws.b)
		copy(controlBuf, ws.b[pref:])
		copy(hashBuf, ws.b[len(ws.b)-digit:])
		dataBuf = ws.b[pref+1 : len(ws.b)-digit]

		// verify hmacBlock
		if !validateCode(dataBuf, hashBuf, ws.hashFunc) {
			log.Warnf("invalid hash %v <-> %v, denied.", ws.conn.LocalAddr(), ws.conn.RemoteAddr())
			_ = ws.conn.Close()
			return
		}
		atomic.AddInt64(&downloaded, int64(len(ws.b)))
		if bytes.Equal(controlBuf, flagData) {
			if c, ok := connPool.Get(string(addressBuf)); ok {
				log.Debugf("data frame %x accpeted", addressBuf)
				c := c.(*muxConn)
				_, err = c.pipeW.Write(dataBuf)
				if err != nil {
					_ = c.Close()
				}
			} else {
				log.Debugf("data frame %x accpeted, but conn not found", addressBuf)
				_, _ = ws.writeData(addressBuf, flagClose, nil)
			}
		} else if bytes.Equal(controlBuf, flagDial) {
			// server only
			log.Debugf("dial frame %x accpeted", addressBuf)
			c := &muxConn{
				id: addressBuf,
				ws: ws,
			}
			c.pipeR, c.pipeW = io.Pipe()
			// wait until dial finish
			connPool.addConn(c)
			host := string(dataBuf)
			taskAdd(func() { server.dialHandler(host, c) })
		} else if bytes.Equal(controlBuf, flagClose) {
			connPool.RemoveCb(string(addressBuf), func(key string, v interface{}, exists bool) bool {
				if exists {
					log.Debugf("Close frame %x accpeted", addressBuf)
					v.(*muxConn).closeStuff()
				}
				return true
			})
		} else if bytes.Equal(controlBuf, flagLoop) {
			_, _ = ws.writeData(addressBuf, flagClose, dataBuf)
		} else {
			log.Warnf("unknown flag: %x", addressBuf)
		}
	}
}

func (ws *webSocket) writeData(prefix, flag, p []byte) (n int, err error) {
	err = ws.write(prefix, flag, p)
	if err != nil {
		log.Printf("error writing message with length %v, %v", len(p), err)
		for i := 0; i < 10; i++ {
			if len(wsPool.keys) > 0 {
				// websocket might failed
				ws.close()
				// retry
				log.Warnf("Connection closed. switching")
				err = wsPool.getWs().write(prefix, flag, p)
			}
		}
		return
	}
	atomic.AddInt64(&uploaded, int64(len(p)))

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
	log.Warnf("websocket connection closed: %v", ws.id)
	ws.closed = true
	wsPool.RemoveWs(ws.id)
	_ = ws.conn.Close()
}

func addWs(server string, count int) {
	for {
		if len(wsPool.keys) < count {
			ws, err := startWs(server)
			if err != nil {
				log.Warnf("websocket connection failed to start: %v", err)
				time.Sleep(time.Second)
				continue
			}
			wsPool.addWs(ws)
			err = wsHandler.Invoke(ws)
			log.Debugf("websocket connection %v started. waiting traffic...", ws.id)
		}
		time.Sleep(time.Second)
	}
}

func startWs(server string) (ws *webSocket, err error) {
	newDialer := &websocket.Dialer{
		ReadBufferSize:   32768, // Expected average message size
		WriteBufferSize:  32768,
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tlsConfig,
	}
	conn, _, err := newDialer.Dial(server, map[string][]string{
		"Auth": {hex.EncodeToString(generateCode([]byte("authenticate"), hashWorker))},
		"via":  {hashFlag},
	})
	if err != nil {
		return
	}
	ws = &webSocket{
		hashFunc: hashWorker,
		conn:     &wsConn{conn},
		closed:   false,
		buf:      bytes.NewBuffer(make([]byte, 32*1024)),
	}
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
