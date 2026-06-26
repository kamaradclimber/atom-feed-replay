# atom-feed-replay

Takes an Atom feed and exposes a new Atom feed that replays entries with smoothed publication dates — useful for e.g. drip-releasing a YouTube back-catalog at a steady pace.

## Why

YouTube (and many platforms) only expose recent entries in their feeds. If you want to publish an entire back-catalog to subscribers at a consistent cadence rather than all at once, this toolchain bridges the gap:

1. **yt-feed-builder** — uses the YouTube Data API v3 to enumerate every video in a channel/playlist and serves a complete Atom feed over HTTP.
2. **replay-server** — consumes that feed, computes smoothed publication dates, and serves a new Atom feed where entries appear at a controlled rate until caught up to real-time.

## How to use

### Local

```bash
# Terminal 1 — build the full history feed
go run ./cmd/yt-feed-builder/ config.yaml

# Terminal 2 — replay with smoothing
go run . config.yaml
```

Point your feed reader at the replay-server endpoint (default `http://localhost:8080/feeds/<id>`).

See `config.yaml.example` for configuration.

### Container

Two images are published to `ghcr.io/kamaradclimber/atom-feed-replay/`:

| Image | Description |
|-------|-------------|
| `replay-server` | Atom feed replay with date smoothing |
| `yt-feed-builder` | Full-history Atom feed via YouTube Data API v3 |

Mount your config file at a known path and pass it as the command:

```bash
docker run -v ./config.yaml:/etc/afr/config.yaml:ro \
  ghcr.io/kamaradclimber/atom-feed-replay/replay-server \
  /etc/afr/config.yaml
```

Or via docker-compose:

```bash
docker compose up -d
```

Edit `docker-compose.yml` to point at your config files.

### CI

On every push to `main` or a `v*` tag, GitHub Actions builds and pushes both images to GHCR. Pull requests are built but not pushed.

## Disclaimer

This project is **vibe coded** — it was built iteratively through an AI chat session without rigorous human review of every line. Use at your own risk, especially in production.
