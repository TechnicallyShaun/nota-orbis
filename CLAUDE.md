# Nota Orbis - Claude Context

## Stack
- **Go**: CLI commands, file watchers, APIs, webhooks
- **TypeScript**: Automation helpers for AI-enhanced vault workflows

## Directory Structure
```
nota_orbis/
├── cmd/              # Go CLI entry points
│   └── nota/         # Main CLI binary
├── pkg/              # Go packages
│   ├── vault/        # Vault detection, path management
│   └── watcher/      # File watchers for ingestion
├── ts/               # TypeScript helpers (vault AI skills)
│   ├── skills/       # Claude skills for vault use
│   └── integrations/ # API clients (Mealie, etc.)
├── docker/           # Docker Compose for services
│   └── whisper/      # Local Whisper transcription
├── install/          # Install/update/uninstall scripts
└── docs/             # Project documentation
```

## Development Philosophy
- **Contract-first where possible**: Define interfaces and tests before implementation
- **Extensibility-aware**: When choosing between approaches, consider future extension
- **Ubuntu-first**: Target Ubuntu, cover full lifecycle (install → maintain → uninstall)

## Integrations
External services run as Docker containers via compose files.
Endpoint URLs stored in config.

## Vault Detection
Tools should detect vault context (like `gt` requires `/gt`):
- Check for vault indicators in working directory
- Clear message if run outside vault
- Vault path stored in config (prompted at install)

## Testing
- Go: `go test ./...`
- TypeScript: Vitest (`npm test`)
- Contract tests ensure deterministic behavior

## Guidelines
- Minimize code dependencies (npm/Go modules)
- Unit test file operations thoroughly
- Tools run inside vaults, as part of user workflows
