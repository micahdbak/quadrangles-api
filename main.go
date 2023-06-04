package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"goblitz/blitz"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	var (
		f  blitz.FileHandler
		ws blitz.WebSocketHandler
		p  blitz.PostHandler
	)

	db, err := sql.Open("postgres", "dbname=goblitz sslmode=disable")
	if err != nil {
		fmt.Printf("%v\nCouldn't connect to database.\n", err)
		return
	}

	root, ok := os.LookupEnv("GOBLITZF")
	if !ok {
		fmt.Print("The GOBLITZF environment variable must be set to determine where files are stored.\n")
		return
	}

	/* root:        upload files into $GOBLITZF,
	 * 2<<20:       maximum file size of 2MB,
	 * 10:          maximum of 10 files in queue at a time
	 * time.Second: maximum of one file written per second
	 * db:          PostgreSQL to store file information */
	f.Init(root, 2<<20, 10, time.Second, db)
	ws.Init(db)
	p.Init("/api/f/", db)

	go f.Factory()
	go ws.Factory()

	http.HandleFunc("/api/f/", f.ServeFile)
	http.HandleFunc("/api/p/", p.ServePost)
	http.HandleFunc("/api/t/", p.ServePosts)

	http.Handle("/api/file", &f)
	http.Handle("/api/ws/", &ws)
	http.Handle("/api/post", &p)

	/* IMPORTANT: This is for testing only.
	 * Comment this line out for production. */
	http.Handle("/", http.FileServer(http.Dir("./test/")))

	log.Fatal(http.ListenAndServe(":8000", nil))
}
