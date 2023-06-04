package blitz

import (
	"fmt"
	"net/http"
	"database/sql"
	"encoding/json"
)

type Post struct {
	PID  int     `json:"pid"`
	Text string  `json:"text"`
	kill bool
}

type PostHandler struct {
	Posts chan Post
	DB *sql.DB
}

func (p *PostHandler) Init(db *sql.DB) {
	p.Posts = make(chan Post)
	p.DB = db

	fmt.Print("Initialized PostHandler\n")
}

func (p *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	PID := r.URL.Path[7:]

	for _, c := range PID {
		if c < '0' || c > '9' {
			// invalid PID
			http.Error(w,
				http.StatusText(http.StatusBadRequest),
				http.StatusBadRequest)
			return
		}
	}

	rows, err := p.DB.Query(
		`SELECT pid, text FROM posts WHERE pid = $1`,
		PID,
	)

	if err != nil || !rows.Next() {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
	}

	var post Post
	rows.Scan(&post.PID, &post.Text)

	encoded, err := json.Marshal(post)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	w.Write(encoded)
}

func (p *PostHandler) ServePosts(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Path[7:]

	if len(topic) > 4 {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	for _, c := range topic {
		// check if topic name isn't alphanumeric
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			// invalid topic
			http.Error(w,
				http.StatusText(http.StatusBadRequest),
				http.StatusBadRequest)
			return
		}
	}

	rows, err := p.DB.Query(
		`SELECT pid, text FROM posts WHERE topic = $1`,
		topic,
	)

	if err != nil || !rows.Next() {
		w.Write([]byte("[]")) // write an empty array; no posts
		return
	}

	// begin JSON array of posts
	w.Write([]byte("[\n"))

	for {
		var post Post
		rows.Scan(&post.PID, &post.Text)

		encoded, err := json.Marshal(post)
		if err != nil {
			continue
		}

		w.Write(encoded)

		if rows.Next() {
			w.Write([]byte(",\n"))
		} else {
			break
		}
	}

	// end JSON array of posts
	w.Write([]byte("\n]\n"))
}

func (p *PostHandler) Factory() {
	for {
		post := <-p.Posts

		// break loop upon kill item
		if post.kill {
			break
		}


	}
}
