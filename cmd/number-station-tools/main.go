package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var webFS embed.FS

type healthResponse struct { Status string `json:"status"`; Time string `json:"time"` }
type station struct { ID string `json:"id"`; Name string `json:"name"`; Aliases []string `json:"aliases,omitempty"`; Languages []string `json:"languages,omitempty"`; Notes string `json:"notes,omitempty"` }
type observation struct { ID string `json:"id"`; StationID string `json:"station_id"`; HeardAt time.Time `json:"heard_at"`; FrequencyHz int64 `json:"frequency_hz"`; Mode string `json:"mode,omitempty"`; Signal string `json:"signal,omitempty"`; Message string `json:"message,omitempty"`; Notes string `json:"notes,omitempty"` }
type dataFile struct { Stations []station `json:"stations"`; Observations []observation `json:"observations"` }
type store struct { mu sync.Mutex; path string; data dataFile }

func openStore(path string) (*store, error) {
	s := &store{path:path, data:dataFile{Stations:[]station{}, Observations:[]observation{}}}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) { return s, nil }
	if err != nil { return nil, err }
	if err := json.Unmarshal(b, &s.data); err != nil { return nil, err }
	return s, nil
}
func (s *store) save() error { b,e:=json.MarshalIndent(s.data,"","  "); if e!=nil{return e}; tmp:=s.path+".tmp"; if e=os.WriteFile(tmp,b,0600);e!=nil{return e}; return os.Rename(tmp,s.path) }
func (s *store) addStation(v station) error { s.mu.Lock(); defer s.mu.Unlock(); s.data.Stations=append(s.data.Stations,v); return s.save() }
func (s *store) addObservation(v observation) error { s.mu.Lock(); defer s.mu.Unlock(); s.data.Observations=append(s.data.Observations,v); return s.save() }
func id() string { return time.Now().UTC().Format("20060102T150405.000000000") }
func writeJSON(w http.ResponseWriter, code int, v any) { w.Header().Set("Content-Type","application/json"); w.WriteHeader(code); _=json.NewEncoder(w).Encode(v) }

func main() {
	addr := getenv("NUMBER_STATION_TOOLS_ADDR", ":8080")
	path := getenv("NUMBER_STATION_TOOLS_DATA", "/data/number-station-tools.json")
	db, err := openStore(path); if err != nil { log.Fatal(err) }
	staticFS, err := fs.Sub(webFS, "web"); if err != nil { log.Fatal(err) }
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w,200,healthResponse{"ok",time.Now().UTC().Format(time.RFC3339)}) })
	mux.HandleFunc("GET /api/stations", func(w http.ResponseWriter, _ *http.Request) { db.mu.Lock(); defer db.mu.Unlock(); out:=append([]station(nil),db.data.Stations...); sort.Slice(out,func(i,j int)bool{return out[i].Name<out[j].Name}); writeJSON(w,200,out) })
	mux.HandleFunc("POST /api/stations", func(w http.ResponseWriter,r *http.Request) { var v station; if json.NewDecoder(r.Body).Decode(&v)!=nil||strings.TrimSpace(v.Name)=="" { http.Error(w,"invalid station",400);return }; v.ID=id(); if db.addStation(v)!=nil { http.Error(w,"store error",500);return }; writeJSON(w,201,v) })
	mux.HandleFunc("GET /api/observations", func(w http.ResponseWriter,_ *http.Request) { db.mu.Lock(); defer db.mu.Unlock(); writeJSON(w,200,db.data.Observations) })
	mux.HandleFunc("POST /api/observations", func(w http.ResponseWriter,r *http.Request) { var v observation; if json.NewDecoder(r.Body).Decode(&v)!=nil||v.StationID==""||v.FrequencyHz<=0 { http.Error(w,"invalid observation",400);return }; if v.HeardAt.IsZero(){v.HeardAt=time.Now().UTC()}; v.ID=id(); if db.addObservation(v)!=nil { http.Error(w,"store error",500);return }; writeJSON(w,201,v) })
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	log.Printf("Number Station Tools listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil { log.Fatal(err) }
}
func getenv(key, fallback string) string { if value:=os.Getenv(key);value!="" {return value};return fallback }
