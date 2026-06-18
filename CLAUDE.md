# pgbridge — project context for Claude

## What this is

A Go daemon that bridges PostgreSQL databases to external systems via LISTEN/NOTIFY.
It connects to one or more databases, listens on channels, and dispatches events to modules
(mail, cross-db notifications, role discovery, etc.).

Part of the **Pansoinco suite** — an internal multi-tenant platform where each software
instance is a separate PostgreSQL database registered in `pansoinco_suite`.

## Architecture

```
pansoinco_suite (central DB)
  ├── sw_instance      — registered DB instances (host, port, name, owner creds)
  ├── sw_pgb           — which instances pgbridge monitors, which modules, pgb role/password
  │     └── B1_connection_string trigger builds connection_string automatically
  ├── ps_notifications — central inbox: notifications forwarded from all source DBs
  └── sw_instance_roles — DB roles discovered per instance by pgb_instance_roles module

Source databases (River, Troubled Water, etc.)
  └── pgb schema (created by pgbridge on first connect)
        ├── pgb_log       — service events
        ├── pgb_mail      — outbound email queue (pgb_mail module)
        ├── pgb_mail_settings — SMTP config
        └── pgb_notify    — notification queue forwarded to ps_notifications (pgb_notify module)
```

## Startup

pgbridge reads ALL database-to-module mappings from pansoinco_suite.
The only local file is `central.conf` — one line: the connection string to pansoinco_suite.

```bash
pgbridge [/path/to/central.conf]
# default path: /etc/pgbridge/central.conf
# env var override: PGBRIDGE_CENTRAL_CONFIG
```

No `pgbridge.conf` file-based mode — that was removed. Config lives in the DB.

## Modules

| Module | Channel | What it does |
|---|---|---|
| `pgb_mail` | `pgb_mail` | Sends email from `pgb.pgb_mail` queue via SMTP |
| `pgb_notify` | `pgb_notify` | Forwards `pgb.pgb_notify` rows to `pansoinco_suite.public.ps_notifications` |
| `pgb_instance_roles` | `pgb_instance_roles` | On new `sw_instance` insert, connects to that DB and discovers its roles into `sw_instance_roles` |

## Key files

```
cmd/pgbridge/main.go          — entry point; always uses DB config
internal/config/db_loader.go  — reads sw_pgb/sw_instance/ps_sw from pansoinco_suite
internal/config/central.go    — loads central.conf
internal/database/            — connection pool + schema initializer
internal/modules/mail/        — pgb_mail module
internal/modules/notify/      — pgb_notify module
internal/modules/roles/       — pgb_instance_roles module
migrations/                   — SQL to run on pansoinco_suite (run once, in order)
dbddl/                        — full pansoinco_suite schema dump (reference)
config/central.conf           — local dev central config (gitignored in prod)
```

## Migrations (run order)

1. `fix_connection_string_quoted_password.sql` — fixes URI bug, re-triggers existing rows
2. `ps_notifications_indexes.sql` — adds missing indexes on ps_notifications
3. `pansoinco_suite_instance_roles_trigger.sql` — trigger on sw_instance for role discovery

## Known schema conventions (pansoinco_suite)

- `sw_pgb.connection_string` is trigger-built — never set manually
- `pgb_services` is a PostgreSQL array of the `pgb_services` ENUM type
- `pgb_instance_roles` module is configured on pansoinco_suite itself (not a source DB)
- `pgb.pgb_notify` has `is_seen` column (tracks whether central DB has seen it)
- Only one trigger on `pgb.pgb_notify`: `s01_send_notification` AFTER INSERT

## Pansoinco suite context

- Central auth: PS (Pansoinco Suite) handles login, sessions, impersonation
- Each software instance is a separate PostgreSQL DB registered in `sw_instance`
- Two environments: production and quality (satellite PS relay model)
- Databases are never internet-exposed; cross-host access via SSH tunnel only
- `yps` schema per-software: stores app-local tables (roles, menus, etc.)
- `tyutyu` is the dev/admin role (foo:bar equivalent — not production credentials)
