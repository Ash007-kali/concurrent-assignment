package main

import (
	"assignment-concurrent-system/internal/jobs"
	"assignment-concurrent-system/internal/requests"
	"log"
	"net/http"
	"time"
)

func main() {

	baseMgr := &jobs.BaseJobManager{
		Jobs: make(map[string]*jobs.BaseJob),
		TTL:  5 * time.Minute, // cache base jobs for 5mins
	}

	apiMgr := &jobs.APIJobManager{
		Jobs:    make(map[string]*jobs.APIJob),
		BaseMgr: baseMgr,
		TTL:     5 * time.Minute, // cache API jobs for 5mins
	}

	mux := requests.NewRouter(apiMgr)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server started on :8080")
	log.Fatal(server.ListenAndServe())
}
