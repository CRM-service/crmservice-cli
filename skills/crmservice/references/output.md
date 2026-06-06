# Output and Pagination

## Structured output shape

By default, `list` and `search` with `-o json` return a **flat JSON array** of records. Each record has `id` at the top level and attributes flattened into the same object:

```json
[
  {"id":"123","name":"Acme Corp","account_type":"Customer"}
]
```

Use array jq expressions for default JSON output:

```bash
crmservice list accounts -o json | jq 'length'
crmservice search accounts '{"$eq":["account_type","Customer"]}' -o json | jq -c '.[]'
```

Do **not** use `.data` with default `-o json` list/search output. Use `--full` only when you need the complete JSON:API envelope (`data`, `links`, `meta`, relationships, or included resources):

```bash
crmservice list accounts --full -o json | jq '.data | length'
```

`-o jsonl` writes one flattened record per line and is **preferred for streaming pipelines**, bulk exports, and agent workflows. Use `-o json` when you need a single JSON array (for example `jq 'length'` on one shot). Prefer `jsonl` for bounded or unlimited multi-page exports.

Supported formats: `table`, `json`, `yaml`, `jsonl`, `csv`.

## Fetch all pages (`--all`)

`list` and `search` support `--all` to fetch every page automatically. Rules:

- `--max-results` is **required** with `--all` (`0` = unlimited)
- `--max-results` without `--all` is an error
- `--page` and `--offset` cannot be used with `--all`
- Default `--page-size` with `--all` is 100 (unless you set `--page-size` explicitly)
- `--full` is supported with `--all`
- With `-o jsonl` or `-o csv`, `--all` streams output as each page is fetched (lower memory use than buffering all pages)
- When the cap is hit, a truncation status is printed to **stderr** in the requested output format
- Use `--verbose 1` to log page progress to stderr

**Important:** truncation status is written to **stderr only**; exit code stays 0 and stdout contains only the returned records. After every `--all --max-results N` export, inspect stderr for a `truncated` status (or capture stderr separately) before treating the result as complete.

```bash
# Preview without --all (single page)
crmservice list accounts --page-size 20 -o json

# Bounded export for pipelines
crmservice list accounts --all --max-results 500 -o jsonl

# CSV export streams one header row plus data rows per page
crmservice list accounts --all --max-results 500 -o csv

# Unlimited export (use with care)
crmservice search accounts '{"$eq":["account_type","Customer"]}' --all --max-results 0 -o jsonl
```

For single-page preview or JSON:API metadata, use normal pagination:

```bash
crmservice list accounts --page 1 --page-size 1 --full -o json | jq '.meta'
```

Manual page loops are still possible but rarely needed now that `--all` exists.

## jq output tips

Use `-o jsonl` when streaming records to `jq`; use `-o json` when you need an array.

```bash
# Convert array JSON to JSONL (use --all when exporting more than one page)
crmservice list accounts --all --max-results 500 -o json \
  | jq -c '.[]'

# Convert JSONL to an array for aggregate jq operations
crmservice list accounts --all --max-results 500 -o jsonl \
  | jq -s 'group_by(.account_type) | map({account_type: .[0].account_type, count: length})'

# Extract total from a full JSON:API response
crmservice list accounts --full -o json --page-size 1 \
  | jq -r '.meta.total // 0'
```

Battle-tested jq tips:

- Use `jq -c` for compact one-object-per-line output that can be piped into `bulk-create` / `bulk-update`.
- Use `//` for defaults: `(.email // "")`.
- Use `del(...)` to remove fields that should not be written.
- Use `jq -e` when a script should fail if a jq expression is false/null.
- Keep `id` for `bulk-update`; remove or ignore `id` for `bulk-create`.