# CRM-service CLI

A command-line tool for interacting with the CRM-service REST API.

## Installation

Download a prebuilt binary for your platform from the GitHub Releases page, or build from source:

```bash
go install .
```

## Configuration

### Config File

By default, the CLI reads `config.yaml` from the operating system's user config directory:

- Linux: `~/.config/crmservice/config.yaml`
- macOS: `~/Library/Application Support/crmservice/config.yaml`
- Windows: `%AppData%\\crmservice\\config.yaml`

Use `--config` only to override this default path.

Create a YAML config file:

```yaml
api:
  url: "https://customer.example.com/api/v1"
  timeout: 30

output:
  format: "table"
  page_size: 20

cache:
  # Optional. Defaults to the OS user cache directory under crmservice/schema.
  schema_dir: ""
  ttl_days: 24
  auto_refresh: true

auth:
  token: "your-bearer-token"
  type: "bearer"
```

### Environment Variables

```
CRMSERVICE_API_URL        API base URL
CRMSERVICE_AUTH_TOKEN     Bearer token
CRMSERVICE_OUTPUT_FORMAT  Output format (table, json, yaml, jsonl, or csv)
CRMSERVICE_PAGE_SIZE      Default page size
CRMSERVICE_TIMEOUT        Request timeout in seconds
```

### Command-line Flags

Flags override config file and environment variables. `--config` overrides the default config path:

```
--url        API base URL
--token      Bearer token
--output     Output format: table, json, yaml, jsonl, or csv
--page-size  Items per page
--timeout    Request timeout
--cache-dir  Schema cache directory
```

## Usage

### List Records

```bash
# List all accounts
crmservice list accounts

# With pagination
crmservice list accounts --page 2 --page-size 50

# With fields selection
crmservice list accounts --fields name,email,phone

# Sort contacts by last name, then first name (JSON:API sort syntax)
crmservice list contacts --sort last_name,first_name

# Show newest accounts first
crmservice list accounts --sort -created_at

# With a JSON filter
crmservice list accounts --filter '{"$and":[{"$eq":["name","Test Corp"]},{"$eq":["account_type","Customer"]}]}'

# Include related data
crmservice list accounts --include contacts

# JSON output
crmservice list accounts --output json

# JSONL output, one record per line
crmservice list accounts --output jsonl

# Full JSON:API response envelope
crmservice list accounts --output json --full
```

#### Fetch all pages (`--all`)

Use `--all` to fetch every page in one command. `--max-results` is required with `--all` (`0` = unlimited). `--page` and `--offset` cannot be used with `--all`. Default `--page-size` with `--all` is 100.

For exports and agent pipelines, prefer `-o jsonl`: one record per line streams cleanly into `jq`, `bulk-create`, and `bulk-update`. Use `-o json` when you need a single JSON array.

```bash
# Preview first page (default pagination)
crmservice list accounts --page-size 20 -o json

# Bounded export: fetch all pages, stop at 500 records
crmservice list accounts --all --max-results 500 -o jsonl

# Unlimited export (use with care)
crmservice search accounts '{"$eq":["account_type","Customer"]}' --all --max-results 0 -o jsonl

# Full JSON:API records across pages
crmservice list accounts --all --max-results 1000 --full -o json
```

When `--max-results` is reached and more records may exist, a truncation status is printed to stderr in the requested output format (`-o json`, `-o jsonl`, etc.). Use `--verbose 1` to log page progress to stderr.

### Get Record

```bash
crmservice get accounts <account-id>
crmservice get accounts <id> --fields name,email
```

### Create Record

```bash
crmservice create accounts \
  --field name="Test Corp" \
  --field email="test@example.com" \
  --field phone="123-456-7890"

# Or pass a flat JSON object via stdin
echo '{"name":"Test Corp","email":"test@example.com","phone":"123-456-7890"}' \
  | crmservice create accounts

# Full JSON:API request bodies are also supported
echo '{"data":{"type":"accounts","attributes":{"name":"Test Corp","email":"test@example.com"}}}' \
  | crmservice create accounts
```

### Bulk Create Records

`bulk-create` reads flat JSONL records or a JSON array from stdin. Client-provided `id` values are ignored because the API assigns IDs.

```bash
# Copy accounts between CRM instances
crmservice --url a.crmservice.fi list accounts --all --max-results 500 -o jsonl \
  | crmservice --url b.crmservice.fi bulk-create accounts

# Preview JSON:API request bodies without sending them
crmservice list accounts --all --max-results 500 -o jsonl \
  | crmservice bulk-create accounts --dry-run -o jsonl

# Run with concurrency and continue after individual record errors
crmservice list accounts --all --max-results 500 -o jsonl \
  | crmservice bulk-create accounts --concurrency 4 --continue-on-error --summary -o json
```

### Update Record

```bash
crmservice update accounts <id> --field name="New Name"

# Or pass a flat JSON object via stdin. If id is present, it must match the argument.
echo '{"id":"<id>","name":"New Name"}' \
  | crmservice update accounts <id>

# Full JSON:API request bodies are also supported
echo '{"data":{"type":"accounts","id":"<id>","attributes":{"name":"New Name"}}}' \
  | crmservice update accounts <id>
```

### Bulk Update Records

`bulk-update` reads flat JSONL records or a JSON array from stdin. Each record must include `id`; `id` is used as the target record ID and removed from the attributes sent to the API.

```bash
crmservice list accounts --all --max-results 500 -o jsonl \
  | jq 'select(.account_type == "Prospect") | .account_type = "Customer"' \
  | crmservice bulk-update accounts --concurrency 4 --continue-on-error
```

Bulk flags:

- `--continue-on-error`: continue processing after individual record failures
- `--dry-run`: build request bodies without sending them
- `--concurrency N`: number of concurrent API requests
- `--skip-empty`: skip empty records instead of failing
- `--summary`: output only operation counts

### Import Multiple JSONL Records

Create multiple records from a JSONL file with one flat record or JSON:API request body per line.

```bash
# accounts.jsonl
{"name":"New Account","account_type":"Customer"}
{"name":"Another Account","account_type":"Partner"}
```

```bash
crmservice bulk-create accounts --concurrency 4 < accounts.jsonl
```

Adjust `--concurrency 4` to control how many create requests run concurrently.

### Delete Record

```bash
crmservice delete accounts <id>
```

### Show Fields

```bash
crmservice fields accounts
crmservice fields accounts --force
crmservice fields accounts -o json
crmservice fields accounts --full -o json  # raw backend schema
```

`fields -o json` produces a clean array. Primary-key fields are marked `"primary": true`. `--full` returns the complete raw schema from the server.

### Search

`search` is an alias for `list --filter`; pass the JSON filter as the second positional argument.

```bash
crmservice search accounts '{"$eq":["name","Test Corp"]}'
crmservice search contacts '{"$or":[{"$eq":["id","123"]},{"$eq":["id","456"]}]}' --fields id,first_name,last_name
crmservice search contacts '{"$eq":["mailing_city","Helsinki"]}' --sort last_name,first_name
```

### Count

`count` returns the number of records in a module, optionally filtered. It uses the reporting API (not JSON:API) and is faster than fetching records when you only need a total.

```bash
# Count all records in a module
crmservice count accounts -o json

# Count with filter (positional, like search)
crmservice count accounts '{"$eq":["account_type","Customer"]}' -o json

# Count with --filter (like list)
crmservice count accounts --filter '{"$eq":["account_type","Customer"]}' -o json

# Relation filters may need --include
crmservice count contacts --include account \
  --filter '{"$eq":["account.account_type","Customer"]}' -o json
```

`-o json` returns `{"module":"accounts","total":42}` and includes `filter` when one was used.

### Filter Language

Filters are JSON expressions used by `list --filter`, `search`, and `count`. Quote them with single quotes in the shell.

Preferred syntax is an operator object:

```bash
# Exact match
crmservice search accounts '{"$eq":["account_type","Customer"]}'

# Contains text
crmservice search accounts '{"$cts":["name","Acme"]}'

# Combine conditions
crmservice list accounts --filter '{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}'

# Match any of a set
crmservice list accounts --filter '{"$in":["id",["123","456"]]}'

# Date range and relative dates
crmservice list activities --filter '{"$between":["start_date","2026-01-01","2026-01-31"]}'
crmservice list activities --filter '{"$gte":["start_date","$now.date -7 days"]}'

# Non-empty email
crmservice list contacts --filter '{"$not.null":["email"]}'
```

Common operators:

- Comparison: `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`, `$in`, `$nin`, `$between`, `$not.between`, `$is.null`, `$not.null`
- Strings: `$beg`, `$end`, `$cts`, `$not.cts`, `$like`, `$regex`
- Logic: `$and`, `$or`, `$nor`, `$not`

Use API field names from `crmservice fields <module>`. Related fields can be addressed as `relation.field` when supported by the backend.

Filter tooling:

```bash
crmservice filter reference
crmservice filter validate '{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}' -o json
crmservice filter validate accounts '{"$eq":["account_type","Customer"]}' -o json
```

`filter validate` prints the result to stdout only. Exit code 0 means the filter is valid; non-zero means invalid. The stdout payload still includes `valid`, `message`, and related fields for inspection.

### Discover Modules

```bash
crmservice modules
```

### Preflight Checks

Run before automated agent work to verify config, API reachability, token, authentication, and schema cache writability:

```bash
crmservice doctor
crmservice doctor -o json
```

`doctor -o json` returns structured fields including `ok`, `issues`, `api_reachable`, `authenticated`, and `cache_writable`. Exit code is non-zero when checks fail.

### Show Current User

```bash
crmservice whoami
crmservice whoami --output json
```

### Agent Skill

The CLI bundles the crmservice Agent Skill used by coding agents for CRM-service API workflows.

```bash
# Show where the bundled skill will be installed
crmservice skill path

# Review the bundled skill content
crmservice skill print

# Install or update the skill in ~/.agents/skills/crmservice/SKILL.md
crmservice skill install
```

`skill install` installs the latest bundled version of the skill.

## JSON:API Compliance

For `--output json` and `--output yaml`, list/search responses are flattened to record objects by default (top-level `id` plus `attributes`). Use `--full` to output the complete JSON:API response envelope. For multi-record exports, prefer `--output jsonl` (especially with `--all`).

`--max-results` is only valid with `--all`. Passing `--max-results` without `--all` is an error.

The client follows JSON:API specification (v1.0) for:
- Resource objects
- Pagination with `page` and `page_size` parameters
- Sparse fieldsets with `fields` parameter
- Filtering with `filter` parameter
- Sorting with `sort` parameter (comma-separated fields, prefix with `-` for descending)
- Compound documents with `include` parameter

## Schema Caching

Schemas are cached per API instance to reduce API calls and improve performance. The `fields <module>` command reads from this cache first, and refreshes it when the cached schema is missing or older than `cache.ttl_days`. Use `fields <module> --force` to bypass and regenerate the cached schema.

Default cache locations:

- Linux: `~/.cache/crmservice/schema/`
- macOS: `~/Library/Caches/crmservice/schema/`
- Windows: `%LocalAppData%\\crmservice\\schema\\`

Cache TTL: 24 days by default (configurable with `cache.ttl_days`).
