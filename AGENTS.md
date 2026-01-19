# Agent Instructions - Nota Orbis

## Project
Personal knowledge management system with PARA-inspired structure.
See CLAUDE.md for stack and directory layout.

## Domain Context
- Vault = Obsidian-style folder structure
- PARA = Projects, Areas, Resources, Archive
- Tools run from within the vault, as part of user workflows

## Session Workflow
Before ending a work session:
1. Run tests (`go test ./...`, `npm test`)
2. Commit and push changes
3. File issues for any remaining work
4. Provide context for the next session
