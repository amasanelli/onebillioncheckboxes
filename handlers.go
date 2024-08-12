package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "[method not allowed]", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "[method not allowed]", http.StatusMethodNotAllowed)
		return
	}

	if _, err := os.Stat(filepath.Join("./public", r.URL.Path)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "[not found]", http.StatusNotFound)
			return
		}
		http.Error(w, "[internal server error]", http.StatusInternalServerError)
		return
	}

	http.FileServer(http.Dir("public")).ServeHTTP(w, r)
}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	connection, err := upg.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[websockets error]: upgrade error: %s\n", err.Error())
		return
	}

	connectionHandler := newConnectionHandler(connection)
	connectionHandler.run()
}
