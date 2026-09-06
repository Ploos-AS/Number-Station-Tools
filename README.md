# Number Station Tools

Self-hosted, local-first tools for numbers stations, shortwave monitoring and
signal-analysis workflows.

## M0 foundation

M0 establishes the runnable project baseline:

- Go backend with embedded web UI
- `GET /healthz` health endpoint
- minimal multi-stage OCI image
- non-root runtime user
- persistent `/data` volume reserved for future SQLite, recordings and analysis data
- Docker Compose example
- amd64/arm64-friendly source layout
- MIT licensed

The application does not require an SDR, receiver, API key or external data
provider at M0.

## Run with Go

```sh
go run ./cmd/number-station-tools
```

Open <http://localhost:8080>.

## Run with Docker/Podman Compose

```sh
docker compose up --build
```

Then open <http://localhost:8080>.

## Health check

```sh
curl http://localhost:8080/healthz
```

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
