package main

import (
	"assignment-concurrent-system/internal/jobs"
	"assignment-concurrent-system/internal/requests"
	"log"
	"net/http"
	"time"
)

func main() {

	baseMgr := jobs.NewBaseJobManager(5 * time.Minute)

	apiMgr := jobs.NewAPIJobManager(baseMgr, 5*time.Minute)

	mux := requests.NewRouter(apiMgr)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server started on :8080")
	log.Fatal(server.ListenAndServe())
}
