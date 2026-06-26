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
| GET | `/api/v1/rules/{ruleID}/explain` | Rule + tool documentation |
| GET | `/api/v1/tools` | List supported scanners |
| GET | `/api/v1/audit` | List audit log |
| GET | `/api/v1/reports/delivery` | Delivery report (`?repo=&org_id=&project_id=&format=json\|html`) |
| GET | `/api/v1/reports/delivery/bundle` | Signed zip bundle (report + audit + manifest) |
| GET | `/api/v1/analytics/summary` | Governance metrics for scope |
| GET | `/api/v1/audit/verify` | Verify audit log hash chain + signatures |
| GET/POST/PATCH | `/api/v1/infra/reviews` | Infra plan queue (submit, list, approve/reject) |
| GET | `/api/v1/repos` | List registered repos (`?org_id=&project_id=`) |
| POST | `/api/v1/policy/evaluate` | Runtime CI policy — which findings are blocked vs suppressed |
| POST | `/api/v1/reports/delivery/publish` | Build + store signed bundle (local dir or S3) |
| GET | `/api/v1/artifacts/{key}` | Download published artifact (local storage) |
| POST | `/api/v1/organizations/{orgID}/teams/{teamID}/members` | Team-scoped approver roles |
| POST | `/api/v1/organizations/{orgID}/members` | Add member by GitHub login (admin) |
| POST | `/api/v1/organizations/{orgID}/teams` | Create team (admin) |
| POST | `/api/v1/organizations/{orgID}/projects` | Create project (admin) |

## Production deploy

```bash
# Full stack (Postgres + API + UI)
docker compose -f docker-compose.prod.yml up --build
```

Set `AIVAR_ARTIFACTS_DIR` for local bundle storage, or `AIVAR_S3_BUCKET` + AWS credentials for S3.

## Notifications

Slack (`AIVAR_SLACK_WEBHOOK_URL`), email (SMTP), and generic webhook (`AIVAR_WEBHOOK_URL` + optional `AIVAR_WEBHOOK_SECRET`).

## GitHub OAuth

See portal README for `AIVAR_GITHUB_CLIENT_ID`, `AIVAR_GITHUB_CLIENT_SECRET`, `AIVAR_ADMIN_GITHUB_LOGINS`, and `AIVAR_APPROVER_GITHUB_LOGINS`.

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
