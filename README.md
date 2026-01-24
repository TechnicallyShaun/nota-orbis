# Nota Orbis

Personal knowledge management system with PARA-inspired structure and AI-driven workflows.

## Installation

### Using go install

```bash
go install github.com/TechnicallyShaun/nota-orbis/cmd/nota@latest
```

### Download binary

Download the latest release for your platform from [GitHub Releases](https://github.com/TechnicallyShaun/nota-orbis/releases).

Available platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

## Transcription Service

Automatically transcribe audio files using a whisper-asr-webservice instance.

### Setup

1. Configure the service (run from within a vault):

```bash
nota transcribe config
```

This prompts for:
- **Watch folder**: Directory to monitor for audio files
- **API URL**: Whisper ASR service endpoint (e.g., `http://localhost:9000/asr`)
- **Output location**: Where to save transcription markdown files
- **Template file** (optional): Custom output template
- **Archive location**: Where to move processed audio files

For advanced settings (stabilization, language, model, etc.):

```bash
nota transcribe config --advanced
```

### Running

**Foreground mode** (for testing):

```bash
nota transcribe start
```

**Daemon mode** (background service):

```bash
nota transcribe start --daemon
```

### Managing the Service

Check service status:

```bash
nota transcribe status
```

Stop the daemon:

```bash
nota transcribe stop
```

### Configuration

Configuration is stored in `.nota/transcribe.json` within your vault.

| Setting | Default | Description |
|---------|---------|-------------|
| `watch_dir` | (required) | Directory to watch for audio files |
| `api_url` | (required) | Whisper ASR service URL |
| `output_dir` | (required) | Output directory for transcriptions |
| `template_path` | (optional) | Custom template file path |
| `archive_dir` | `~/.nota/archive/audio` | Archive directory for processed files |
| `watch_patterns` | `*.m4a,*.mp3,*.wav` | File patterns to watch |
| `stabilization_interval_ms` | `2000` | Interval between file stability checks |
| `stabilization_checks` | `3` | Number of stable checks before processing |
| `language` | `auto` | Transcription language |
| `model` | `base` | Whisper model to use |
| `max_file_size_mb` | `100` | Maximum file size to process |
| `retry_count` | `3` | Number of retry attempts |

### Logs

Logs are stored in `~/.nota/logs/transcribe-YYYY-MM-DD.log`.

## Stack

- **Go**: CLI commands, file watchers, APIs, webhooks
- **TypeScript**: Automation helpers for fixed routines and integrations
