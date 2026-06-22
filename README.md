# Aivar Shield — Backend API

Go REST API for exception governance, rule registry, and audit logging.

## Stack

- Go 1.22 + chi
- PostgreSQL 16

## Quick start

```bash
cp .env.example .env
make dev
```

API runs at `http://localhost:8080`.

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check |
| POST | `/api/v1/repos` | Register a repo |
| GET | `/api/v1/repos/{fullName}` | Get repo |
| GET | `/api/v1/suppressions` | List suppressions |
| POST | `/api/v1/suppressions` | File a suppression |
| PATCH | `/api/v1/suppressions/{id}/status` | Approve / reject |
| GET | `/api/v1/rules` | List rules |
| GET | `/api/v1/rules/{ruleID}` | Get rule details |
| GET | `/api/v1/audit` | List audit log |

## Project layout

```
cmd/server/          API entrypoint
internal/api/        Router + handlers
internal/config/     Environment config
internal/db/         Database connection
internal/models/     Shared types
internal/services/   Business logic
migrations/          SQL schema
```
