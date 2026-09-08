# Number Station Tools

Self-hosted, local-first tools for numbers stations, shortwave monitoring and
signal-analysis workflows.

## Current milestone: M5.6

The application currently provides:

- local station catalogue
- structured observation logbook
- persistent UTC schedules and frequencies
- server-side `on now / next` schedule evaluation in UTC
- schedule-to-observation logging
- recording metadata linked to observations
- managed local WAV/FLAC upload under `/data/audio`
- automatic size, SHA-256, duration, sample rate and channel metadata
- compact waveform previews for managed PCM WAV recordings
- compact 64-bin frequency magnitude previews for managed PCM WAV recordings
- compact 24×32 time-frequency mini-spectrograms for managed PCM WAV recordings
- compact local signal fingerprints and deterministic recording similarity search
- local fingerprint clustering with configurable similarity threshold
- exact file-duplicate flagging from matching SHA-256 values inside clusters
- persistent local cluster review/classification with notes and UTC review time
- membership-stable cluster identifiers so reviews only attach to the reviewed member set
- inline local playback for managed recordings with HTTP byte-range seeking
- synchronized playback cursor across waveform and mini-spectrogram previews
- click/drag seeking directly on waveform and mini-spectrogram previews
- keyboard seeking on focused previews with arrow, Home and End keys
- temporary A-B loop regions with repeated playback and highlighted selection
- persistent recording point bookmarks and timestamp-interval annotations
- visual point/interval annotation markers directly on waveform and spectrogram previews
- edit existing annotations while preserving their stable annotation IDs
- direct Bookmark creation by right-clicking a waveform/spectrogram position
- keyboard Bookmark creation with `B` on a focused seek preview
- fixed annotation categories: `call-up`, `station ID`, `message`, `tone`, `noise`, `fade`, `other`
- per-recording annotation filtering by category with category-specific marker styling
- cross-recording annotation search by type, label and notes with direct jump-to-recording/timestamp
- versioned JSON annotation export/import with conservative recording matching and duplicate suppression
- flat CSV annotation export for archival and offline analysis
- versioned metadata-only recording/annotation manifest export with checksums and provenance
- full portable `tar.gz` archival export containing `manifest.json` plus verified managed WAV/FLAC files
- safe restore/import of M5.0 archives with staged extraction, checksum validation and collision protection
- dry-run restore planning with per-recording `import`, `duplicate`, `conflict` and `unmatched` classification
- selective restore of operator-chosen `import` candidates from a validated plan
- deterministic SHA-256 plan tokens binding selective restore to the reviewed archive and local state
- strict rejection of stale plan tokens after archive, recording, annotation or observation changes
- strict rejection of selected duplicates, conflicts, unmatched observations and unknown recording IDs
- persistent local restore receipts for successful full and selective restores
- SHA-256 receipt hash chaining with `previous_hash` and `receipt_hash`
- read-only receipt-chain verification with tamper detection and chain head reporting
- read-only restore audit history with mode, UTC time, plan token, selection and result counts
- per-recording annotation import/skip counts before restore
- local rebuild of waveform, frequency, spectrogram and fingerprint data from restored managed audio
- local JSON persistence under `/data`
- embedded web UI and JSON API
- dependency-free Go build and non-root OCI runtime

Cluster reviews support `same transmission`, `same station`, `false positive`,
and `duplicate capture`. They are stored locally alongside recording metadata and
do not require AI, an API key, or an external service.

Managed playback reads audio directly from the appliance. Browser-native codec
support determines whether a WAV/FLAC recording can be played; M5.6 does not
transcode audio or send it to an external service. Saved timestamp annotations
persist in `recordings.json`, can be searched across the archive, exported as JSON
or CSV, and safely imported from the versioned JSON format.

M5.6 makes the M5.5 restore audit trail tamper-evident. Every newly persisted
receipt includes the SHA-256 of its canonical content and the hash of the preceding
receipt. Appending a receipt first verifies the complete existing chain and refuses
to extend a broken log. `GET /api/recording-archive/receipts/verify` reports the
chain status, receipt count and current head hash, or identifies the first failing
receipt. Existing M5.5 receipt files that contain no hashes are sealed once when
first opened by M5.6; this establishes integrity from that migration point onward
but cannot retroactively prove that legacy data was untouched before migration.

The application does not require an SDR, receiver, API key or external data
provider for these workflows.

See [docs/M5_6.md](docs/M5_6.md) for the current scope.

## Run with Go

For a host run, select writable data paths:

```sh
NUMBER_STATION_TOOLS_DATA=./number-station-tools.json \
NUMBER_STATION_TOOLS_RECORDINGS_DATA=./recordings.json \
NUMBER_STATION_TOOLS_AUDIO_DIR=./audio \
  go run ./cmd/number-station-tools
```

Open <http://localhost:8080>.

## Run with Docker/Podman Compose

```sh
docker compose up --build
```

Then open <http://localhost:8080>.

## Useful API endpoints

- `GET /healthz`
- `GET/POST /api/stations`
- `GET/POST /api/observations`
- `GET/POST /api/schedules`
- `GET /api/now-next`
- `GET/POST /api/recordings`
- `GET /api/recording-bundle`
- `GET /api/recording-archive`
- `GET /api/recording-archive/receipts?limit=50`
- `GET /api/recording-archive/receipts/verify`
- `POST /api/recording-archive/plan`
- `POST /api/recording-archive/import-selected?recording_id=...&plan_token=...`
- `POST /api/recording-archive/import`
- `GET /api/recordings/{id}/similar`
- `GET /api/recording-clusters?threshold=98`
- `PUT /api/recording-clusters/{id}/review?threshold=98`
- `DELETE /api/recording-clusters/{id}/review`
- `GET /api/observations/{id}/recordings`
- `GET /api/annotations?q=&type=&limit=`
- `GET /api/annotations/export?format=json`
- `GET /api/annotations/export?format=csv`
- `POST /api/annotations/import`
- `GET/POST /api/recordings/{id}/annotations`
- `PUT/DELETE /api/recordings/{id}/annotations/{annotationID}`
- `POST /api/audio`
- `GET /api/recordings/{id}/file`
- `GET /api/recordings/{id}/stream`

## Direction

Number Station Tools is intended to grow into a workbench for:

- station identifiers, aliases and historical metadata
- UTC transmission schedules and frequencies
- structured observation logging
- message and group transcription
- WAV/FLAC recording archive and interactive playback
- waveform, frequency, spectrogram, fingerprint and signal-analysis workflows
- categorized timestamp annotations, bookmarks and operator notes on recordings
- cross-recording annotation search, portable transfer and timeline navigation
- portable recording manifests, verified archival bundles, token-bound dry-run planning, policy-controlled restore and tamper-evident local audit receipts
- human review and classification of signal matches and repeated transmissions
- optional SDR and network-receiver integrations
- optional data-provider imports
- local-first archival and search

Core functionality should remain useful without API keys, and receiver,
recording and observation data should remain local by default.

## License

MIT. See [LICENSE](LICENSE).
