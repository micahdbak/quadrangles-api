package blitz

import (
	"fmt"
	"net/http"
	"golang.org/x/net/html"
)

type Handler struct {
	requests int
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.requests += 1
	fmt.Fprintf(w, "Hello, %s!\nNumber of requests is %d.",
		html.EscapeString(r.RemoteAddr),
		h.requests)
}
