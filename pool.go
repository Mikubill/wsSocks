package main

import (
	"encoding/hex"
	cmap "github.com/orcaman/concurrent-map"
	"math/rand"
)

type nPool struct {
	cmap.ConcurrentMap
	keys []string
}

func newPool() *nPool {
	return &nPool{
		cmap.New(),
		[]string{},
	}
}

func (c *nPool) getWs() *webSocket {
	for {
		if len(c.keys) == 0 {
			continue
		} else {
			if item, ok := c.Get(c.keys[rand.Intn(len(c.keys))]); ok {
				return item.(*webSocket)
			}
		}
	}

}

func (c *nPool) addWs(ws *webSocket) {
	if ws.id == "" {
		ws.id = hex.EncodeToString(genRandBytes(6))
	}
	c.Set(ws.id, ws)
	c.keys = c.Keys()
}

func (c *nPool) addConn(conn *muxConn) {
	if conn.id == nil {
		conn.id = genRandBytes(3)
	}
	c.Set(string(conn.id), conn)
}

func (c *nPool) RemoveWs(id string) {
	c.ConcurrentMap.Remove(id)
	c.keys = c.Keys()
}
