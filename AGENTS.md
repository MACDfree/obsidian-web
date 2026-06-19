# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Project Overview

obsidian-web is a Go web application that publishes an Obsidian vault as a browsable website. It supports both public and private (password-protected) notes, with an optional comment system. The primary use case is reading PC-created notes on mobile devices. The codebase uses Chinese for comments and UI text.

## Build & Run

```bash
go build                    # Produces ./obsidian-web binary
go run main.go              # Run directly
./obsidian-web -gitpull     # Pull vault via go-git before starting (requires git_url in config)
```

- Requires **CGO enabled** (uses `mattn/go-sqlite3`)
- Config is loaded from `config.yml` in the working directory
- No tests, Makefile, or linter configuration exists in this project

## Architecture

**Startup flow** (`main.go` → `flag.Parse()` → `gitutil.Pull()` → `noteloader.Load()` → `db.InitCommentDB()` → `server.NewRouter()` → `job.Start()` → `r.Run()`):
1. If `-gitpull` flag is set and `git_url` is configured, pull (or clone) the vault via go-git
2. Vault scan (`noteloader.Load()`): walks the Obsidian vault directory, parses YAML frontmatter from `.md` files, indexes notes/attachments/wikilinks into SQLite (`tmp.db`), and builds note-to-note link relationships
3. Initialize the comment database (`db.InitCommentDB()`): creates `data/comments.db` and migrates the `Comment` schema
4. Gin HTTP server with middleware chain (`server.NewRouter()`)
5. Cron scheduler starts (`job.Start()`) — daily go-git pull at 04:12

**Request flow**: Gin middleware chain is `gin.Recovery() → Logger → Session("webauth") → Auth → Error`. Auth middleware sets `isLogin` in context; handlers check note publish status to enforce access control.

**Key packages**:
- `config/` — Config singleton loaded from `config.yml` via `sync.Once`
- `db/` — GORM models (`Note`, `AttachInfo`, `NoteAttachment`, `NoteLink`, `Comment`) and all database queries. Also includes `BacklinkInfo`, `TagCount` response types and a sensitive-word blocklist (`IsContentBlocked`)
- `handler/` — Gin HTTP handlers for notes, tags, search, attachments, auth, comments (CRUD + admin), git pull, and static files
- `mdparser/` — Goldmark markdown pipeline with wikilink resolution, syntax highlighting (Chroma/Monokai), MathJax, and custom image rendering
- `noteloader/` — Vault scanner that parses frontmatter, builds the SQLite index, and creates note-link/note-attachment relationships
- `server/` — Router setup, middleware registration, template loading (multitemplate with Sprig funcmap)
- `middleware/` — Auth, session (in-memory `memstore`), logging (Gin formatter), error pages
- `job/` — Cron-based scheduled tasks (daily go-git pull)
- `gitutil/` — go-git wrapper: pull, clone, status text, and `IsConfigured()` check
- `logger/` — Zap structured logging with file rotation (`logs/app.log`)

**Access control**: Notes with `publish: true` in frontmatter are public; others require login. Attachments inherit the publish status of referencing notes (checked via `NoteAttachment` join).

**Comment system**: Supports nested comments (parent/child tree), IP-based rate limiting (3 per 10 minutes), sensitive-word blocklist filtering, and admin management (hide/delete). Comments are stored in a separate SQLite database (`data/comments.db`).

**Markdown pipeline**: Goldmark with GFM, `[[wikilink]]` resolution (→ `/note/` or `/attachment/` based on file extension), Chroma syntax highlighting (Monokai, line numbers), optional MathJax (enabled via `mathjax: true` in frontmatter), YAML frontmatter extraction, and auto-prefixed image paths for bare filenames.

**Backlinks**: `NoteLink` tracks inter-note wikilinks. `GetBacklinks()` returns all notes linking to a given note, filtered by publish status for non-logged-in users.

**Search**: Full-text search scans accessible note titles and file contents via regex, returning highlighted matches.

**Templates**: Go `html/template` with `multitemplate` for layout composition (`base.html` + view templates + partials). Sprig functions and a `SafeHTML` helper are available.

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Note list (paginated) |
| GET | `/page/:num` | Note list page N |
| GET | `/tag/*tag` | Tag list or notes by tag |
| GET | `/note/*path` | Note detail page |
| GET | `/search` | Search notes |
| GET | `/attachment/:attach` | Serve attachment file |
| GET | `/auth` | Login page |
| POST | `/auth` | Login action |
| GET | `/gitpull` | Git pull page (login required) |
| POST | `/gitpull` | Execute git pull (login required) |
| GET | `/api/comments` | List comments for a note |
| POST | `/api/comments` | Create a comment |
| POST | `/api/comments/:id/delete` | Hide comment (login required) |
| GET | `/admin/comments` | Admin comment management (login required) |
| GET | `/assets/*` | Static assets directory |
| GET | `/:static` | Static files from `static/` |

## Configuration (`config.yml`)

- `note_path` — Absolute path to the Obsidian vault
- `ignore_paths` — Directory names to skip during vault scan
- `attachment_path` — Subdirectory name for attachments
- `paginate` — Notes per page (default: 20)
- `title` — Site title
- `password` — Password for private notes (empty = no auth required)
- `bind_addr` — Server bind address (default: `127.0.0.1:8888`)
- `git_url` — Git repository URL for go-git pull/clone (optional, enables `-gitpull` flag and cron job)
- `git_token` — Authentication token for private git repositories (optional)

## Conventions

- Commit messages follow conventional-commit style: `feat(scope):`, `fix(scope):`, `refactor:`, etc.
- Session keys are randomly generated at startup (in-memory store, sessions lost on restart)
- Two separate SQLite databases: `tmp.db` (notes/attachments/links) and `data/comments.db` (comments), both using WAL mode
- The Dockerfile uses multi-stage build with UPX compression, Chinese mirror repos, and Asia/Shanghai timezone
- CI/CD is via Gitea Actions (`.gitea/workflows/`) — builds Docker image on tag push
- Git operations use go-git library (`github.com/go-git/go-git/v6`), not shell commands
