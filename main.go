package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/urfave/negroni"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	storage, err := NewEnvStorage()
	if err != nil {
		if err == ErrCouldNotConnectToStorage {
			log.Fatal("[main] unable to connect to configured storage")
		}
		log.Fatal(err.Error())
	}

	syncer := NewSyncer(storage, 1*time.Hour)
	go syncer.Run()

	server := &server{
		router:  mux.NewRouter(),
		storage: storage,
	}

	server.routes()
	n := negroni.Classic()
	n.Use(cors.Default())
	n.UseHandler(server)

	s := &http.Server{
		Addr:         ":8000",
		Handler:      n,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Can't be bothered to make TLS configurable. Just
	// use a reverse proxy for that...
	return s.ListenAndServe()
}
