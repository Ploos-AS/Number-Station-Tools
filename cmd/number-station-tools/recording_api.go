package main

import "net/http"

func registerRecordingHandlers(mux *http.ServeMux, db *store, rs *recordingStore) {
	mux.HandleFunc("GET /api/recordings", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, rs.list())
	})
	mux.HandleFunc("GET /api/observations/{id}/recordings", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, rs.listForObservation(r.PathValue("id")))
	})
	mux.HandleFunc("POST /api/recordings", func(w http.ResponseWriter, r *http.Request) {
		var v recording
		if decodeJSON(r, &v) != nil {
			http.Error(w, "invalid recording", http.StatusBadRequest)
			return
		}
		v.ID = id()
		if err := rs.add(db, v); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, v)
	})
	mux.HandleFunc("PUT /api/recordings/{id}", func(w http.ResponseWriter, r *http.Request) {
		var v recording
		if decodeJSON(r, &v) != nil {
			http.Error(w, "invalid recording", http.StatusBadRequest)
			return
		}
		if err := rs.update(db, r.PathValue("id"), v); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		v.ID = r.PathValue("id")
		writeJSON(w, http.StatusOK, v)
	})
	mux.HandleFunc("DELETE /api/recordings/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := rs.delete(r.PathValue("id")); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
