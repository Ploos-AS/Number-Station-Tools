package main

import "net/http"

func registerRecordingBundleHandlers(mux *http.ServeMux, rs *recordingStore, audioDir string) {
	mux.HandleFunc("GET /api/recording-bundle", func(w http.ResponseWriter, _ *http.Request) {
		body, err := marshalRecordingBundle(rs.exportRecordingBundle())
		if err != nil {
			http.Error(w, "could not encode recording bundle", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="number-station-recording-bundle.json"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})

	mux.HandleFunc("GET /api/recording-archive", func(w http.ResponseWriter, _ *http.Request) {
		plan, err := preparePortableArchive(rs, audioDir)
		if err != nil {
			http.Error(w, "recording archive integrity preflight failed", http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="number-station-archive.tar.gz"`)
		w.WriteHeader(http.StatusOK)
		_, _ = writePreparedPortableArchive(w, plan)
	})
}
