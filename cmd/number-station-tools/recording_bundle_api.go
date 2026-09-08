package main

import "net/http"

func registerRecordingBundleHandlers(mux *http.ServeMux, rs *recordingStore) {
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
}
