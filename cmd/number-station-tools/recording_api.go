package main

import (
	"net/http"
	"strconv"
)

func registerRecordingHandlers(mux *http.ServeMux, db *store, rs *recordingStore) {
	registerAudioHandlers(mux, db, rs, getenv("NUMBER_STATION_TOOLS_AUDIO_DIR", "/data/audio"))
	registerAnnotationHandlers(mux, rs)

	mux.HandleFunc("GET /api/recordings", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, rs.list())
	})
	mux.HandleFunc("GET /api/recording-clusters", func(w http.ResponseWriter, r *http.Request) {
		threshold, _ := strconv.ParseFloat(r.URL.Query().Get("threshold"), 64)
		writeJSON(w, http.StatusOK, rs.clusters(threshold))
	})
	mux.HandleFunc("PUT /api/recording-clusters/{id}/review", func(w http.ResponseWriter, r *http.Request) {
		threshold, _ := strconv.ParseFloat(r.URL.Query().Get("threshold"), 64)
		clusterID := r.PathValue("id")
		var cluster *recordingCluster
		for _, candidate := range rs.clusters(threshold) {
			if candidate.ID == clusterID {
				copy := candidate
				cluster = &copy
				break
			}
		}
		if cluster == nil {
			http.Error(w, "recording cluster does not exist", http.StatusNotFound)
			return
		}
		var request struct {
			Classification string `json:"classification"`
			Notes          string `json:"notes"`
		}
		if decodeJSON(r, &request) != nil {
			http.Error(w, "invalid cluster review", http.StatusBadRequest)
			return
		}
		review, err := rs.setClusterReview(*cluster, request.Classification, request.Notes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, review)
	})
	mux.HandleFunc("DELETE /api/recording-clusters/{id}/review", func(w http.ResponseWriter, r *http.Request) {
		if err := rs.deleteClusterReview(r.PathValue("id")); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/recordings/{id}/similar", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		results, err := rs.similar(r.PathValue("id"), limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, results)
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
		v.Managed = false
		v.OriginalName = ""
		v.Fingerprint = nil
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
		current, ok := rs.byID(r.PathValue("id"))
		if !ok {
			http.Error(w, "recording does not exist", http.StatusNotFound)
			return
		}
		v.Managed = current.Managed
		v.OriginalName = current.OriginalName
		v.Fingerprint = current.Fingerprint
		if current.Managed {
			v.Path = current.Path
			v.Format = current.Format
			v.SizeBytes = current.SizeBytes
			v.SHA256 = current.SHA256
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
