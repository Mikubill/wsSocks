package main

import (
	"bytes"
	"crypto/rand"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

var (
	uploaded   int64
	downloaded int64
)

type Server struct {
	ListenAddr *url.URL
	Reverse    *url.URL
	Cert       string
	PrivateKey string
	Resolver   *websocket.Upgrader
	CreatedAt  time.Time
}

func (server *Server) dialHandler(host string, c *muxConn) {
	log.Debugf("connection %x, dial %s", c.id, host)

	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		log.Warnf(err.Error())
		_ = c.Close()
		return
	}
	conn, err := net.Dial("tcp", tcpAddr.String())
	if err != nil {
		log.Warn("dial error:", err)
		_ = c.Close()
		return
	}

	err = transfer.Invoke(&dataPack{
		writer: c,
		reader: conn,
	})
	if err != nil {
		log.Warn(err)
		_ = conn.Close()
		_ = c.Close()
		return
	}

	err = transfer.Invoke(&dataPack{
		writer: conn,
		reader: c.pipeR,
	})
	if err != nil {
		log.Warn(err)
		_ = conn.Close()
		_ = c.Close()
		return
	}
}

func (server *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {

	hFunc, err := hashSelector(r.Header.Get("via"))
	if err != nil {
		log.Warnf("auth invalid from %s", r.RemoteAddr)
		http.NotFound(w, r)
		return
	}

	if !validateStringCode("authenticate", r.Header.Get("Auth"), hFunc) {
		log.Warnf("auth invalid from %s", r.RemoteAddr)
		http.NotFound(w, r)
		return
	}
	c, err := server.Resolver.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	ws := &webSocket{
		id: genRandBytes(wsAddrLen),
		hashFunc: hFunc,
		conn:     &wsConn{c},
		closed:   false,
		buf:      bytes.NewBuffer(make([]byte, 32*1024)),
	}
	wsPool.Store(u64(ws.id), ws)
	go wsHandler(ws)
}

func (server *Server) Listen() (err error) {

	mux := http.NewServeMux()
	if server.ListenAddr.Path == "" {
		server.ListenAddr.Path = "/"
	}
	mux.HandleFunc(server.ListenAddr.Path, server.HandleWebSocket)
	if server.Reverse != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			proxy := httputil.NewSingleHostReverseProxy(server.Reverse)
			proxy.ServeHTTP(w, r)
		})
	}

	s := http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         server.ListenAddr.Host,
		Handler:      mux,
	}

	log.Infof("Listening at %s", server.ListenAddr)
	if server.ListenAddr.Scheme == "ws" {
		err = s.ListenAndServe()
		if err != nil {
			return err
		}
		return
	} else {
		err = s.ListenAndServeTLS(server.Cert, server.PrivateKey)
		if err != nil {
			return err
		}
	}

	return
}

// GenRandBytes generates a random bytes slice in given length.
func genRandBytes(byteLength int) []byte {
	b := make([]byte, byteLength)
	_, err := rand.Read(b)
	if err != nil {
		return nil
	}
	return b
}
