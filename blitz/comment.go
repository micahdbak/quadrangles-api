package blitz

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/net/html"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Comment struct {
	PID  int
	CID  int
	Unix int64
	Text string
	kill bool
}

func (c *Comment) Json() []byte {
	return []byte(fmt.Sprintf(`{"cid": %d, "time": %d, "text": "%s"}`, c.CID, c.Unix, c.Text))
}

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

func (c *Channel) WriteComment(comment Comment) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	encoded := comment.Json()

	for _, conn := range c.Conns {
		// ignores possible error in writing message
		conn.WriteMessage(websocket.TextMessage, encoded)
	}
}

type WebSocketHandler struct {
	Channels map[int]*Channel
	Comments chan Comment
	DB       *sql.DB
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (ws *WebSocketHandler) Init(db *sql.DB) {
	ws.Channels = make(map[int]*Channel, 0)
	ws.Comments = make(chan Comment)
	ws.DB = db

	fmt.Print("Initialized WebSocketHandler\n")
}

func (ws *WebSocketHandler) CheckChannel(pid int) {
	c, ok := ws.Channels[pid]
	if !ok { // channel already doesn't exist
		return
	}

	c.Mutex.Lock()

	// check if this channel is empty
	if len(c.Conns) == 0 {
		c.Mutex.Unlock()
		delete(ws.Channels, pid) // close this channel
	}

	// channel isn't empty; do nothing

	c.Mutex.Unlock()
}

func (ws *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	PID, err := strconv.Atoi(r.URL.Path[8:])
	if err != nil || PID < 0 {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// check if post exists for this PID
	rows, err := ws.DB.Query(`SELECT * FROM posts WHERE pid = $1`, PID)
	if err != nil || !rows.Next() {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}
	rows.Close() // post exists

	// upgrade connection to websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// bad request message is already sent by Upgrade
		fmt.Printf("%v\n", err)
		return
	}

	c, ok := ws.Channels[PID]
	if !ok {
		c = new(Channel)
		ws.Channels[PID] = c
	}

	// add this connection to the channel
	c.Add(conn)
	defer c.Remove(conn)
	defer ws.CheckChannel(PID)

	fmt.Printf("WebSocketHandler.ServeHTTP: Started connection at /api/ws/%d\n", PID)

	rows, err = ws.DB.Query(
		`SELECT cid, time, text FROM comments WHERE pid = $1`,
		PID,
	)
	if err != nil {
		// no bother
	} else {
		// send all previously written comments
		for rows.Next() {
			var comment Comment

			rows.Scan(&comment.CID, &comment.Unix, &comment.Text)
			conn.WriteMessage(websocket.TextMessage, comment.Json())
		}
	}

	// maintain connection with client
	for {
		t, m, err := conn.ReadMessage()
		if err != nil || t == websocket.CloseMessage {
			fmt.Printf("WebSocketHandler.ServeHTTP: Stopped connection at /api/ws/%d\n", PID)
			break
		}

		var comment Comment

		comment.PID = PID
		comment.Unix = time.Now().Unix()
		comment.Text = html.EscapeString(string(m))

		ws.Comments <- comment
	}
}

func (ws *WebSocketHandler) Factory() {
	for {
		comment := <-ws.Comments

		// break loop upon kill item
		if comment.kill {
			break
		}

		c, ok := ws.Channels[comment.PID]
		if !ok {
			continue // channel doesn't exist anymore; ignore
		}

		// insert comment into database
		rows, err := ws.DB.Query(
			`INSERT INTO comments (pid, time, text)
				VALUES ($1, $2, $3)
				RETURNING cid`,
			comment.PID, comment.Unix, comment.Text,
		)
		if err != nil || !rows.Next() {
			continue // failed to write comment; ignore
		}

		rows.Scan(&comment.CID)
		rows.Close()

		// write comment to connections
		c.WriteComment(comment)
	}
}
