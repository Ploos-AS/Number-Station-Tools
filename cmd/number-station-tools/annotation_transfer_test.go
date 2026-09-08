package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnnotationTransferExportAndSafeImport(t *testing.T) {
	_, source := annotationTestStore(t)
	source.data.Recordings[0].SHA256 = strings.Repeat("a", 64)
	if _, err := source.addAnnotation("r1", recordingAnnotation{StartMS: 1000, Type: annotationTypeStationID, Label: "ID", Notes: "clear"}); err != nil {
		t.Fatal(err)
	}
	bundle := source.exportAnnotations()
	if bundle.Version != annotationTransferVersion || len(bundle.Items) != 1 {
		t.Fatalf("bundle=%#v", bundle)
	}
	if bundle.Items[0].Recording.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("missing recording hash: %#v", bundle.Items[0].Recording)
	}

	_, target := annotationTestStore(t)
	target.data.Recordings[0].ID = "different-id"
	target.data.Recordings[0].SHA256 = strings.Repeat("a", 64)
	result, err := target.importAnnotations(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.Duplicates != 0 || result.Unmatched != 0 {
		t.Fatalf("result=%#v", result)
	}
	if len(target.data.Annotations) != 1 || target.data.Annotations[0].RecordingID != "different-id" || target.data.Annotations[0].ID == bundle.Items[0].Annotation.ID {
		t.Fatalf("imported=%#v", target.data.Annotations)
	}

	result, err = target.importAnnotations(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 0 || result.Duplicates != 1 {
		t.Fatalf("duplicate result=%#v", result)
	}
}

func TestAnnotationTransferRejectsUnsafePathOnlyMatch(t *testing.T) {
	_, target := annotationTestStore(t)
	bundle := annotationTransferBundle{Version: annotationTransferVersion, Items: []annotationTransferItem{{
		Recording:  annotationTransferRecording{ID: "wrong-id", Path: target.data.Recordings[0].Path},
		Annotation: recordingAnnotation{StartMS: 100, Type: annotationTypeOther, Label: "bookmark"},
	}}}
	result, err := target.importAnnotations(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if result.Unmatched != 1 || result.Imported != 0 || len(target.data.Annotations) != 0 {
		t.Fatalf("unsafe import result=%#v annotations=%#v", result, target.data.Annotations)
	}
}

func TestAnnotationTransferAPI(t *testing.T) {
	_, rs := annotationTestStore(t)
	if _, err := rs.addAnnotation("r1", recordingAnnotation{StartMS: 500, Type: annotationTypeTone, Label: "Tone"}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerAnnotationHandlers(mux, rs)

	req := httptest.NewRequest(http.MethodGet, "/api/annotations/export?format=csv", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "recording_id,recording_path") || !strings.Contains(w.Header().Get("Content-Disposition"), ".csv") {
		t.Fatalf("csv status=%d headers=%v body=%s", w.Code, w.Header(), w.Body.String())
	}

	bundle := rs.exportAnnotations()
	bundle.Items = nil
	payload, _ := json.Marshal(bundle)
	req = httptest.NewRequest(http.MethodPost, "/api/annotations/import", strings.NewReader(string(payload)))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("import status=%d body=%s", w.Code, w.Body.String())
	}
}
