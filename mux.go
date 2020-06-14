package main

import (
	"io"
	"math/rand"
	"net"
	"sync"
)

type muxConn struct {
	io.ReadWriter
	pipeW *io.PipeWriter
	pipeR *io.PipeReader
	conn  net.Conn
	id    []byte
	ws    *webSocket
}

type wPool struct {
	sync.Map
}

var (
	wsPool   = new(wPool)
	connPool = new(sync.Map)
)

func (c *wPool) getWs() (ws *webSocket) {
	id := wsKeys[rand.Intn(wsLen)]
	if s, ok := c.Load(u64(id)); !ok {
		ws = startWs(id)
		c.Store(u64(id), ws)
		return ws
	} else {
		return s.(*webSocket)
	}
}

func createConn(conn net.Conn) (c *muxConn) {
	c = &muxConn{
		id:  genRandBytes(connAddrLen),
		conn: conn,
		ws:   wsPool.getWs(),
	}
	c.pipeR, c.pipeW = io.Pipe()
	connPool.Store(u32(c.id), c)
	return
}

func (c *muxConn) dial(host Addr) (n int, err error) {
	n, err = c.send(c.id, flagDial, []byte(host.String()))
	return
}

func (c *muxConn) closeStuff() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
	if c.pipeR != nil {
		_ = c.pipeW.Close()
		_ = c.pipeR.Close()
	}
}

func (c *muxConn) Close() (err error) {
	connPool.Delete(u32(c.id))
	c.closeStuff()
	_, err = c.send(c.id, flagClose, nil)
	return
}

func (c *muxConn) Write(p []byte) (n int, err error) {
	n, err = c.send(c.id, flagData, p)
	return
}

func (c *muxConn) bench(p []byte) (n int, err error) {
	n, err = c.send(c.id, flagLoop, p)
	return
}

func (c *muxConn) send(prefix, flag, p []byte) (n int, err error) {
	n, err = c.ws.writeData(prefix, flag, p)
	return
}
