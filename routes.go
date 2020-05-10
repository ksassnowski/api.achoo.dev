package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type invalidRequestResponse struct {
	Message string `json:"message"`
}

func (s *server) routes() {
	s.router.HandleFunc("/ping", s.handlePing()).Methods("GET")
	s.router.HandleFunc("/regions", s.handleGetRegions()).Methods("GET")
	s.router.HandleFunc("/subregions", s.handleGetSubregions()).Methods("GET")
	s.router.HandleFunc("/pollen", s.HandleGetAllReports()).Methods("GET")
	s.router.HandleFunc("/pollen/subregion/{subregion}", s.handleGetSubRegion()).Methods("GET")
	s.router.HandleFunc("/pollen/region/{region}", s.handleGetRegion()).Methods("GET")
}

func (s *server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Message string `json:"message"`
		}{"pong"}

		respond(w, http.StatusOK, data)
	}
}

func (s *server) HandleGetAllReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rs, err := s.storage.AllReports()
		if err != nil {
			log.Println(err)
			respond(w, http.StatusInternalServerError, nil)
			return
		}

		respond(w, http.StatusOK, rs)
	}
}

func (s *server) handleGetSubRegion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		subRegion := v["subregion"]

		data, err := s.storage.GetBySubregion(subRegion)
		if err != nil {
			if err == ErrNotFound {
				respond(w, http.StatusNotFound, &invalidRequestResponse{"No data found"})
				return
			}

			respond(w, http.StatusInternalServerError, nil)
			return
		}

		respond(w, http.StatusOK, data)
	}
}

func (s *server) handleGetRegions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rs, err := s.storage.AllRegions()
		if err != nil {
			respond(w, http.StatusInternalServerError, nil)
			return
		}

		respond(w, http.StatusOK, rs)
	}
}

func (s *server) handleGetSubregions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rs, err := s.storage.AllSubregions()
		if err != nil {
			respond(w, http.StatusInternalServerError, nil)
			return
		}

		respond(w, http.StatusOK, rs)
	}
}

func (s *server) handleGetRegion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reg := mux.Vars(r)["region"]

		rs, err := s.storage.GetByRegion(reg)
		if err != nil {
			if err == ErrNotFound {
				respond(w, http.StatusNotFound, &invalidRequestResponse{"No data found"})
				return
			}

			respond(w, http.StatusInternalServerError, nil)
			return
		}

		respond(w, http.StatusOK, rs)
	}
}

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if data == nil {
		w.WriteHeader(status)
		return
	}

	json, err := json.Marshal(data)
	if err != nil {
		log.Printf("[routes] unable to marshal response data: %q", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(json)
}
