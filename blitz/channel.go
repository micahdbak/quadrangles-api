package blitz

import (
	"github.com/gorilla/websocket"
	"sync"
)

type Channel struct {
	Mutex sync.Mutex
	Conns []*websocket.Conn
}

func (c *Channel) Add(target *websocket.Conn) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	c.Conns = append(c.Conns, target)
}

func (c *Channel) Remove(target *websocket.Conn) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	for i, conn := range c.Conns {
		if conn == target {
			// swap this connection with the end of the list
			c.Conns[i] = c.Conns[len(c.Conns)-1]
			// remove last element in this array
			c.Conns = c.Conns[:len(c.Conns)-1]

			break
		}
	}
}

func (c *Channel) Message(message string) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	for _, conn := range c.Conns {
		// ignores possible error in writing message
		conn.WriteMessage(websocket.TextMessage, []byte(message))
	}
}
