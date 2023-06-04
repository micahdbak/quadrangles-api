package main

import (
	"os"
	"log"
	"fmt"
	"net/http"
	"goblitz/blitz"
	"database/sql"
	_ "github.com/lib/pq"
)

func main() {
	var (
		f blitz.FileHandler
		ws blitz.WebSocketHandler
		p blitz.PostHandler
	)

	db, err := sql.Open("postgres", "dbname=goblitz sslmode=disable")
	if err != nil {
		fmt.Printf("%v\nCouldn't connect to database.\n", err)
		return
	}

	f.Init(os.Getenv("GOBLITZF"), 2 << 20, 10, db)
	ws.Init(db)
	p.Init(db)

	go f.Factory()
	go ws.Factory()
	go p.Factory()

	http.HandleFunc("/api/f/", f.ServeFile)
	http.Handle("/api/file", &f)
	http.Handle("/api/ws/", &ws)
	http.Handle("/api/p/", &p)
	http.HandleFunc("/api/t/", p.ServePosts)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
