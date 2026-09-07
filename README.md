# Number Station Tools

Self-hosted, local-first tools for numbers stations, shortwave monitoring and
signal-analysis workflows.

## Current milestone: M3.5

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
- local JSON persistence under `/data`
- embedded web UI and JSON API
- dependency-free Go build and non-root OCI runtime

The application does not require an SDR, receiver, API key or external data
provider for these workflows.

See [docs/M3_5.md](docs/M3_5.md) for the current scope.

## Run with Go

For a host run, select writable data paths:

```sh
NUMBER_STATION_TOOLS_DATA=./number-station-tools.json \
NUMBER_STATION_TOOLS_RECORDINGS=./recordings.json \
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
- `GET /api/observations/{id}/recordings`
- `POST /api/audio`
- `GET /api/recordings/{id}/file`

## Direction

Number Station Tools is intended to grow into a workbench for:

- station identifiers, aliases and historical metadata
- UTC transmission schedules and frequencies
- structured observation logging
- message and group transcription
- WAV/FLAC recording archive
- waveform, frequency, spectrogram and signal-analysis workflows
- optional SDR and network-receiver integrations
- optional data-provider imports
- local-first archival and search

Core functionality should remain useful without API keys, and receiver,
recording and observation data should remain local by default.

## License

MIT. See [LICENSE](LICENSE).
