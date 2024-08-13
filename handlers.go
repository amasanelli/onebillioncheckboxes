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

type indexTemplateData struct {
	Email             string
	BuyMeACoffeeURL   string
	WebsocketURL      string
	TotalCheckboxes   uint32
	ReconnectInterval int
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "[method not allowed]", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path == "/" {
		if err := indexTemplate.Execute(w, indexTemplateData{Email: envData.EMAIL, BuyMeACoffeeURL: envData.BUY_ME_A_COFFEE_URL, WebsocketURL: envData.WEBSOCKET_URL, TotalCheckboxes: TOTAL_CHECKBOXES, ReconnectInterval: int(RECONNECT_INTERVAL.Milliseconds())}); err != nil {
			http.Error(w, "[internal server error]: error executing template", http.StatusInternalServerError)
			return
		}
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
	con, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[websockets error]: upgrade error: %s\n", err.Error())
		return
	}

	handler := newWebsocketHandler(con)
	handler.run()
}
