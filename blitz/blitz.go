package blitz

import (
	"net/http"
	"fmt"
	"strconv"
	"github.com/gorilla/websocket"
)

type Comment struct {
	PID int
	Text string
}

type Post struct {
	Text string
}

type WebSocketHandler struct {
	Channels map[int]*Channel
	Comments chan Comment
	Store []Comment
}

type PostHandler struct {
	Posts chan Post
	Store []Post
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (ws *WebSocketHandler) Init() {
	ws.Channels = make(map[int]*Channel, 0)
	ws.Comments = make(chan Comment)

	fmt.Print("Initialized WebSocketHandler\n")
}

func (p *PostHandler) Init() {
	p.Posts = make(chan Post)

	fmt.Print("Initialized PostHandler\n")
}

func (ws *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	PID, err := strconv.Atoi(r.URL.Path[8:])
	if err != nil || PID < 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// check if post at PID exists

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// bad request message is already sent by Upgrade
		fmt.Printf("%v\n", err)
		return
	}

	fmt.Print("Got connection!\n")

	c, ok := ws.Channels[PID]

	if !ok {
		c = new(Channel)
		ws.Channels[PID] = c
	}

	// add this connection to the channel
	c.Add(conn)
	defer c.Remove(conn)

	for {
		t, m, err := conn.ReadMessage()
		if err != nil || t == websocket.CloseMessage {
			break
		}

		var comment Comment

		comment.PID = PID
		comment.Text = string(m)

		ws.Comments <- comment
	}
}

func (p *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Access to PostHandler from %q", r.RemoteAddr)
}

func (ws *WebSocketHandler) Factory() {
	for {
		comment := <-ws.Comments

		// break loop upon nil item
		if comment.Text == "" {
			break
		}

		ws.Channels[comment.PID].Message(comment.Text)

		// add comment to store
		//ws.Store = append(ws.Store, *comment)
	}
}

func (p *PostHandler) Factory() {
	for {
		post := <-p.Posts

		// break loop upon nil item
		if post.Text == "" {
			break
		}

		// add post to posts
		p.Store = append(p.Store, post)
	}
}
