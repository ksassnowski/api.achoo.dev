package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

type server struct {
	router  *mux.Router
	storage Storage
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
