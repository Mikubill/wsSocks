package main

import (
	"github.com/gorilla/websocket"
	"net"
	"net/url"
	"os"
	"time"
)

type Client struct {
	Connections   int
	ListenTCPAddr *net.TCPAddr
	ServerAddr    *url.URL
	Dialer        *websocket.Dialer
	CreatedAt     time.Time
}

func (client *Client) Listen() {
	// init
	for i := 0; i < client.Connections; i++ {
		wsKeys = append(wsKeys, genRandBytes(wsAddrLen))
	}
	wsLen = client.Connections
	mainServer = client.ServerAddr.String()
	log.Infof("Listening at %s", client.ListenTCPAddr.String())

	client.listenTCP()
}

func (client *Client) listenTCP() {
	listener, err := net.ListenTCP("tcp", client.ListenTCPAddr)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

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
		netConn: conn,
		muxConn: ws,
	})
	if err != nil {
		log.Warn("invoke error: %V", err)
		_ = ws.Close()
		return
	}
}
