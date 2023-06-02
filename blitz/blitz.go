package blitz

import (
	//"sync"
	"net/http"
	"fmt"
	//"golang.org/x/net/html"
	//"github.com/gorilla/websocket"
)

/*
type Comment struct {
	PID int64
	Created int64
	Text string
}

type Post struct {
	Created int64
	Text string
}

type Channel struct {
	mutex sync.Mutex
	conn []*websocket.Conn
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
*/

type WebSocketHandler struct {
	//Comments chan *Comment
}

type PostHandler struct {
	//Posts chan *Post
}

func (ws *WebSocketHandler) Init() {
	fmt.Print("Initialized ws\n")
}

func (p *PostHandler) Init() {
	fmt.Print("Initialized p\n")
}

func (ws *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Access to WebSocketHandler from %q", r.RemoteAddr)
}

func (p *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Access to PostHandler from %q", r.RemoteAddr)
}

/*
func CommentFactory(comments []*Comment, c chan *Comment) {
	for {
		comment := <-c

		// break loop upon nil item
		if comment == nil {
			break
		}

		// add comment to comments
		comments = append(comments, comment)
	}
}

func PostFactory(posts []*Post, c chan *Post) {
	for {
		post := <-c

		// break loop upon nil item
		if post == nil {
			break
		}

		// add post to posts
		posts = append(posts, post)
	}
}
*/
