from pathlib import Path

p = Path("cmd/number-station-tools/main.go")
s = p.read_text()

old = '''\taddr := getenv("NUMBER_STATION_TOOLS_ADDR", ":8080")
\tpath := getenv("NUMBER_STATION_TOOLS_DATA", "/data/number-station-tools.json")
\tdb, err := openStore(path)
\tif err != nil {
\t\tlog.Fatal(err)
\t}
'''
new = '''\taddr := getenv("NUMBER_STATION_TOOLS_ADDR", ":8080")
\tpath := getenv("NUMBER_STATION_TOOLS_DATA", "/data/number-station-tools.json")
\trecordingPath := getenv("NUMBER_STATION_TOOLS_RECORDINGS", "/data/recordings.json")
\tdb, err := openStore(path)
\tif err != nil {
\t\tlog.Fatal(err)
\t}
\trs, err := openRecordingStore(recordingPath)
\tif err != nil {
\t\tlog.Fatal(err)
\t}
'''
if old not in s:
    raise SystemExit("main store anchor not found")
s = s.replace(old, new, 1)

old = '''\tmux := http.NewServeMux()
\tmux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
'''
new = '''\tmux := http.NewServeMux()
\tregisterRecordingHandlers(mux, db, rs)
\tmux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
'''
if old not in s:
    raise SystemExit("mux anchor not found")
s = s.replace(old, new, 1)

old = '''\tmux.HandleFunc("DELETE /api/observations/{id}", func(w http.ResponseWriter, r *http.Request) {
\t\tif err := db.deleteObservation(r.PathValue("id")); err != nil {
\t\t\thttp.Error(w, err.Error(), 404)
\t\t\treturn
\t\t}
\t\tw.WriteHeader(http.StatusNoContent)
\t})
'''
new = '''\tmux.HandleFunc("DELETE /api/observations/{id}", func(w http.ResponseWriter, r *http.Request) {
\t\tobservationID := r.PathValue("id")
\t\tif rs.observationReferenced(observationID) {
\t\t\thttp.Error(w, "observation has recordings", http.StatusConflict)
\t\t\treturn
\t\t}
\t\tif err := db.deleteObservation(observationID); err != nil {
\t\t\thttp.Error(w, err.Error(), http.StatusNotFound)
\t\t\treturn
\t\t}
\t\tw.WriteHeader(http.StatusNoContent)
\t})
'''
if old not in s:
    raise SystemExit("observation delete anchor not found")
s = s.replace(old, new, 1)

p.write_text(s)
