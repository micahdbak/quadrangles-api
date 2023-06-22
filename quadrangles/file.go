package quadrangles

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type File struct {
	FID   int            // fid of this file in the database
	Ctype string         // content type of file
	File  multipart.File // reader that will be used to write file to disk
}

type FileHandler struct {
	Root        string        // directory to read and write files
	MaxFileSize int64         // maximum permitted file size
	QueueSize   int           // maximum number of files to store in queue
	Speed       time.Duration // minimum period of time between files in the queue
	Last        time.Time     // time of last file written
	Mutex       sync.Mutex    // used to ensure a file's place in the queue
	Files       chan File     // file queue
	DB          *sql.DB       // database to store file information
}

func (f *FileHandler) Init(root string, maxFileSize int64, queueSize int, speed time.Duration, db *sql.DB) {
	f.Root = root

	// ensure that root is a directory
	if f.Root[len(f.Root)-1] != '/' {
		f.Root = f.Root + "/"
	}

	f.MaxFileSize = maxFileSize
	f.QueueSize = queueSize
	f.Speed = speed
	f.Last = time.Now()
	f.DB = db

	f.Files = make(chan File, queueSize)

	fmt.Printf("Initialized FileHandler;\n\tserving files in %q\n", root)
}

// /api/f/<fid>.<ctype>
/*
func (f *FileHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[7:] // get file name component of path

	// get numerical component (fid) of path
	i := 0
	for i < len(path) && path[i] != '.' {
		i++
	}

	// if i is the length of the path, then there is no file extension
	if i == len(path) {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// attempt to read fid
	fid, err := strconv.Atoi(path[:i])
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// get provided file extension; check with database entry
	reqCtype := path[i+1:]
	rows, err := f.DB.Query("SELECT ctype, name, time FROM files WHERE fid = $1", fid)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}

	var ctype, name string
	var unix int64

	rows.Next()
	rows.Scan(&ctype, &name, &unix)
	rows.Close()

	if ctype != reqCtype {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}

	// attempt to open file
	file, err := os.Open(f.Root + path)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
		return
	}
	defer file.Close()

	// set content type
	w.Header().Set("Content-Type", "image/"+ctype)
	http.ServeContent(w, r, name, time.Unix(unix, 0), file)
}
*/

func (f *FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ensure that client is posting
	if r.Method != "POST" {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	// parse multipart form
	r.ParseMultipartForm(f.MaxFileSize)
	formFile, header, err := r.FormFile("file")
	if err != nil {
		fmt.Printf("FileHandler.ServeHTTP: %v\n", err)
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		return
	}

	var ctype string

	/* read content type; this will be in the form
	 * "image/<png/jpeg/...>" if it is an image */
	if ctype_s, ok := header.Header["Content-Type"]; !ok || len(ctype_s) == 0 {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		formFile.Close()
		return
	} else {
		ctype = ctype_s[0]
	}

	/* ensure content type is an image;
	 * len(ctype) will be >6 if ctype starts with "image/";
	 * len[:5] will equal "image" if content is an image. */
	if len(ctype) < 6 || ctype[:5] != "image" {
		http.Error(w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest)
		formFile.Close()
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

	// ensure that no other files can take this file's place in the queue
	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	// check if queue is full
	if len(f.Files) >= f.QueueSize {
		http.Error(w,
			http.StatusText(http.StatusServiceUnavailable),
			http.StatusServiceUnavailable)
		formFile.Close()
		return
	}

	// insert this file into the database
	rows, err := f.DB.Query(
		`INSERT INTO files (ctype, name, time)
			VALUES ($1, $2, $3)
			RETURNING fid`,
		ctype[6:], header.Filename, time.Now().Unix(),
	)
	if err != nil || !rows.Next() {
		fmt.Printf("FileHandler.ServeHTTP: %v\n", err)
		http.Error(w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	var (
		fid  int
		file File
	)

	rows.Scan(&fid)
	rows.Close()

	file.FID = fid
	file.Ctype = ctype[6:]
	file.File = formFile

	f.Files <- file

	w.Write([]byte(strconv.Itoa(fid)))
}

func (f *FileHandler) Factory() {
	for {
		now := time.Now()
		elapsed := now.Sub(f.Last)

		if elapsed < f.Speed {
			/* wait the remaining period of time
			 * before moving onto the next file */
			time.Sleep(f.Speed - elapsed)
		}

		f.Last = time.Now()
		file := <-f.Files

		// open destination file
		dest, err := os.Create(f.Root + strconv.Itoa(file.FID) + "." + file.Ctype)
		if err != nil {
			fmt.Printf("FileHandler.Factory: %v\n", err)
			file.File.Close()
			continue
		}

		// copy form file to destination file
		if _, err := io.Copy(dest, file.File); err != nil {
			fmt.Printf("FileHandler.Factory: %v\n", err)
		} else {
			fmt.Printf("FileHandler.Factory: Inserted fid %d ctype %s\n", file.FID, file.Ctype)
		}

		// close files
		file.File.Close()
		dest.Close()
	}
}
