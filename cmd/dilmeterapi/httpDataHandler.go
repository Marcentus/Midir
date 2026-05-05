// httpHandlerData.go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func dataRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/{file}", handleGetDataFile)
	return r
}

func handleGetDataFile(w http.ResponseWriter, r *http.Request) {
	fileName := chi.URLParam(r, "file")
	// Sanitize file name to prevent directory traversal
	if fileName != "skills.json" && fileName != "races.json" && fileName != "conditions.json" && fileName != "talents.json" && fileName != "items.json" && fileName != "overrides.json" {
		http.NotFound(w, r)
		return
	}

	// Read the file from our embedded filesystem
	data, err := staticData.ReadFile("static_data/" + fileName)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
