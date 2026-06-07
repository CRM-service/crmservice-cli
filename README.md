# CRM-service CLI

Command-line client for the [CRM-service](https://crmservice.com) REST API. Read and write CRM records, run filtered queries, export data, and automate workflows from the terminal or in agent pipelines.

Built for humans and automation: structured **stdout** for data, **stderr** for errors and status, schema-aware filter validation, and a bundled Agent Skill for coding assistants.

## Features

- **JSON:API client** — list, get, create, update, delete with pagination, sparse fieldsets, sorting, includes, and filters
- **Reporting `count`** — fast record totals without fetching full datasets
- **Bulk operations** — `bulk-create` and `bulk-update` from JSONL or JSON arrays with concurrency and dry-run
- **Filter language** — JSON operators (`$eq`, `$and`, `$cts`, …) with local syntax and schema validation
- **Schema cache** — field discovery via `fields` with TTL-based refresh
- **Output formats** — `table`, `json`, `yaml`, `jsonl`, `csv`; flattened records by default, `--full` for JSON:API envelopes
- **Agent tooling** — `doctor` preflight checks, bundled skill install/drift detection, deterministic JSON for `modules` and `fields`

## Quick start

```bash
# Install (see Installation below for release binaries)
go install .

# Configure (or use flags / env vars)
export CRMSERVICE_API_URL=customer.crmservice.fi
export CRMSERVICE_AUTH_TOKEN=your-token

# Preflight
crmservice doctor -o json

# Discover modules and fields
crmservice modules -o json
crmservice fields accounts -o json

# Search and export
crmservice search accounts '{"$eq":["account_type","Customer"]}' --all --max-results 500 -o jsonl
```

## Installation

Download a prebuilt binary from [GitHub Releases](https://github.com/CRM-service/crmservice-cli/releases), or build from source:

```bash
go install .
```

Requires Go 1.23+.

## Configuration

Set the CRM hostname or API base URL — for example `customer.crmservice.fi` or `https://customer.crmservice.fi/api/v1`.

**Precedence:** command-line flags → environment variables → config file.

### Config file

Default path:

| OS | Path |
|----|------|
| Linux | `~/.config/crmservice/config.yaml` |
| macOS | `~/Library/Application Support/crmservice/config.yaml` |
| Windows | `%AppData%\crmservice\config.yaml` |

Override with `--config`. See `config.example.yaml` for a starter template.

```yaml
api:
  url: "customer.crmservice.fi"
  timeout: 30

output:
  format: "table"
  page_size: 20

cache:
  schema_dir: ""          # default: OS cache dir / crmservice/schema
  ttl_days: 24
  auto_refresh: true

auth:
  token: "your-bearer-token"
```

> [!WARNING]
> If you store `auth.token` in a config file, restrict permissions (`chmod 600` on Linux and macOS). Prefer `CRMSERVICE_AUTH_TOKEN` or `--token` in CI and agent environments.

| Key | Description |
|-----|-------------|
| `api.url` | CRM hostname or API base URL |
| `api.timeout` | Request timeout (seconds) |
| `output.format` | Default output: `table`, `json`, `yaml`, `jsonl`, `csv` |
| `output.page_size` | Default page size |
| `cache.schema_dir` | Schema cache directory (`~` expanded) |
| `cache.ttl_days` | Schema cache TTL (days) |
| `cache.auto_refresh` | Refresh cache when missing or expired |
| `auth.token` | Bearer token |

### Environment variables

| Variable | Description |
|----------|-------------|
| `CRMSERVICE_API_URL` | CRM hostname or API base URL |
| `CRMSERVICE_AUTH_TOKEN` | Bearer token |
| `CRMSERVICE_OUTPUT_FORMAT` | Default output format |
| `CRMSERVICE_PAGE_SIZE` | Default page size |
| `CRMSERVICE_TIMEOUT` | Request timeout (seconds) |
| `CRMSERVICE_CACHE_DIR` | Schema cache directory |
| `CRMSERVICE_CACHE_TTL_DAYS` | Schema cache TTL (days) |
| `CRMSERVICE_CACHE_AUTO_REFRESH` | `true` or `false` |

### Global flags

Available on every command:

| Flag | Description |
|------|-------------|
| `--config` | Config file path |
| `--url` | CRM hostname or API base URL |
| `--token` | Bearer token |
| `--timeout` | Request timeout (seconds) |
| `-o`, `--output` | Output format |
| `--page-size` | Default page size |
| `--cache-dir` | Schema cache directory |
| `--cache-ttl-days` | Schema cache TTL (days) |
| `--cache-auto-refresh` | Auto-refresh schema cache |

Commands may define additional flags (`--filter`, `--all`, `--verbose`, …). Per-command `-o` overrides the global default.

## Output

| Stream | Contents |
|--------|----------|
| **stdout** | Records, validation results, bulk summaries |
| **stderr** | Structured command errors when output format is known, truncation status, `--verbose` progress |

Supported formats: `table` (default), `json`, `yaml`, `jsonl`, `csv`.

For `list` and `search`, JSON/YAML responses are **flattened** by default (`id` plus attribute fields). Use `--full` for the complete JSON:API envelope. For multi-record exports, prefer `jsonl` — especially with `--all`. Note that `--all -o jsonl --full` streams full resource objects and included resources as JSONL records rather than a single JSON:API envelope.

## Commands

| Command | Description |
|---------|-------------|
| `list <module>` | List records with pagination, filters, sorting |
| `search <module> <filter>` | Alias for `list` with a positional filter |
| `get <module> <id>` | Fetch one record |
| `count <module> [filter]` | Count records (reporting API) |
| `create <module>` | Create a record (`--field` or stdin) |
| `update <module> <id>` | Update a record |
| `delete <module> <id>` | Delete a record |
| `bulk-create <module>` | Create many records from stdin (JSONL / JSON array) |
| `bulk-update <module>` | Update many records from stdin |
| `fields <module>` | List module fields (from schema cache) |
| `modules` | List API modules |
| `filter reference` | Print filter operator reference |
| `filter validate [module] <json>` | Validate filter syntax and optionally schema |
| `doctor` | Preflight: config, URL, auth, cache |
| `whoami` | Authenticated CRM user |
| `skill` | Install, inspect, and print the bundled Agent Skill |
| `completion` | Shell completion scripts |

### Reading data

```bash
crmservice list accounts --page 2 --page-size 50
crmservice list accounts --fields name,email --sort -created_at
crmservice list accounts --filter '{"$eq":["account_type","Customer"]}' -o jsonl
crmservice get accounts <id> --fields name,email
crmservice search contacts '{"$eq":["mailing_city","Helsinki"]}' --sort last_name
crmservice count accounts '{"$eq":["account_type","Customer"]}' -o json
```

`count` uses the reporting API and is faster than listing when you only need a total. Table and JSON output include the filter when one was used.

#### Export all pages (`--all`)

`--all` fetches every page in one command. `--max-results` is **required** with `--all` (`0` = unlimited). Cannot combine with `--page` or `--offset`. Default page size with `--all` is 100.

```bash
crmservice list accounts --all --max-results 500 -o jsonl
crmservice search accounts '{"$eq":["account_type","Customer"]}' --all --max-results 0 -o jsonl
```

`jsonl` and `csv` stream page by page. Other formats buffer all pages before writing.

> [!IMPORTANT]
> When `--max-results` is reached and more records may exist, truncation status is written to **stderr** (exit code stays `0`). Inspect stderr after every bounded `--all` export.

Use `--verbose 1` for page progress on stderr.

### Writing data

```bash
crmservice create accounts --field name="Acme" --field account_type="Customer"
echo '{"name":"Acme","account_type":"Customer"}' | crmservice create accounts

crmservice update accounts <id> --field name="New Name"
crmservice delete accounts <id>
```

`create` and `update` accept flat JSON on stdin or full JSON:API request bodies. Use `--dry-run` to preview the request without sending it.

Empty attribute payloads are rejected before the API call.

### Bulk operations

`bulk-create` and `bulk-update` read **flat JSONL**, a **JSON array**, or a single **JSON object** from stdin.

| Input shape | Notes |
|-------------|-------|
| Flat JSONL | `{"id":"123","name":"Acme"}` — typical for pipelines |
| JSON:API | `{"data":{"type":"accounts","id":"123","attributes":{...}}}` |
| Record `id` | String or numeric; normalized to string for update URLs |

`bulk-create` ignores client-provided IDs. `bulk-update` requires an `id` per record.

Cross-site pipelines use separate config files so the read and write targets are explicit:

```bash
# Export from site A, create on site B
crmservice --config site-a.yaml list accounts --all --max-results 500 -o jsonl \
  | crmservice --config site-b.yaml bulk-create accounts

# Preview request bodies before sending to site B
crmservice --config site-a.yaml list accounts --all --max-results 500 -o jsonl \
  | crmservice --config site-b.yaml bulk-create accounts --dry-run -o jsonl

# Export from site A, transform, update on site B (IDs must exist on site B)
crmservice --config site-a.yaml list accounts --all --max-results 500 -o jsonl \
  | jq 'select(.account_type == "Prospect") | .account_type = "Customer"' \
  | crmservice --config site-b.yaml bulk-update accounts --concurrency 4 --continue-on-error --summary -o json
```

Same-site bulk update:

```bash
crmservice --config site-a.yaml list accounts --all --max-results 500 -o jsonl \
  | jq 'select(.account_type == "Prospect") | .account_type = "Customer"' \
  | crmservice --config site-a.yaml bulk-update accounts --concurrency 4 --continue-on-error --summary -o json
```

| Flag | Description |
|------|-------------|
| `--concurrency N` | Parallel API requests |
| `--continue-on-error` | Keep processing after per-record failures |
| `--dry-run` | Build bodies without sending |
| `--skip-empty` | Skip empty records |
| `--summary` | Output operation counts only |

> [!TIP]
> For production bulk writes, use `--summary -o json`. With `--continue-on-error`, exit code can be `0` while `failed > 0` — check the summary payload.

### Schema and discovery

```bash
crmservice modules -o json
crmservice fields accounts -o json          # clean field list; primary keys marked
crmservice fields accounts --full -o json   # complete backend schema under the data key
crmservice fields accounts --force          # bypass cache
```

Module and field JSON output is sorted by name for stable diffs and hashes.

Default schema cache locations:

| OS | Path |
|----|------|
| Linux | `~/.cache/crmservice/schema/` |
| macOS | `~/Library/Caches/crmservice/schema/` |
| Windows | `%LocalAppData%\crmservice\schema\` |

TTL defaults to 24 days (`cache.ttl_days`).

### Filters

Filters are JSON expressions for `list --filter`, `search`, and `count`. Quote with single quotes in the shell.

**Use explicit operators** in scripts and agent workflows:

```bash
crmservice search accounts '{"$eq":["account_type","Customer"]}'
crmservice list accounts --filter '{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}'
crmservice list contacts --filter '{"$eq":["account.account_type","Customer"]}'
```

| Category | Operators |
|----------|-----------|
| Comparison | `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`, `$in`, `$nin`, `$between`, `$not.between`, `$is.null`, `$not.null` |
| Strings | `$beg`, `$end`, `$cts`, `$not.cts`, `$like`, `$regex` |
| Logic | `$and`, `$or`, `$nor`, `$not` |

Use field **names** from `crmservice fields <module>`. Relation paths like `account.account_type` are supported when the backend exposes the relation.

`list`, `search`, and `count` validate filters locally before calling the API: JSON syntax, operator shape, and field names against the module schema.

**Equality shorthand** for interactive use: `{"account_type":"Customer"}` is treated as `$eq`. Field names in bare keys are schema-validated the same way. Prefer explicit operators everywhere else.

```bash
crmservice filter reference
crmservice filter validate accounts '{"$eq":["account_type","Customer"]}' -o json
```

`filter validate` writes the result to stdout. Exit `0` = valid; exit `1` = invalid.

### Automation helpers

```bash
crmservice doctor -o json
crmservice whoami -o json
crmservice skill install
crmservice skill install --check -o json
```

`doctor` checks config, API reachability, authentication, and cache writability. `skill install --check` compares the installed Agent Skill tree to the bundled copy (SHA-256 manifest).

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error |

| Situation | Exit `0` | Exit `1` |
|-----------|----------|----------|
| Most commands | Completed | Any error |
| `filter validate` | Valid filter | Invalid filter |
| `doctor` | All checks passed | One or more failed |
| `skill install --check` | Skill up to date | Missing or stale |
| `list` / `search --all` | Records returned (may be truncated) | Fetch/validation error |
| `bulk-* --continue-on-error` | Processing finished | Input/setup error |
| `bulk-*` (default) | All records succeeded | Any record failed |

## Shell completion

```bash
# Bash (requires bash-completion package)
crmservice completion bash > ~/.local/share/bash-completion/completions/crmservice

# Zsh
crmservice completion zsh > ~/.zcompletions/_crmservice

# Fish
crmservice completion fish > ~/.config/fish/completions/crmservice.fish
```

Restart your shell after installing.

## Agent Skill

The CLI bundles a multi-file Agent Skill (`SKILL.md` plus `references/*.md`) for CRM workflows in coding assistants.

```bash
crmservice skill path      # default install target path
crmservice skill print     # print SKILL.md
crmservice skill install   # install to ~/.agents/skills/crmservice/
```

After upgrading the CLI, run `crmservice skill install --check -o json` and `crmservice skill install` when the check reports `stale` or `missing`. The check compares the installed skill files against the bundled skill manifest.

## JSON:API

The client speaks JSON:API v1.0: resource objects, pagination query parameters (`page[number]`, `page[size]`), sparse fieldsets (`fields[...]`), filters (`filter`), sorting (`sort`, prefix `-` for descending), and compound documents (`include`).

`--max-results` requires `--all`.