package blitz

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Post struct {
	PID   int    `json:"pid"`
	File  string `json:"file"`
	Topic string `json:"topic"`
	Text  string `json:"text"`
	Unix  int64  `json:"time"`
	kill  bool
}

type PostHandler struct {
	Files string
	DB    *sql.DB
}

func (p *PostHandler) Init(files string, db *sql.DB) {
	p.Files = files
	p.DB = db

	fmt.Print("Initialized PostHandler\n")
}

func validTopic(topic string) bool {
	for _, c := range topic {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}

	return true
}

func (p *PostHandler) ServePost(w http.ResponseWriter, r *http.Request) {
	PID, err := strconv.Atoi(r.URL.Path[7:])

	// pid must be numerical
	if err != nil || PID < 1 {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// attempt to find post
	rows, err := p.DB.Query(
		`SELECT files.fid, files.ctype, posts.topic, posts.text, posts.time
			FROM posts JOIN files ON posts.fid = files.fid
			WHERE posts.pid = $1`,
		PID,
	)

	if err != nil || !rows.Next() {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}

	var (
		fid   int
		ctype string
		topic string
		text  string
		unix  int64
		post  Post
	)

	rows.Scan(&fid, &ctype, &topic, &text, &unix)
	rows.Close()

	post.PID = PID
	post.File = p.Files + strconv.Itoa(fid) + "." + ctype
	post.Topic = topic
	post.Text = text
	post.Unix = unix

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

	// validate topic
	if !validTopic(topic) {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// read posts related to this topic, with their file information
	rows, err := p.DB.Query(
		`SELECT posts.pid, files.fid, files.ctype, posts.topic, posts.text, posts.time
			FROM posts JOIN files ON posts.fid = files.fid
			WHERE posts.topic = $1`,
		topic,
	)

	if err != nil || !rows.Next() {
		fmt.Printf("PostHandler.ServePosts: %v\n", err)
		w.Write([]byte("[]")) // write an empty array; no posts
		return
	}
	defer rows.Close()

	// begin JSON array of posts
	w.Write([]byte("[\n"))

	for {
		var (
			pid   int
			fid   int
			ctype string
			topic string
			text  string
			unix  int64
			post  Post
		)

		rows.Scan(&pid, &fid, &ctype, &topic, &text, &unix)

		post.PID = pid
		post.File = p.Files + strconv.Itoa(fid) + "." + ctype
		post.Topic = topic
		post.Text = text
		post.Unix = unix

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

/* POST /api/post
 * "fid": file ID (requires that a file was uploaded for this post)
 * "topic": topic that this post is related to
 * "text": text content of post */
func (p *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	var (
		pid   int
		topic string
		text  string
	)

	// read and validate fid
	fid, err := strconv.Atoi(r.FormValue("fid"))
	if err != nil || fid < 1 {
		http.Error(w, "Invalid fid.", http.StatusBadRequest)
		return
	}

	rows, err := p.DB.Query("SELECT fid FROM files WHERE fid = $1", fid)
	if err != nil || !rows.Next() {
		http.Error(w, "Unknown fid.", http.StatusBadRequest)
		return
	}
	rows.Close() // fid is valid; no need to read row

	// read and validate topic
	if topic = r.FormValue("topic"); topic == "" || len(topic) > 4 || !validTopic(topic) {
		http.Error(w, "Invalid topic.", http.StatusBadRequest)
		return
	}

	// read and validate text
	if text = r.FormValue("text"); text == "" || len([]byte(text)) > 2000 {
		http.Error(w, "Invalid text.", http.StatusBadRequest)
		return
	}

	rows, err = p.DB.Query(
		`INSERT INTO posts (fid, topic, text, time)
			VALUES ($1, $2, $3, $4)
			RETURNING pid`,
		fid, topic, text, time.Now().Unix(),
	)
	if err != nil || !rows.Next() {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	rows.Scan(&pid)
	rows.Close()

	w.Write([]byte(strconv.Itoa(pid)))
}
