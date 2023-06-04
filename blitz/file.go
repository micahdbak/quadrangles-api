package blitz

import (
	"os"
	"io"
	"fmt"
	"time"
	"strconv"
	"net/http"
	"database/sql"
	"mime/multipart"
)

type File struct {
	FormFile multipart.File
	Header   *multipart.FileHeader
}

type FileHandler struct {
	Root string
	MaxFileSize int64
	QueueSize int
	DB *sql.DB
	Files chan File
}

func (f *FileHandler) Init(root string, maxFileSize int64, queueSize int, db *sql.DB) {
	f.Root = root

	// ensure that root is a directory
	if f.Root[len(f.Root) - 1] != '/' {
		f.Root = f.Root + "/"
	}

	f.MaxFileSize = maxFileSize
	f.QueueSize = queueSize
	f.DB = db

	f.Files = make(chan File, queueSize)

	fmt.Printf("Initialized FileHandler;\n\tserving files in %q\n", root)
}

func (f *FileHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[7:] // get file name component of path

	i := 0
	for path[i] != '.' {
		i++
	}

	fid, err := strconv.Atoi(path[:i])
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	reqCtype := path[i+1:]
	rows, err := f.DB.Query("SELECT ctype, name FROM files WHERE fid = $1", fid)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}

	var ctype, name string

	rows.Next()
	rows.Scan(&ctype, &name)
	rows.Close()

	if ctype != reqCtype {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}

	file, err := os.Open(f.Root + path)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}
	defer file.Close()

	// set content type
	w.Header().Set("Content-Type", "image/" + ctype)
	http.ServeContent(w, r, name, time.Now(), file)
}

func (f *FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ensure that client is posting
	if r.Method != "POST" {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// get file posted
	r.ParseMultipartForm(f.MaxFileSize)
	formFile, header, err := r.FormFile("file")
	if err != nil {
		fmt.Printf("FileHandler.ServeHTTP: %v\n", err)
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	ctype := header.Header["Content-Type"][0]

	// ensure content type is an image
	if len(ctype) < 6 || ctype[:5] != "image" {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// check file size
	if header.Size > f.MaxFileSize {
		http.Error(w,
			http.StatusText(http.StatusRequestEntityTooLarge),
			http.StatusRequestEntityTooLarge)
		formFile.Close()
		return
	}

	// check if queue is full
	if len(f.Files) >= f.QueueSize {
		http.Error(w,
			http.StatusText(http.StatusServiceUnavailable),
			http.StatusServiceUnavailable)
		formFile.Close()
		return
	}

	// prepare file for factory

	var file File

	file.FormFile = formFile
	file.Header = header

	// enqueue this file
	f.Files <- file
}

func (f *FileHandler) Factory() {
	for {
		file := <-f.Files
		ctype := file.Header.Header["Content-Type"][0]

		// insert this file into the database
		rows, err := f.DB.Query(
			"INSERT INTO files (ctype, name) VALUES ($1, $2) RETURNING fid",
			ctype[6:], file.Header.Filename)
		if err != nil {
			fmt.Printf("FileHandler.Factory: %v\n", err)
			file.FormFile.Close()
			continue
		}

		var fid int

		rows.Next()
		rows.Scan(&fid) // get fid of inserted file
		rows.Close()

		// open destination file
		dest, err := os.Create(f.Root + strconv.Itoa(fid) + "." + ctype[6:])
		if err != nil {
			fmt.Printf("FileHandler.Factory: %v\n", err)
			file.FormFile.Close()
			continue
		}

		// copy form file to destination file
		if _, err := io.Copy(dest, file.FormFile); err != nil {
			fmt.Printf("FileHandler.Factory: %v\n", err)
		}

		fmt.Printf("FileHandler.Factory: Inserted fid %d ctype %s\n", fid, ctype)

		// close files
		file.FormFile.Close()
		dest.Close()
	}
}
