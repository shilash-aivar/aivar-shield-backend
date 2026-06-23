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
| GET/PATCH | `/api/v1/organizations/{orgID}/members` | List / update member roles (admin) |
| POST | `/api/v1/organizations/{orgID}/members` | Add member by GitHub login (admin) |
| POST | `/api/v1/organizations/{orgID}/teams` | Create team (admin) |
| POST | `/api/v1/organizations/{orgID}/projects` | Create project (admin) |

## Slack notifications

Set `AIVAR_SLACK_WEBHOOK_URL` to post to Slack when suppressions are filed, approved, or rejected.

## Email notifications

Optional SMTP settings notify the same events (and expiry) to `AIVAR_NOTIFY_EMAILS` plus the requester when their email is known:

```
AIVAR_SMTP_HOST=smtp.example.com
AIVAR_SMTP_PORT=587
AIVAR_SMTP_USER=
AIVAR_SMTP_PASSWORD=
AIVAR_EMAIL_FROM=shield@example.com
AIVAR_NOTIFY_EMAILS=devops@example.com,security@example.com
```

Approved suppressions with `expires_at` are auto-marked `expired` every 15 minutes (run `aivar sync` after expiry).

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
