# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

obsidian-web is a Go web application that publishes an Obsidian vault as a browsable website. It supports both public and private (password-protected) notes. The primary use case is reading PC-created notes on mobile devices. The codebase uses Chinese for comments and UI text.

## Build & Run

```bash
go build                    # Produces ./obsidian-web binary
go run main.go              # Run directly
./obsidian-web -pre "git pull"  # Run with pre-startup script
```

- Requires **CGO enabled** (uses `mattn/go-sqlite3`)
- Config is loaded from `config.yml` in the working directory
- No tests, Makefile, or linter configuration exists in this project

## Architecture

**Startup flow** (`main.go` → `noteloader.Load()` → `server.NewRouter()` → `job.Start()` → `r.Run()`):
1. Optional pre-script (e.g., `git pull` to sync vault)
2. Vault scan: walks the Obsidian vault directory, parses YAML frontmatter from `.md` files, indexes notes/attachments/wikilinks into SQLite (`tmp.db`)
3. Gin HTTP server with middleware chain
4. Cron scheduler starts (daily git pull at 04:12)

**Request flow**: Gin middleware chain is `logger → session → auth → error handler`. Auth middleware sets `isLogin` in context; handlers check note publish status to enforce access control.

**Key packages**:
- `config/` — Config singleton loaded from `config.yml` via `sync.Once`
- `db/` — GORM models (`Note`, `AttachInfo`, `NoteAttachment`, `NoteLink`) and all database queries
- `handler/` — Gin HTTP handlers for notes, attachments, auth, git pull, and static files
- `mdparser/` — Goldmark markdown pipeline with wikilink resolution, syntax highlighting (Chroma/Monokai), MathJax, and custom image rendering
- `noteloader/` — Vault scanner that parses frontmatter and builds the SQLite index
- `server/` — Router setup, middleware registration, template loading
- `middleware/` — Auth, session (in-memory `memstore`), logging, error pages
- `job/` — Cron-based scheduled tasks
- `logger/` — Zap structured logging with file rotation

**Access control**: Notes with `publish: true` in frontmatter are public; others require login. Attachments inherit the publish status of referencing notes.

**Markdown pipeline**: Goldmark with GFM, `[[wikilink]]` resolution (→ `/note/` or `/attachment/`), Chroma syntax highlighting, optional MathJax, YAML frontmatter extraction, and auto-prefixed image paths.

**Templates**: Go `html/template` with `multitemplate` for layout composition (`base.html` + view templates). Sprig functions and a `SafeHTML` helper are available.

## Configuration (`config.yml`)

- `note_path` — Absolute path to the Obsidian vault
- `ignore_paths` — Directory names to skip during vault scan
- `attachment_path` — Subdirectory name for attachments
- `paginate` — Notes per page (default: 20)
- `title` — Site title
- `password` — Password for private notes (empty = no auth)
- `bind_addr` — Server bind address (default: `127.0.0.1:8888`)

## Conventions

- Commit messages follow conventional-commit style: `feat(scope):`, `fix(scope):`, `refactor:`, etc.
- Session keys are randomly generated at startup (in-memory store, sessions lost on restart)
- The Dockerfile uses multi-stage build with UPX compression, Chinese mirror repos, and Asia/Shanghai timezone
- CI/CD is via Gitea Actions (`.gitea/workflows/`) — builds Docker image on tag push
