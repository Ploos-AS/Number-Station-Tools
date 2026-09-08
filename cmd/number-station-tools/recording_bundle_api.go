package main

import (
	"net/http"
	"strconv"
	"strings"
)

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

func registerRecordingRestoreHandler(mux *http.ServeMux, db *store, rs *recordingStore, audioDir string) {
	receipts, receiptStoreErr := openRestoreReceiptStore(rs.path + ".restore-receipts.json")

	mux.HandleFunc("GET /api/recording-archive/receipts", func(w http.ResponseWriter, r *http.Request) {
		if receiptStoreErr != nil {
			http.Error(w, "restore receipt store is unavailable", http.StatusInternalServerError)
			return
		}
		limit := 50
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 || parsed > 200 {
				http.Error(w, "limit must be between 1 and 200", http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		writeJSON(w, http.StatusOK, receipts.list(limit))
	})

	mux.HandleFunc("GET /api/recording-archive/receipts/verify", func(w http.ResponseWriter, _ *http.Request) {
		if receiptStoreErr != nil {
			http.Error(w, "restore receipt store is unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, receipts.verify())
	})

	mux.HandleFunc("POST /api/recording-archive/plan", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxPortableArchiveUploadBytes)
		plan, err := planPortableRestore(r.Body, db, rs, audioDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, plan)
	})
	mux.HandleFunc("POST /api/recording-archive/import-selected", func(w http.ResponseWriter, r *http.Request) {
		if receiptStoreErr != nil {
			http.Error(w, "restore receipt store is unavailable", http.StatusInternalServerError)
			return
		}
		planToken := strings.TrimSpace(r.URL.Query().Get("plan_token"))
		if planToken == "" {
			http.Error(w, "plan_token is required; plan the archive before selective restore", http.StatusPreconditionRequired)
			return
		}
		selectedIDs := r.URL.Query()["recording_id"]
		r.Body = http.MaxBytesReader(w, r.Body, maxPortableArchiveUploadBytes)
		result, err := restorePortableArchiveSelectedWithPlanToken(r.Body, db, rs, audioDir, selectedIDs, planToken)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		receipt, err := receipts.add(restoreReceipt{
			Mode:                 "selected",
			PlanToken:            planToken,
			SelectedRecordingIDs: selectedIDs,
			ImportedRecordings:   result.ImportedRecords,
			ImportedAnnotations:  result.ImportedAnnotations,
			Duplicates:           result.Duplicates,
			Conflicts:            result.Conflicts,
			Unmatched:            result.Unmatched,
			SkippedByPolicy:      result.SkippedByPolicy,
		})
		if err != nil {
			http.Error(w, "restore succeeded but receipt persistence failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, selectiveRestoreResponse{portableSelectiveRestoreResult: result, Receipt: receipt})
	})
	mux.HandleFunc("POST /api/recording-archive/import", func(w http.ResponseWriter, r *http.Request) {
		if receiptStoreErr != nil {
			http.Error(w, "restore receipt store is unavailable", http.StatusInternalServerError)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxPortableArchiveUploadBytes)
		result, err := restorePortableArchive(r.Body, db, rs, audioDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		receipt, err := receipts.add(restoreReceipt{
			Mode:                "full",
			ImportedRecordings:  result.ImportedRecords,
			ImportedAnnotations: result.ImportedAnnotations,
			Duplicates:          result.Duplicates,
			Conflicts:           result.Conflicts,
			Unmatched:           result.Unmatched,
		})
		if err != nil {
			http.Error(w, "restore succeeded but receipt persistence failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, restoreResponse{portableRestoreResult: result, Receipt: receipt})
	})
}
