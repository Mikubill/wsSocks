package main

import (
	"github.com/panjf2000/ants/v2"
	"io"
	"net"
	"time"
)

type dataPack struct {
	netConn net.Conn
	muxConn *muxConn
	ch      chan struct{}
}

var mainThread *ants.Pool

var transfer, _ = ants.NewPoolWithFunc(500000, func(i interface{}) {
	pack := i.(*dataPack)
	pack.ch = make(chan struct{})
	_ = receiver.Invoke(pack)
	_, err := io.Copy(pack.muxConn, pack.netConn)
	if err != nil {
		log.Debug("connection copy error: ", err)
	}
	<-pack.ch
	_ = pack.muxConn.Close()
})

var receiver, _ = ants.NewPoolWithFunc(500000, func(i interface{}) {
	pack := i.(*dataPack)
	defer func() { pack.ch <- struct{}{} }()
	_, err := io.Copy(pack.netConn, pack.muxConn.pipeR)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		log.Debug("connection copy error: ", err)
	}
	_ = pack.netConn.SetReadDeadline(time.Now()) // unblock read on right
})

var wsHandler = func(ws *webSocket) {
	err := ws.Reader()
	ws.close()
	log.Warnf("websocket connection %x closed", ws.id)
	if err != nil {
		log.Warn(err)
	}
}

func taskAdd(f func()) {
	for {
		err := mainThread.Submit(f)
		if err == nil {
			break
		}
	}
}
