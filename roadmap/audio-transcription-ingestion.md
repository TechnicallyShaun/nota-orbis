# Audio Transcription Ingestion

```yaml
status: plan
created: 2026-01-22
author: shaun
feature: audio-transcription-ingestion
```

---

## Overview

Automated pipeline for transcribing voice recordings and ingesting them into a Nota vault.

### User Story

> As a user, I record voice notes on my phone. These sync via Syncthing to a droppoint folder on my server. I want these audio files automatically transcribed and saved as markdown notes in my vault inbox, with the original audio archived.

### Flow Diagram

```
[Phone Voice Recorder]
        ↓ (M4A)
    [Syncthing]
        ↓
[Droppoint Folder] ←── [File Watcher (inotify)]
        ↓
[Wait for file stability]
        ↓
[Send to Transcription API]
        ↓
[Receive text response]
        ↓
[Apply template (optional)]
        ↓
[Save to Vault Inbox]
        ↓
[Archive original M4A to /nota/archive/audio]
        ↓
[Delete from droppoint]
```

---

## Commands

### `nota transcribe config`

Interactive configuration for transcription service.

**Standard Mode:**
```bash
$ nota transcribe config

Transcription Service Configuration
===================================

Watch folder [required]: /mnt/sync/voice-notes
Transcription API URL [required]: http://nas:9000/asr
Output location (inbox) [required]: /home/user/vault/Inbox
Template file [optional, Enter to skip]:
Audio archive location [default: ~/.nota/archive/audio]:

Configuration saved to /home/user/vault/.nota/transcribe.json
```

**Advanced Mode:**
```bash
$ nota transcribe config --advanced

# Includes standard params plus:
Stabilization interval [default: 2s]:
Stabilization checks [default: 3]:
Language [default: auto]:
Model [default: base]:
Max file size [default: 100MB]:
Retry count [default: 3]:
Watch patterns [default: *.m4a,*.mp3,*.wav]:
```

### `nota transcribe start`

Start the file watcher service.

```bash
$ nota transcribe start           # Foreground mode
$ nota transcribe start --daemon  # Background with systemd
```

### `nota transcribe stop`

Stop the file watcher service.

```bash
$ nota transcribe stop
```

### `nota transcribe status`

Show service status and recent activity.

```bash
$ nota transcribe status
Status: running (pid 12345)
Watching: /mnt/sync/voice-notes
Last processed: 2026-01-22T14:30:00Z (meeting-notes.m4a)
Files processed today: 7
Errors today: 0
```

---

## Configuration

### Storage Location

Configuration stored at: `{vault}/.nota/transcribe.json`

### Schema

```json
{
  "watch_dir": "/mnt/sync/voice-notes",
  "api_url": "http://nas:9000/asr",
  "output_dir": "/home/user/vault/Inbox",
  "template_path": null,
  "archive_dir": "~/.nota/archive/audio",
  "watch_patterns": ["*.m4a", "*.mp3", "*.wav"],
  "stabilization_interval_ms": 2000,
  "stabilization_checks": 3,
  "language": "auto",
  "model": "base",
  "max_file_size_mb": 100,
  "retry_count": 3
}
```

### Parameters Reference

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `watch_dir` | Yes | - | Folder to watch for incoming audio files |
| `api_url` | Yes | - | Transcription service endpoint |
| `output_dir` | Yes | - | Vault inbox location for transcribed notes |
| `template_path` | No | `null` | Optional markdown template to append to |
| `archive_dir` | No | `~/.nota/archive/audio` | Where to archive processed M4A files |
| `watch_patterns` | No | `["*.m4a", "*.mp3", "*.wav"]` | File patterns to process |
| `stabilization_interval_ms` | No | `2000` | Ms between file size checks |
| `stabilization_checks` | No | `3` | Consecutive stable checks required |
| `language` | No | `auto` | Whisper language hint |
| `model` | No | `base` | Whisper model size |
| `max_file_size_mb` | No | `100` | Reject files larger than this |
| `retry_count` | No | `3` | API retry attempts before failure |

---

## Output Format

### Filename Convention

```
YYYY-MM-DD-HHmm-voice-note.md
```

- DateTime extracted from audio file metadata
- Falls back to current time if metadata unavailable
- Collision handling: `-2`, `-3` suffix if file exists

**Examples:**
```
2026-01-22-1430-voice-note.md
2026-01-22-1430-voice-note-2.md  (if first exists)
```

### Without Template

Plain markdown with transcription only:

```markdown
[Transcribed text from audio file...]
```

### With Template

Appends transcription to template file content:

```markdown
---
type: voice-note
date: 2026-01-22
source: meeting-notes.m4a
---

# Voice Note

[Transcribed text from audio file...]
```

---

## Error Handling

### On Failure

1. Original file remains in droppoint (not deleted)
2. Error logged to `~/.nota/logs/transcribe-YYYY-MM-DD.log`
3. Error note created in vault inbox:

```markdown
---
type: error
source: transcribe-service
timestamp: 2026-01-22T14:30:00Z
---

# Transcription Failed

**File:** meeting-notes.m4a
**Error:** Connection refused to transcription service
**Log:** See ~/.nota/logs/transcribe-2026-01-22.log

Original file remains at: /mnt/sync/voice-notes/meeting-notes.m4a
```

### Retry Behavior

- Retries `retry_count` times with exponential backoff
- After all retries exhausted, creates error note
- File left in droppoint for manual intervention

---

## System Paths

| Path | Purpose |
|------|---------|
| `~/.nota/logs/` | Service logs |
| `~/.nota/archive/audio/` | Archived original audio files |
| `{vault}/.nota/transcribe.json` | Per-vault configuration |

---

## Logging

### Log Location

`~/.nota/logs/transcribe-YYYY-MM-DD.log`

### Log Format

Structured logging with levels:

```
2026-01-22T14:30:00Z INFO  [watcher] File detected: meeting-notes.m4a
2026-01-22T14:30:02Z INFO  [watcher] File stable, processing: meeting-notes.m4a
2026-01-22T14:30:02Z INFO  [transcribe] Sending to API: meeting-notes.m4a (2.4MB)
2026-01-22T14:30:15Z INFO  [transcribe] Received response: 1247 chars
2026-01-22T14:30:15Z INFO  [output] Saved: 2026-01-22-1430-voice-note.md
2026-01-22T14:30:15Z INFO  [archive] Archived: /nota/archive/audio/meeting-notes.m4a
2026-01-22T14:30:15Z INFO  [cleanup] Deleted from droppoint: meeting-notes.m4a
```

### Log Rotation

Daily rotation with 30-day retention (configurable).

---

## Architecture

### Interface Design

```go
// FileWatcher detects new files in a directory
type FileWatcher interface {
    Watch(ctx context.Context, dir string, patterns []string) (<-chan FileEvent, error)
    Stop() error
}

// FileEvent represents a detected file
type FileEvent struct {
    Path      string
    Size      int64
    Timestamp time.Time
}

// Stabilizer waits for a file to finish writing
type Stabilizer interface {
    WaitForStable(ctx context.Context, path string) error
}

// TranscriptionClient sends audio and receives text
type TranscriptionClient interface {
    Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscriptionResult, error)
}

// TranscribeOptions configures the transcription request
type TranscribeOptions struct {
    Language string
    Model    string
}

// TranscriptionResult contains the API response
type TranscriptionResult struct {
    Text     string
    Language string
    Duration float64
}

// OutputWriter saves transcriptions to the vault
type OutputWriter interface {
    Write(ctx context.Context, text string, opts OutputOptions) (string, error)
}

// OutputOptions configures output writing
type OutputOptions struct {
    OutputDir    string
    TemplatePath string
    SourceFile   string
    Timestamp    time.Time
}

// Archiver moves processed files to archive
type Archiver interface {
    Archive(ctx context.Context, sourcePath, archiveDir string) error
}

// Logger handles structured logging
type Logger interface {
    Info(msg string, fields ...Field)
    Error(msg string, err error, fields ...Field)
    Debug(msg string, fields ...Field)
}
```

### Implementation Structure

```
internal/
├── transcribe/
│   ├── service.go          # Main orchestrator
│   ├── config.go           # Configuration loading/saving
│   ├── watcher/
│   │   ├── watcher.go      # FileWatcher interface
│   │   └── inotify.go      # Linux inotify implementation
│   ├── stabilizer/
│   │   └── poll.go         # Polling stabilizer
│   ├── client/
│   │   ├── client.go       # TranscriptionClient interface
│   │   └── faster_whisper.go # LinuxServer faster-whisper impl
│   ├── output/
│   │   └── writer.go       # OutputWriter implementation
│   ├── archive/
│   │   └── archiver.go     # Archiver implementation
│   └── logging/
│       └── logger.go       # Structured logger
cmd/
└── nota/
    └── transcribe.go       # CLI commands
```

### Concurrency Model

```
[inotify event]
      ↓
[spawn goroutine per file]
      ↓
[stabilizer.WaitForStable] ←── independent per file
      ↓
[transcriptionClient.Transcribe]
      ↓
[outputWriter.Write]
      ↓
[archiver.Archive]
      ↓
[delete from droppoint]
```

Each file processed in its own goroutine. Multiple files can stabilize and process concurrently.

---

## Transcription Service

### Container Selection

**Image:** `onerahmet/openai-whisper-asr-webservice`

| Feature | Detail |
|---------|--------|
| REST API | Yes, with Swagger UI |
| Port | 9000 |
| Engines | openai_whisper, faster_whisper, whisperx |
| Output formats | text, JSON, VTT, SRT, TSV |
| GPU support | Optional (CPU works fine) |

**Source:** [GitHub - ahmetoner/whisper-asr-webservice](https://github.com/ahmetoner/whisper-asr-webservice)

### Docker Compose

See: `compose/whisper-asr/docker-compose.yml` (to be created by polecat)

```yaml
version: "3.8"

services:
  whisper-asr:
    image: onerahmet/openai-whisper-asr-webservice:latest
    container_name: whisper-asr
    environment:
      - ASR_MODEL=base
      - ASR_ENGINE=faster_whisper
    volumes:
      - whisper-cache:/root/.cache
    ports:
      - "9000:9000"
    restart: unless-stopped

volumes:
  whisper-cache:
```

### Environment Variables

| Variable | Options | Default | Purpose |
|----------|---------|---------|---------|
| `ASR_ENGINE` | `openai_whisper`, `faster_whisper`, `whisperx` | `openai_whisper` | Transcription engine |
| `ASR_MODEL` | `tiny`, `base`, `small`, `medium`, `large-v3` | `base` | Model size |
| `ASR_MODEL_PATH` | Path | - | Custom model location |
| `ASR_DEVICE` | `cuda`, `cpu` | auto | Hardware selection |

### API Endpoint

**Swagger UI:** `http://<host>:9000/docs`

```bash
# Transcribe audio file
curl -X POST "http://localhost:9000/asr" \
  -H "accept: application/json" \
  -H "Content-Type: multipart/form-data" \
  -F "audio_file=@recording.m4a"

# With options
curl -X POST "http://localhost:9000/asr?output=json&language=en" \
  -H "accept: application/json" \
  -F "audio_file=@recording.m4a"
```

**Response (JSON):**
```json
{
  "text": "Transcribed text content...",
  "segments": [...],
  "language": "en"
}
```

**Query Parameters:**
| Param | Options | Description |
|-------|---------|-------------|
| `output` | `text`, `json`, `vtt`, `srt`, `tsv` | Response format |
| `language` | ISO code or `auto` | Source language hint |
| `word_timestamps` | `true`, `false` | Include word-level timing |

### NAS Deployment Notes

For TerraMaster T4 Max:
1. Install Docker via TOS App Center
2. Create docker-compose.yml in shared folder
3. Pull image: `docker-compose pull`
4. Start: `docker-compose up -d`
5. Access Swagger UI at: `http://<nas-ip>:9000/docs`

CPU-only transcription (no GPU). Model recommendations:
- `tiny` - Fastest, lower accuracy
- `base` - Good balance (recommended for NAS)
- `small` - Better accuracy, slower
- `medium`/`large` - Not recommended for NAS CPU

---

## Systemd Integration

### Unit File

`/etc/systemd/system/nota-transcribe.service`

```ini
[Unit]
Description=Nota Transcription Service
After=network.target

[Service]
Type=simple
User=shaun
ExecStart=/usr/local/bin/nota transcribe start
Restart=always
RestartSec=10
Environment=NOTA_VAULT_ROOT=/home/shaun/vault

[Install]
WantedBy=multi-user.target
```

### Installation

```bash
# Install service
sudo cp nota-transcribe.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable nota-transcribe
sudo systemctl start nota-transcribe

# Check status
sudo systemctl status nota-transcribe

# View logs
journalctl -u nota-transcribe -f
```

---

## Implementation Phases

### Phase 1: Core Infrastructure

**Beads:**
1. Create logging service with file rotation
2. Define interfaces (FileWatcher, TranscriptionClient, etc.)
3. Implement configuration loading/saving
4. Add `nota transcribe config` command (standard mode)

### Phase 2: File Watching

**Beads:**
1. Implement inotify FileWatcher for Linux
2. Implement polling Stabilizer
3. Add file metadata extraction (datetime from M4A)
4. Integration test with mock events

### Phase 3: Transcription Client

**Beads:**
1. Implement whisper-asr-webservice HTTP client
2. Add retry logic with exponential backoff
3. **[Polecat]** Find and document sample compose file for whisper-asr-webservice in `compose/whisper-asr/`
4. Integration test with actual API

### Phase 4: Output & Archiving

**Beads:**
1. Implement OutputWriter with template support
2. Implement filename collision handling
3. Implement Archiver
4. Add error note generation on failure

### Phase 5: Service Lifecycle

**Beads:**
1. Implement `nota transcribe start` (foreground)
2. Add daemon mode with PID file
3. Implement `nota transcribe stop`
4. Add `nota transcribe status`
5. Create systemd unit file
6. Add `nota transcribe config --advanced`

### Phase 6: Testing & Documentation

**Beads:**
1. Unit tests for all components
2. Integration tests for full pipeline
3. Update README with transcription docs
4. NAS deployment guide

---

## Testing Strategy

### Unit Tests

- Config parsing/validation
- Filename generation with collision handling
- Template application
- Metadata extraction

### Integration Tests

- File watcher detects new files
- Stabilizer correctly waits
- API client sends/receives
- Full pipeline end-to-end

### Manual Testing

- Deploy to Ubuntu server
- Deploy faster-whisper to NAS
- Record voice note on phone
- Verify Syncthing → transcription → inbox flow

---

## Future Considerations

- Windows file watcher implementation (ReadDirectoryChangesW)
- Multiple watch directories
- Webhook notifications on completion
- Web UI for status/history
- Support for other transcription backends (OpenAI API, local whisper.cpp)
