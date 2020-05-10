package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func createServer() *server {
	s := &server{}
	s.router = mux.NewRouter()
	s.routes()
	return s
}

func TestHealthEndpoint(t *testing.T) {
	s := httptest.NewServer(createServer())
	defer s.Close()

	res, err := http.Get(s.URL + "/ping")
	if err != nil {
		t.Errorf("got error: %q", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("wanted status 200, got %d", res.StatusCode)
	}

	defer res.Body.Close()
	want := `{"message":"pong"}`
	got, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("got error: %q", err)
	}

	if want != string(got) {
		t.Errorf("want %q, got %q", want, got)
	}
}
