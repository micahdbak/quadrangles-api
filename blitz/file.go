package blitz

import (
	"fmt"
	"net/http"
)

type FileHandler struct {
	Root string
}

func (f *FileHandler) Init(root string) {
	f.Root = root

	fmt.Printf("Initialized FileHandler;\n\tserving files in %q\n", root)
}

func (f *FileHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Path[7:] // get file name component of path

	for _, c := range file {
		// check if character isn't alphanumeric
		if (c < '0' || c > '9') &&
			(c < 'a' || c > 'z') &&
			(c < 'A' || c > 'Z') &&
			c != '.' {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	http.ServeFile(w, r, f.Root + file)
}
