package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	QUERY_EXT = "ext"
)

type Handler struct {
}

type UploadHandler struct {
	HostAddr  string
	UploadDir string
}

type ListHandler UploadHandler

type Employee struct {
	Name   string  `json:"name"`
	Age    int     `json:"age"`
	Salary float32 `json:"salary"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name := r.FormValue("name")
		fmt.Fprintf(w, "Parsed query-param with key \"name\": %s", name)
	case http.MethodPost:
		var employee Employee

		contentType := r.Header.Get("Content-Type")

		switch contentType {
		case "application/json":
			err := json.NewDecoder(r.Body).Decode(&employee)
			if err != nil {
				http.Error(w, "Unable to unmarshal JSON", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "Unknown content type", http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, "Got a new employee!\nName: %s\nAge: %dy.o.\nSalary %0.2f\n",
			employee.Name,
			employee.Age,
			employee.Salary,
		)
	}
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Unable to read file", http.StatusBadRequest)
		return
	}

	filePath := h.UploadDir + "/" + header.Filename

	//TODO: Вопрос с перезаписью файла
	if fileinfo, err := os.Stat(filePath); err == nil {
		name := fileNameWithoutExtension(fileinfo.Name())
		ext := filepath.Ext(filePath)
		i := 2
		for {
			tmpName := name + strconv.Itoa(i) + ext
			tmpPath := h.UploadDir + "/" + tmpName
			if _, err := os.Stat(tmpPath); errors.Is(err, os.ErrNotExist) {
				filePath = tmpPath
				header.Filename = tmpName
				break
			}
			i++
		}
	}

	err = ioutil.WriteFile(filePath, data, 0777)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	fileLink := h.HostAddr + "/" + header.Filename
	fmt.Fprintln(w, fileLink)

}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query()
		files, err := ioutil.ReadDir(h.UploadDir)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			if ext, ok := query[QUERY_EXT]; ok {
				if strings.HasSuffix(file.Name(), "."+ext[0]) {
					fmt.Fprintf(w, "<a href='%s/%s'>%s</a> %dBytes<br>\n", h.HostAddr, file.Name(), file.Name(), file.Size())
				}
			} else {
				fmt.Fprintf(w, "<a href='%s/%s'>%s</a> %dBytes<br>\n", h.HostAddr, file.Name(), file.Name(), file.Size())
			}

			//, file.IsDir()
		}
	}
}

func main() {
	handler := &Handler{}
	http.Handle("/", handler)

	uploadHandler := &UploadHandler{
		UploadDir: "upload",
		HostAddr:  "http://localhost:3002",
	}
	http.Handle("/upload", uploadHandler)

	listHandler := &ListHandler{
		UploadDir: "upload",
		HostAddr:  "http://localhost:3002",
	}
	http.Handle("/list", listHandler)

	go func() {
		srv := &http.Server{
			Addr:         ":3000",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		srv.ListenAndServe()
	}()

	dirToServe := http.Dir(uploadHandler.UploadDir)

	fs := &http.Server{
		Addr:         ":3002",
		Handler:      http.FileServer(dirToServe),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	fs.ListenAndServe()

}

func fileNameWithoutExtension(fileName string) string {
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}
