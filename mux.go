package main

import (
	"io"
	"net"
)

type muxConn struct {
	io.ReadWriter
	pipeW *io.PipeWriter
	pipeR *io.PipeReader
	conn  net.Conn
	id    []byte
	ws    *webSocket
}

var (
	wsPool   = newPool()
	connPool = newPool()
)

func createConn(conn net.Conn) (c *muxConn) {
	c = &muxConn{
		conn: conn,
		ws:   wsPool.getWs(),
	}
	c.pipeR, c.pipeW = io.Pipe()
	connPool.addConn(c)
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
	connPool.RemoveCb(string(c.id), func(key string, v interface{}, exists bool) bool {
		c.closeStuff()
		return true
	})
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
