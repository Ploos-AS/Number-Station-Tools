package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
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
	mux.HandleFunc("GET /api/annotations/export", func(w http.ResponseWriter, r *http.Request) {
		bundle := rs.exportAnnotations()
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" || format == "json" {
			payload, err := marshalAnnotationBundle(bundle)
			if err != nil {
				http.Error(w, "cannot export annotations", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="number-station-annotations.json"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(payload)
			return
		}
		if format == "csv" {
			payload, err := encodeAnnotationCSV(bundle)
			if err != nil {
				http.Error(w, "cannot export annotations", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="number-station-annotations.csv"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(payload)
			return
		}
		http.Error(w, "format must be json or csv", http.StatusBadRequest)
	})
	mux.HandleFunc("POST /api/annotations/import", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
		var bundle annotationTransferBundle
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&bundle); err != nil {
			http.Error(w, "invalid annotation import", http.StatusBadRequest)
			return
		}
		result, err := rs.importAnnotations(bundle)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, result)
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
