package main

import (
	"github.com/gorilla/websocket"
	"net"
	"net/url"
	"time"
)

type Client struct {
	Connections int
	ListenAddr  *net.TCPAddr
	ServerAddr  *url.URL
	Dialer      *websocket.Dialer
	CreatedAt   time.Time
}

func (client *Client) Listen() (err error) {
	// init
	for i := 0; i < client.Connections; i++ {
		wsKeys = append(wsKeys, genRandBytes(wsAddrLen))
	}
	wsLen = client.Connections
	mainServer = client.ServerAddr.String()

	listener, err := net.ListenTCP("tcp", client.ListenAddr)
	if err != nil {
		return err
	}

	log.Infof("Listening at %s", client.ListenAddr.String())
	defer func() {
		err = listener.Close()
		if err != nil {
			log.Infoln("listener ends with error: ", err)
		}
	}()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Infoln("socks conn ends with error: ", err)
			continue
		}

		go client.handleConn(conn)
	}
}

func (client *Client) handleConn(conn *net.TCPConn) {

	err := conn.SetLinger(0)
	if err != nil {
		_ = conn.Close()
		return
	}

	addr, err := Handshake(conn)
	if err != nil {
		_ = conn.Close()
		return
	}

	log.Debugln(addr.String())
	ws := createConn(conn)

	_, err = ws.dial(addr)
	if err != nil {
		_ = ws.Close()
		return
	}

	err = transfer.Invoke(&dataPack{
		reader: conn,
		writer: ws,
	})
	if err != nil {
		log.Warn(err)
		_ = conn.Close()
		_ = ws.Close()
		return
	}

	err = transfer.Invoke(&dataPack{
		writer: conn,
		reader: ws.pipeR,
	})
	if err != nil {
		log.Warn(err)
		_ = conn.Close()
		_ = ws.Close()
		return
	}
}
