# Bulk Create and Update

Bulk commands read flat JSONL records or a JSON array from stdin.

- `bulk-create` ignores input `id`
- `bulk-update` requires input `id` (selects the record; `id` is not sent as an attribute)

## Flags

- `--continue-on-error` — keep processing after individual record failures
- `--dry-run` — preview request bodies without sending
- `--concurrency N` — parallel requests
- `--skip-empty` — skip empty records
- `--summary` — output only operation counts (recommended for production)

## Bulk create

```bash
# Copy accounts between CRM instances. Source id is ignored on create.
crmservice --url a.crmservice.fi list accounts --all --max-results 500 -o jsonl \
  | crmservice --url b.crmservice.fi bulk-create accounts

# Preview request bodies without sending them
crmservice list accounts --all --max-results 500 -o jsonl \
  | crmservice bulk-create accounts --dry-run -o jsonl

# Production write with summary
crmservice list accounts --all --max-results 500 -o jsonl \
  | crmservice bulk-create accounts --concurrency 4 --continue-on-error --summary -o json
```

## Bulk update

```bash
# Each input record must include id
crmservice list accounts --all --max-results 500 -o jsonl \
  | jq 'select(.account_type == "Prospect") | .account_type = "Customer"' \
  | crmservice bulk-update accounts --concurrency 4 --continue-on-error
```

### Filtered batch update

Validate the filter, build a minimal `{id, changed_field}` stream, and use `--summary` for production updates:

```bash
filter='{"$eq":["account_type","Prospect"]}'

crmservice filter validate accounts "$filter" -o json

crmservice search accounts "$filter" --all --max-results 500 --fields "id,account_type" -o jsonl \
  | jq -c '{id, account_type:"Customer"}' \
  | crmservice bulk-update accounts --summary -o json --concurrency 4
```

## Failure visibility

For production batch writes, always use `--summary -o json`. With `--continue-on-error`, exit code stays 0 even when individual records fail — parse the summary and treat `failed > 0` as a partial failure. Without `--summary`, stdout includes only succeeded records and failures are omitted from the output stream.

Prefer `bulk-create --dry-run` / `bulk-update --dry-run` for generated JSONL payloads before sending.