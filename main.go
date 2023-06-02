package main

import (
	"log"
	"time"
	"net/http"
	"goblitz/blitz"
)

func main() {
	s := &http.Server{
		Addr:		":8000",
		Handler:	&blitz.Handler{},
		ReadTimeout:	5 * time.Second,
		WriteTimeout:	5 * time.Second,
		MaxHeaderBytes:	1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
