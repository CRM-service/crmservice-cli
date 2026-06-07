---
name: crmservice
description: "REQUIRED for interacting with the CRM-service REST API via the crmservice CLI tool. Use when working with CRM data operations including creating, reading, updating, and deleting records across CRM modules. Triggers: CRM record management, data queries, API interactions, module exploration, record field operations, and CRM data synchronization."
---

# crmservice CLI

Command-line tool for the CRM-service REST API. This skill is split into a quick start (below) and topic references in `references/`.

## Agent Quick Start

Canonical workflow for autonomous agent work. Pass `-o json` or `-o jsonl` explicitly on every data command.

```bash
# 0. Preflight (config, URL, token, auth, cache)
crmservice doctor -o json

# 1. Discover real API field names and types
crmservice fields <module> -o json

# 2. Optional filter dry-run (list/search/count validate filters automatically)
crmservice filter validate <module> '<filter-json>' -o json

# 3. Read records (prefer jsonl + --all; inspect stderr for truncated)
crmservice search <module> '<filter-json>' --all --max-results 500 -o jsonl

# 4. Preview writes before sending
crmservice create <module> --field "name=..." --dry-run -o json
printf '{"name":"..."}\n' | crmservice bulk-create <module> --dry-run -o jsonl

# 5. Execute batch writes (always --summary -o json; check failed in output)
printf '{"id":"123","field":"value"}\n' | crmservice bulk-update <module> --summary -o json
```

## Key contracts

- **stdout** = data only (records, validation results, bulk summary)
- **stderr** = truncation status, structured errors (matches `-o`)
- `list` / `search` / `count` → validate filter syntax and schema field names before API calls
- `filter validate` → optional standalone check; stdout only; exit 0 = valid, non-zero = invalid
- `--all --max-results N` → truncation status on **stderr**; exit code stays 0
- `bulk-*` production writes → `--summary -o json`; with `--continue-on-error`, check `failed > 0`
- `skill install --check` → stdout only; exit 0 = up to date; non-zero = missing or stale

API host: `customer.crmservice.fi` (see [references/configuration.md](references/configuration.md) for all config keys, environment variables, and global flags).

Recommended env defaults: `CRMSERVICE_API_URL=customer.crmservice.fi`, `CRMSERVICE_OUTPUT_FORMAT=json`, `CRMSERVICE_PAGE_SIZE=100`. After upgrading the CLI, run `crmservice skill install --check -o json`; if not up to date, run `crmservice skill install`.

## Commands

| Command | Description |
|---------|-------------|
| `bulk-create` | Create multiple records from JSONL or a JSON array |
| `bulk-update` | Update multiple records from JSONL or a JSON array |
| `count` | Count records matching a filter (reporting API) |
| `create` | Create a new record |
| `delete` | Delete a record by ID |
| `doctor` | Preflight checks (config, URL, token, cache) |
| `fields` | Show available fields for a module |
| `filter` | Filter language reference and validate |
| `get` | Get a single record by ID |
| `list` | List records with pagination |
| `modules` | List available API modules |
| `search` | Search with a JSON filter (alias for `list --filter`) |
| `skill` | Install, inspect, and print the bundled skill |
| `update` | Update an existing record |
| `whoami` | Show the authenticated CRM user |

## Reference guide

Read the matching file when you need detail beyond the quick start:

| When you need to… | Read |
|-------------------|------|
| Export, paginate, or choose output format | [references/output.md](references/output.md) |
| Build or debug a filter | [references/filters.md](references/filters.md) |
| Read, count, or inspect schema | [references/reads.md](references/reads.md) |
| Create, update, or delete one record | [references/writes.md](references/writes.md) |
| Batch create/update or migrate data | [references/bulk.md](references/bulk.md) |
| Copy-paste workflows (recent records, jq, safe flow) | [references/patterns.md](references/patterns.md) |
| Setup, auth, doctor, env defaults, skill install | [references/configuration.md](references/configuration.md) |

## Skill files

```bash
crmservice skill path
crmservice skill print
crmservice skill install
crmservice skill install --check -o json
```

`skill install` copies the full skill tree (`SKILL.md` + `references/`) to `~/.agents/skills/crmservice/`. `skill print` shows the entry-point `SKILL.md` only.