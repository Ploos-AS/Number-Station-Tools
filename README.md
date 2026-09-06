# Number Station Tools

Self-hosted, local-first tools for numbers stations, shortwave monitoring and
signal-analysis workflows.

## Current milestone: M2.1

The application currently provides:

- local station catalogue
- structured observation logbook
- persistent UTC schedules and frequencies
- station/schedule web forms
- server-side `on now / next` schedule evaluation in UTC
- correct handling of schedules that cross midnight
- local JSON persistence under `/data`
- embedded web UI and JSON API
- dependency-free Go build and non-root OCI runtime

The application does not require an SDR, receiver, API key or external data
provider for these workflows.

See [docs/M2_1.md](docs/M2_1.md) for the current scope.

## Run with Go

For a host run, select a writable data path:

```sh
NUMBER_STATION_TOOLS_DATA=./number-station-tools.json \
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

## Direction

Number Station Tools is intended to grow into a workbench for:

- station identifiers, aliases and historical metadata
- UTC transmission schedules and frequencies
- structured observation logging
- message and group transcription
- WAV/FLAC recording metadata
- spectrogram and signal-analysis workflows
- optional SDR and network-receiver integrations
- optional data-provider imports
- local-first archival and search

Core functionality should remain useful without API keys, and receiver,
recording and observation data should remain local by default.

## License

MIT. See [LICENSE](LICENSE).
