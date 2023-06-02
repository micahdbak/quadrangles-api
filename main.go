package main

import (
	"log"
	//"time"
	"net/http"
	"goblitz/blitz"
)

func main() {
	var (
		ws blitz.WebSocketHandler
		p blitz.PostHandler
	)

	ws.Init()
	p.Init()

	http.Handle("/api/ws/", &ws)
	http.Handle("/api/p/", &p)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
