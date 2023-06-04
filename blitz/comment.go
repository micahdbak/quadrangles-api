package blitz

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strconv"
	"sync"
)

type Comment struct {
	PID  int
	Text string
	kill bool
}

type WebSocketHandler struct {
	Channels map[int]*Channel
	Comments chan Comment
	Mutex    sync.Mutex
	Store    []Comment
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

func (ws *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	PID, err := strconv.Atoi(r.URL.Path[8:])
	if err != nil || PID < 0 {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// check if post at PID exists

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

	fmt.Printf("Opened socket on PID: %d\n", PID)

	comments := ws.Match(PID)

	// send all previous comments to connection
	for _, comment := range comments {
		conn.WriteMessage(websocket.TextMessage, []byte(comment))
	}

	for {
		t, m, err := conn.ReadMessage()
		if err != nil || t == websocket.CloseMessage {
			fmt.Printf("Closed socket on PID: %d\n", PID)
			break
		}

		var comment Comment

		comment.PID = PID
		comment.Text = string(m)

		ws.Comments <- comment
	}
}

func (ws *WebSocketHandler) Append(comment Comment) {
	ws.Mutex.Lock()
	defer ws.Mutex.Unlock()

	ws.Store = append(ws.Store, comment)
}

func (ws *WebSocketHandler) Match(PID int) []string {
	ws.Mutex.Lock()
	defer ws.Mutex.Unlock()

	comments := make([]string, 0)

	// find all comments that match this PID
	// NOTE: SLOW! this will be replaced with a database command
	for _, comment := range ws.Store {
		if comment.PID == PID {
			comments = append(comments, comment.Text)
		}
	}

	return comments
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
			continue
		}

		c.Message(comment.Text)

		// temporary: add comment to store
		ws.Append(comment)

		// TODO: insert comment into database
	}
}
