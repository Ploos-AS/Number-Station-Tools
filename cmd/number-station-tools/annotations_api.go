package main

import (
	"net/http"
	"strconv"
)

func registerAnnotationHandlers(mux *http.ServeMux, rs *recordingStore) {
	mux.HandleFunc("GET /api/annotations", func(w http.ResponseWriter, r *http.Request) {
		limit := 50
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 || parsed > 200 {
				http.Error(w, "limit must be between 1 and 200", http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		hits, err := rs.searchAnnotations(r.URL.Query().Get("q"), r.URL.Query().Get("type"), limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, hits)
	})
	mux.HandleFunc("GET /api/recordings/{id}/annotations", func(w http.ResponseWriter, r *http.Request) {
		annotations, err := rs.listAnnotations(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, annotations)
	})
	mux.HandleFunc("POST /api/recordings/{id}/annotations", func(w http.ResponseWriter, r *http.Request) {
		var annotation recordingAnnotation
		if decodeJSON(r, &annotation) != nil {
			http.Error(w, "invalid recording annotation", http.StatusBadRequest)
			return
		}
		created, err := rs.addAnnotation(r.PathValue("id"), annotation)
		if err != nil {
			status := http.StatusBadRequest
			if err.Error() == "recording does not exist" {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	})
	mux.HandleFunc("PUT /api/recordings/{id}/annotations/{annotationID}", func(w http.ResponseWriter, r *http.Request) {
		var annotation recordingAnnotation
		if decodeJSON(r, &annotation) != nil {
			http.Error(w, "invalid recording annotation", http.StatusBadRequest)
			return
		}
		updated, err := rs.updateAnnotation(r.PathValue("id"), r.PathValue("annotationID"), annotation)
		if err != nil {
			status := http.StatusBadRequest
			if err.Error() == "recording does not exist" || err.Error() == "annotation does not exist" {
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	})
	mux.HandleFunc("DELETE /api/recordings/{id}/annotations/{annotationID}", func(w http.ResponseWriter, r *http.Request) {
		if err := rs.deleteAnnotation(r.PathValue("id"), r.PathValue("annotationID")); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
