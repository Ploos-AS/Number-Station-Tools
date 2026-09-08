# Number Station Tools

Self-hosted, local-first tools for numbers stations, shortwave monitoring and
signal-analysis workflows.

## Current milestone: M5.8

The application currently provides a local station catalogue, structured observations and UTC schedules; managed WAV/FLAC recording ingest and playback; waveform, frequency, spectrogram and fingerprint analysis; clustering and human review; timestamp annotations with search and transfer; portable metadata manifests and verified `tar.gz` archives; safe dry-run, selective and full restore; token-bound restore planning; and tamper-evident restore receipts.

M5.7 added deterministic external receipt-chain anchors. M5.8 adds a separate local registry of anchor export requests so the operator can see when the chain was last anchored and how many receipts have accumulated since then. The registry stores only anchor metadata (ID, UTC export time, receipt count and head hash), never archive or audio payloads.

`GET /api/recording-archive/receipts/anchors/status` reports `never-anchored`, `current`, `extended`, or `mismatch`, plus `receipts_since_last_anchor`. `GET /api/recording-archive/receipts/anchors?limit=50` returns anchor-export history newest first. A successful anchor download request is registered automatically.

The local registry is operational metadata, not an independent trust anchor. M5.7's security benefit still requires keeping the downloaded anchor outside the appliance trust boundary. Registration proves that the server generated an export response; it cannot prove that the client safely retained the file.

Core workflows remain local-first and require no SDR, API key or external provider.

See [docs/M5_8.md](docs/M5_8.md) for the current scope.

## Run with Go

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
- `GET /api/recording-archive/receipts/anchor`
- `GET /api/recording-archive/receipts/anchors?limit=50`
- `GET /api/recording-archive/receipts/anchors/status`
- `POST /api/recording-archive/receipts/anchor/verify`
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

Number Station Tools is intended to grow into a local-first workbench for station metadata, schedules, observation logging, message transcription, recording archival/playback, signal analysis, annotations, portable verified archives, policy-controlled restore, and independently anchorable tamper-evident audit history. Optional SDR/network-receiver and data-provider integrations may be added later without making them requirements for core use.

## License

MIT. See [LICENSE](LICENSE).
