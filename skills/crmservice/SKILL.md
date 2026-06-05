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

## Common Agent / Scripting Patterns

These are preferred copy-pasteable idioms for agents and scripts. Always inspect module fields first when constructing writes or non-trivial filters.

### Most Recently Created / Most Recently Updated

```bash
# Newest created account
crmservice list accounts --sort "-created_at" --page-size 1 -o json

# Newest updated account, if the module exposes updated_at
crmservice list accounts --sort "-updated_at" --page-size 1 -o json

# Same as JSONL for easy shell pipelines
crmservice list accounts --sort "-created_at" --page-size 1 -o jsonl
```

If a module uses different timestamp field names, check first:

```bash
crmservice fields accounts -o json | jq -r '.[].name | select(test("created|updated|modified"; "i"))'
```

### Search by Name (Partial, Case-Insensitive)

Use `$cts` for contains searches. On normal CRM text fields this is the most convenient partial-name search.

```bash
crmservice search accounts '{"$cts":["name","acme"]}' --fields "id,name,entity_no" -o json
crmservice search contacts '{"$or":[{"$cts":["first_name","john"]},{"$cts":["last_name","john"]}]}' --fields "id,first_name,last_name,email" -o json
```

If you specifically need SQL wildcard matching, use `$like` with `%`:

```bash
crmservice search accounts '{"$like":["name","%acme%"]}' --fields "id,name" -o json
```

Filter values are case-insensitive.

### List Owner / Creator Fields

`--include` is supported by the CLI for `list` and `search`, but only for real JSON:API relationship names accepted by the backend. Do not assume names like `owner` or creator-related names are includable relationships; in many modules they are ordinary attributes or use installation-specific relation names. If the backend returns `The requested relationship (...) doesn't exist`, the `--include` name is invalid for that module.

First discover owner/creator/assignee fields exposed as attributes:

```bash
crmservice fields accounts -o json \
  | jq -r '.[] | select(.name|test("owner|created|creator|modified|assigned"; "i")) | [.name,.label,.type,.extras] | @tsv'
```

Then list only fields that actually exist in the schema output:

```bash
# Replace <owner_field> / <creator_field> with real field names from `crmservice fields`.
crmservice list accounts --fields "id,name,<owner_field>,<creator_field>" -o json
```

If you know the module has a real relationship name, use `--include` with `--full` and inspect the returned JSON:API envelope:

```bash
# Replace <relationship> with a relationship name confirmed for the module/API.
crmservice list accounts --include "<relationship>" --full -o json \
  | jq '{data: [.data[] | {id, type, attributes, relationships}], included}'
```

Use `--full` when you need included resources, relationships, links, or metadata. Without `--full`, JSON/YAML list output is intentionally flattened to record attributes plus `id`.

### Paginate Until Exhausted / Get Total First

Get total and page metadata first:

```bash
crmservice list accounts --page 1 --page-size 1 --full -o json \
  | jq '.meta'
```

Paginate until an empty page is returned:

```bash
page=1
while :; do
  batch=$(crmservice list accounts --page "$page" --page-size 100 -o json)
  count=$(jq 'length' <<<"$batch")
  [ "$count" -eq 0 ] && break
  jq -c '.[]' <<<"$batch"
  page=$((page + 1))
done
```

If the backend returns a total in `.meta.total`, compute page count first:

```bash
page_size=100
total=$(crmservice list accounts --page 1 --page-size 1 --full -o json | jq -r '.meta.total // 0')
pages=$(( (total + page_size - 1) / page_size ))
for page in $(seq 1 "$pages"); do
  crmservice list accounts --page "$page" --page-size "$page_size" -o jsonl
done
```

### Safe Create / Update Flow

Use this flow before writes, especially in automated agent work:

```bash
# 1. Inspect real API field names, labels, and datatypes
crmservice fields accounts -o json | jq -r '.[] | [.name,.label,.type,.nullable] | @tsv'

# 2. Confirm target fields exist
crmservice fields accounts -o json \
  | jq -e 'map(.name) as $names | ["name","account_type"] | all(. as $f | $names | index($f))'

# 3. Validate filter syntax before reading/updating target records
crmservice filter validate '{"$eq":["name","Acme Corp"]}'

# 4. Preview creates without sending them
printf '%s\n' '{"name":"Acme Corp","account_type":"Customer"}' \
  | crmservice bulk-create accounts --dry-run -o jsonl

# 5. Send after validation/preview
printf '%s\n' '{"name":"Acme Corp","account_type":"Customer"}' \
  | crmservice bulk-create accounts --summary -o json

# 6. For updates, require id and dry-run first
crmservice search accounts '{"$eq":["name","Acme Corp"]}' -o jsonl \
  | jq -c '.account_type = "Customer"' \
  | crmservice bulk-update accounts --dry-run -o jsonl
```

Prefer `bulk-create --dry-run` / `bulk-update --dry-run` for generated JSONL payloads. Prefer `--summary` for production batch writes unless you need every returned record.

### Effective jq Combinations

Use `-o jsonl` when streaming records to `jq`; use `-o json` when you need an array.

```bash
# Pick only a few fields from JSONL output
crmservice list accounts -o jsonl \
  | jq -c '{id, name, entity_no, account_type}'

# Select records with missing/blank values
crmservice list contacts -o jsonl \
  | jq -c 'select((.email // "") == "") | {id, first_name, last_name}'

# Build a safe bulk-update stream: preserve id, modify one attribute
crmservice search accounts '{"$eq":["account_type","Prospect"]}' -o jsonl \
  | jq -c '{id, account_type: "Customer"}' \
  | crmservice bulk-update accounts --dry-run -o jsonl

# Remove read-only/system fields before bulk-create into another instance
crmservice --url a.crmservice.fi list accounts -o jsonl \
  | jq -c 'del(.id, .created_at, .updated_at)' \
  | crmservice --url b.crmservice.fi bulk-create accounts --dry-run -o jsonl

# Convert array JSON to JSONL
crmservice list accounts -o json \
  | jq -c '.[]'

# Convert JSONL to an array for aggregate jq operations
crmservice list accounts -o jsonl \
  | jq -s 'group_by(.account_type) | map({account_type: .[0].account_type, count: length})'

# Extract total from a full JSON:API response
crmservice list accounts --full -o json --page-size 1 \
  | jq -r '.meta.total // 0'
```

Battle-tested jq tips:

- Use `jq -c` for compact one-object-per-line output that can be piped into `bulk-create` / `bulk-update`.
- Use `//` for defaults: `(.email // "")`.
- Use `del(...)` to remove fields that should not be written.
- Use `jq -e` when a script should fail if a validation expression is false/null.
- Keep `id` for `bulk-update`; remove or ignore `id` for `bulk-create`.

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
