package main

import (
	"github.com/panjf2000/ants/v2"
	"io"
	"sync"
)

type dataPack struct {
	reader io.ReadCloser
	writer io.WriteCloser
	close  bool
}

//type trsPack struct {
//	dst  io.WriteCloser
//	data *[]byte
//}

var mainThread *ants.Pool

var wBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 30*1024)
	},
}

var transfer, _ = ants.NewPoolWithFunc(500000, func(i interface{}) {
	pack := i.(*dataPack)
	buf := wBufPool.Get().([]byte)
	_, err := io.Copy(pack.writer, pack.reader)
	if err != nil {
		log.Debug("connection copy error: ", err)
	}
	wBufPool.Put(buf)
	_ = pack.writer.Close()
	_ = pack.reader.Close()
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
