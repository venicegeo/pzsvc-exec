package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/gorilla/mux"
)

// This is a quick script to mock up a Piazza server that accepts uploads and gives status on a job

func main() {
	rand.Seed(time.Now().Unix())
	router := mux.NewRouter()

	router.Methods("POST").Path("/data").HandlerFunc(mockFilePostHandler)
	router.Methods("POST").Path("/data/file").HandlerFunc(mockFilePostHandler)
	router.Methods("GET").Path("/job/mock-job-id-123-abc").HandlerFunc(mockJobGetHandler)

	router.NotFoundHandler = verboseNotFoundHandler{}

	http.ListenAndServe(":8080", router)
}

func mockFilePostHandler(w http.ResponseWriter, r *http.Request) {
	content := `{
    "data": { "jobId": "mock-job-id-123-abc" }
  }`
	logRequest(r, []byte(content))
	w.Write([]byte(content))
}

func mockJobGetHandler(w http.ResponseWriter, r *http.Request) {

	content := `{
    "data": {
      "jobId": "mock-job-id-123-abc",
      "createdBy": "someone",
      "jobType": "job-type-abc",
      "status": "%s",
      "result": {}
    }
  }`

	if rand.Float64() < 0.3 {
		content = fmt.Sprintf(content, "Success")
	} else {
		content = fmt.Sprintf(content, "Running")
	}
	logRequest(r, []byte(content))
	w.Write([]byte(content))
}

type verboseNotFoundHandler struct{}

func (h verboseNotFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logRequest(r, []byte("404"))
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404"))
}

func logRequest(r *http.Request, responseData []byte) {
	log.Printf("LOGGING REQUEST")
	body, err := httputil.DumpRequest(r, true)
	log.Print(string(body), err)
	log.Print("RESPONSE BODY:")
	log.Print(string(responseData))
}
