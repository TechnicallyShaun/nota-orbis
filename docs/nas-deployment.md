# NAS Deployment Guide (TerraMaster T4 Max)

This guide describes a practical setup for running Nota Orbis workflows on a
TerraMaster T4 Max NAS. It focuses on:
- Docker installation on TOS
- Deploying `whisper-asr-webservice`
- Syncthing configuration for audio ingestion
- Network considerations and troubleshooting

## Prerequisites

- TerraMaster T4 Max running a recent TOS release
- Admin access to TOS
- Local network with stable IP addressing
- Basic familiarity with Docker and Syncthing

## 1) TerraMaster T4 Max Initial Setup

1. Install drives and complete TOS onboarding.
2. Create a storage pool and volume (RAID if desired).
3. Create shared folders for:
   - `vault` (your Obsidian/Nota vault)
   - `droppoint` (incoming audio)
   - `compose` (docker-compose files)
4. Create a non-admin user for daily access and grant access to those shares.
5. Set the system timezone and enable time sync (NTP).

Tip: keep `/share/compose` for Docker stacks and `/share/droppoint` for inbound
Syncthing files so paths are consistent across services.

## 2) Install Docker on TOS

1. Open **TOS App Center**.
2. Install **Docker Manager** (or the official Docker package if listed).
3. Enable SSH in TOS (Control Panel -> Terminal/SSH) for easier management.
4. Verify Docker from SSH:

```bash
ssh <nas-user>@<nas-ip>
docker --version
docker compose version
```

If `docker compose` is missing, install the plugin via your package manager or
use the Docker Manager UI to run compose stacks.

## 3) Deploy whisper-asr-webservice

A sample compose file should be present at:

```
compose/whisper-asr/docker-compose.yml
```

### Steps

1. Copy the compose directory to your NAS share, e.g. `/share/compose/whisper-asr`.
2. From SSH on the NAS, start the container:

```bash
cd /share/compose/whisper-asr

docker compose pull

docker compose up -d
```

3. Verify the service:

- Swagger UI: `http://<nas-ip>:9000/docs`
- Health check (basic): open the docs page in a browser

### Recommended settings

The compose file uses `onerahmet/openai-whisper-asr-webservice` with env vars:
- `ASR_ENGINE=faster_whisper`
- `ASR_MODEL=base`

For CPU-only NAS use, recommended model sizes:
- `tiny` (fastest, lowest accuracy)
- `base` (balanced default)
- `small` (slower, better accuracy)

Avoid `medium`/`large` on CPU-only NAS hardware unless you accept long runtimes.

### Storage

Ensure the compose file includes a persistent cache volume (model downloads):
- `whisper-cache:/root/.cache`

This prevents re-downloading models on container restart.

## 4) Syncthing Configuration Tips

### Folder layout

- **Phone**: send-only folder (e.g. `/VoiceNotes`)
- **NAS**: receive-only folder at `/share/droppoint/voice-notes`

This avoids deletions on the NAS when you clean up files on the phone.

### Permissions

Make sure the NAS user that runs Syncthing can read/write the droppoint folder.

### Ignore rules

Create a `.stignore` in the droppoint folder to reduce noise:

```
# macOS
.DS_Store
# iOS temp files
*.tmp
# Partial/in-progress uploads
.~lock.*
```

### Versioning

Enable simple file versioning on the NAS if you want a safety net for accidental
overwrites or sync conflicts.

## 5) Network Considerations

- **Static IP**: Use a DHCP reservation for the NAS to keep IP stable.
- **LAN-only access**: Do not expose port 9000 to the public internet.
- **VPN for remote**: If remote access is needed, use WireGuard/Tailscale.
- **DNS**: Consider a local hostname (e.g. `nas.local`) for stable endpoints.

Port overview:
- `9000` (whisper-asr-webservice)
- Syncthing ports (default 8384 UI / 22000 sync / 21027 discovery)

## 6) Troubleshooting

### Container won't start

```bash
docker compose ps
docker compose logs -f
```

Common causes:
- Bad compose syntax
- Port 9000 already in use
- Incorrect environment variables

### Slow transcription

- Use smaller model sizes (`base` or `tiny`)
- Keep the NAS well-ventilated and avoid other heavy workloads

### Syncthing conflicts

- Ensure NAS folder is set to **Receive Only**
- Enable versioning to recover overwritten files

### Permission errors

- Confirm Syncthing and Docker paths map to folders owned by the same user
- Verify folder permissions under `/share/*`

## 7) Quick Checklist

- [ ] TOS storage pool + volume created
- [ ] Shared folders: `vault`, `droppoint`, `compose`
- [ ] Docker Manager installed and SSH enabled
- [ ] whisper-asr-webservice running on port 9000
- [ ] Syncthing syncing to `/share/droppoint/voice-notes`
- [ ] NAS uses a stable IP / DNS name
