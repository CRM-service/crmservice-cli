---
name: crmservice
description: "REQUIRED for interacting with the CRM-service REST API via the crmservice CLI tool. Use when working with CRM data operations including creating, reading, updating, and deleting records across CRM modules. Triggers: CRM record management, data queries, API interactions, module exploration, record field operations, and CRM data synchronization."
---

# crmservice CLI Tool Skill

This skill provides guidance for using the `crmservice` CLI tool to interact with the CRM-service REST API.

## Overview

CRM-service CLI is a command-line tool for interacting with the CRM-service REST API. It provides commands for managing CRM records, listing modules, searching data, and exploring field schemas.

## Base Commands

| Command | Description |
|---------|-------------|
| `completion` | Generate autocompletion for bash, fish, powershell, or zsh |
| `bulk-create` | Create multiple records from JSONL or a JSON array |
| `bulk-update` | Update multiple records from JSONL or a JSON array |
| `create` | Create a new record in a module |
| `delete` | Delete a record by ID |
| `fields` | Show available fields for a module |
| `filter` | Show and validate filter language expressions |
| `get` | Get a single record by ID |
| `list` | List records with pagination |
| `modules` | List available API modules |
| `search` | Search records with a JSON filter (alias for `list --filter`) |
| `update` | Update an existing record |
| `whoami` | Show the authenticated CRM user |

## Global Flags

All commands support these global flags:

- `--config string`: Config file path
- `--token string`: Bearer token (overrides config)
- `--url string`: API base URL (overrides config)
- `--verbose int`: Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)

## Common Flags

Most data commands support:

- `-o, --output string`: Output format (table, json, yaml, jsonl, csv)
- `--full`: Include full response (not just attributes)

## Usage Examples

### List Modules
```bash
crmservice modules
crmservice modules --output json
```

### List Records
```bash
crmservice list accounts
crmservice list accounts --page 2 --page-size 50
crmservice list accounts --fields "name,id,account_type"
crmservice list contacts --sort "last_name,first_name"
crmservice list accounts --sort "-created_at"
crmservice list accounts --filter '{"$and":[{"$eq":["account_type","Customer"]}]}'
crmservice list accounts --filter '{"$or":[{"$eq":["id","123"]},{"$eq":["id","456"]}]}'
```

### Get Record
```bash
crmservice get accounts 123
crmservice get accounts 123 --fields "name,email"
```

### Create Record
```bash
crmservice create accounts --field "name=Acme Corp" --field "account_type=Customer"

# Or provide a complete JSON:API request body via stdin
printf '{"data":{"type":"accounts","attributes":{"name":"Acme Corp","account_type":"Customer"}}}' \
  | crmservice create accounts
```

### Bulk Create Records
```bash
# Copy accounts between CRM instances. Source id is ignored on create.
crmservice --url a.crmservice.fi list accounts -o jsonl \
  | crmservice --url b.crmservice.fi bulk-create accounts

# Preview request bodies without sending them
crmservice list accounts -o jsonl \
  | crmservice bulk-create accounts --dry-run -o jsonl
```

### Update Record
```bash
crmservice update accounts 123 --field "account_type=Partner" --field "notes=Updated"

# Or provide a complete JSON:API request body via stdin
printf '{"data":{"type":"accounts","id":"123","attributes":{"account_type":"Partner","notes":"Updated"}}}' \
  | crmservice update accounts 123
```

### Bulk Update Records
```bash
# Each input record must include id. id selects the record and is not sent as an attribute.
crmservice list accounts -o jsonl \
  | jq 'select(.account_type == "Prospect") | .account_type = "Customer"' \
  | crmservice bulk-update accounts --concurrency 4 --continue-on-error
```

Bulk flags: `--continue-on-error`, `--dry-run`, `--concurrency N`, `--skip-empty`, and `--summary`.

### Delete Record
```bash
crmservice delete accounts 123
```

### Search Records
```bash
crmservice search accounts '{"$and":[{"$eq":["account_type","Customer"]}]}'
crmservice search contacts '{"$or":[{"$eq":["id","123"]},{"$eq":["id","456"]}]}' --fields "id,entity_no,first_name,last_name"
crmservice search contacts '{"$eq":["mailing_city","Helsinki"]}' --sort "last_name,first_name"
```

### Show Fields
```bash
crmservice fields accounts
crmservice fields accounts --force
crmservice fields accounts --output json
```

### Filter Language Tooling
```bash
crmservice filter reference
crmservice filter validate '{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}'
```

### Show Current User
```bash
crmservice whoami
crmservice whoami --output json
```

### Completion Setup
```bash
# Bash
crmservice completion bash > ~/.local/share/bash-completion/completions/crmservice

# Zsh
crmservice completion zsh > ~/.zcompletions/_crmservice

# Fish
crmservice completion fish > ~/.config/fish/completions/crmservice.fish
```

## Filter Language

Filters are JSON expressions used by `crmservice list <module> --filter '<json>'` and `crmservice search <module> '<json>'`. Always quote the JSON in the shell with single quotes.

Use field **names** from `crmservice fields <module>` (not labels). Related fields can be addressed as `relation.field` when the backend exposes and permits that relation.

### Preferred Expression Shape

Each operator is a JSON object whose key is the operator and whose value is an array of arguments:

```json
{"$eq":["account_type","Customer"]}
```

Combine expressions with logical operators:

```json
{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}
```

### Operators

| Operator | Arguments | Meaning |
|----------|-----------|---------|
| `$eq` | `[field, value]` | equals |
| `$ne` | `[field, value]` | not equals |
| `$gt`, `$gte` | `[field, value]` | greater than / greater than or equal |
| `$lt`, `$lte` | `[field, value]` | less than / less than or equal |
| `$in`, `$nin` | `[field, [values...]]` | in / not in a set |
| `$between`, `$not.between` | `[field, from, to]` | between / outside range |
| `$is.null`, `$not.null` | `[field]` | is NULL / is not NULL |
| `$beg`, `$end` | `[field, value]` | begins with / ends with |
| `$cts`, `$not.cts` | `[field, value]` | contains / does not contain |
| `$like` | `[field, pattern]` | SQL LIKE pattern, use `%` wildcards |
| `$regex` | `[field, pattern]` | SQL REGEXP pattern |
| `$and`, `$or`, `$nor` | `[expressions...]` | logical groups |
| `$not` | `expression` | negates one expression |

### Common Filter Examples

```bash
# Exact match
crmservice search accounts '{"$eq":["account_type","Customer"]}'

# Contains text
crmservice search accounts '{"$cts":["name","Acme"]}'

# Combine conditions
crmservice list accounts --filter '{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}'

# Match any of a set
crmservice list accounts --filter '{"$in":["id",["123","456"]]}'

# Date range
crmservice list activities --filter '{"$between":["start_date","2026-01-01","2026-01-31"]}'

# Relative date/time with $now expressions
crmservice list activities --filter '{"$gte":["start_date","$now.date -7 days"]}'

# Null / non-null
crmservice list contacts --filter '{"$not.null":["email"]}'

# Related field, if the relation exists and is readable
crmservice list contacts --filter '{"$eq":["account.account_type","Customer"]}'
```

### Filter Tooling

Use `crmservice filter reference` for an offline reference and `crmservice filter validate '<json>'` to catch JSON syntax errors and common operator shape mistakes before calling the API. Validation is intentionally lightweight; field existence and permissions are still checked by the backend.

## Configuration

By default, the CLI reads `config.yaml` from the operating system's user config directory: Linux `~/.config/crmservice/config.yaml`, macOS `~/Library/Application Support/crmservice/config.yaml`, and Windows `%AppData%\\crmservice\\config.yaml`. Use `--config` only to override this default path.

Set environment variables or use a config file:

- `CRMSERVICE_API_URL`: API base URL
- `CRMSERVICE_AUTH_TOKEN`: Bearer token for authentication

## Notes

- Module names are case-sensitive
- To get available modules use the modules command
- To get available fields for module use the fields [module] command. This schema also defines the datatype for the field
- The fields command uses a persistent per-API schema cache in the OS user cache directory; cache TTL is controlled by `cache.ttl_days` and refresh behavior by `cache.auto_refresh`
- Use `crmservice fields <module> --force` to bypass and regenerate the schema cache
- All fields have name and label. API call must always use the name
- Schema defines the datatype fields. Invalid values must never be sent to the API
- Field schema may contain custom fields. Name prefixed with `cf_`
- Field values for create/update use `--field "name=value"` syntax
- Create/update can alternatively read a complete JSON:API request body from stdin; do not combine stdin body input with `--field`
- Bulk-create/bulk-update read flat JSONL records or a JSON array from stdin; bulk-create ignores input `id`, bulk-update requires input `id`
- Output formats support table (default), json, yaml, jsonl, and csv
- For `--output json` and `--output yaml`, list/search output is a flat array of records by default: top-level `id` plus resource `attributes`; use `--full` for the complete JSON:API response envelope
- `--output jsonl` writes one JSON record per line
- Filters must be valid JSON filter expressions; use `crmservice filter reference` and `crmservice filter validate '<json>'` for help
- Pagination uses --page and --page-size or --offset
- List/search sorting uses JSON:API `--sort` syntax: comma-separated fields, with `-` prefix for descending (for example `--sort "last_name,first_name"` or `--sort "-created_at"`)
- Authentication must always be available by either: default config file, environment variable or command-line argument
