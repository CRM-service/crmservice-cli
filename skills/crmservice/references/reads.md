# Reading Records and Schema

## List modules

```bash
crmservice modules
crmservice modules --output json
```

Module names are case-sensitive.

## List and search

```bash
crmservice list accounts
crmservice list accounts --page 2 --page-size 50
crmservice list accounts --fields "name,id,account_type"
crmservice list contacts --sort "last_name,first_name"
crmservice list accounts --sort "-created_at"
crmservice list accounts --filter '{"$and":[{"$eq":["account_type","Customer"]}]}'
crmservice list accounts --filter '{"$or":[{"$eq":["id","123"]},{"$eq":["id","456"]}]}'

crmservice search accounts '{"$and":[{"$eq":["account_type","Customer"]}]}'
crmservice search contacts '{"$or":[{"$eq":["id","123"]},{"$eq":["id","456"]}]}' --fields "id,entity_no,first_name,last_name"
crmservice search contacts '{"$eq":["mailing_city","Helsinki"]}' --sort "last_name,first_name"
```

`search` is an alias for `list --filter` with a positional filter argument.

For multi-page exports, prefer `--all` (see [output.md](output.md)). For totals without fetching records, prefer `count` (below).

## Get one record

```bash
crmservice get accounts 123
crmservice get accounts 123 --fields "name,email"
```

## Count records

Use `count` when you need a total without fetching records. It accepts the same filter JSON as `search` and uses the reporting API internally.

```bash
crmservice count accounts -o json
crmservice count accounts '{"$eq":["account_type","Customer"]}' -o json
crmservice count accounts --filter '{"$eq":["account_type","Customer"]}' -o json
crmservice count contacts --include account \
  --filter '{"$eq":["account.account_type","Customer"]}' -o json
```

`-o json` returns `{"module":"accounts","total":42}` and includes `filter` when one was used.

## Fields (schema)

```bash
crmservice fields accounts
crmservice fields accounts --force
crmservice fields accounts --output json
crmservice fields accounts --full -o json   # full raw backend schema (envelope)
```

The default output for `-o json` / `-o jsonl` etc. is a clean array of field objects. Each field includes at minimum `name`, `type`, `label`, `nullable`. Primary key field(s) are marked with `"primary": true`:

```bash
crmservice fields accounts -o json | jq '.[] | select(.primary) | .name'
```

Use `--full` to receive the complete raw schema document returned by the backend.

The `fields` command uses a persistent per-API schema cache in the OS user cache directory; cache TTL is controlled by `cache.ttl_days` and refresh behavior by `cache.auto_refresh`. Use `crmservice fields <module> --force` to bypass and regenerate the schema cache.

All fields have `name` and `label`. API calls must always use the **name**. Schema defines datatypes; invalid values must never be sent to the API. Custom fields are often prefixed with `cf_`.

## `--include` caveats

`--include` is supported for `list` and `search`, but only for real JSON:API relationship names accepted by the backend. Do not assume names like `owner` are includable relationships; in many modules they are ordinary attributes or use installation-specific relation names. If the backend returns `The requested relationship (...) doesn't exist`, the `--include` name is invalid for that module.

When you need included resources, relationships, links, or metadata, use `--full` and inspect the JSON:API envelope:

```bash
crmservice list accounts --include "<relationship>" --full -o json \
  | jq '{data: [.data[] | {id, type, attributes, relationships}], included}'
```

Without `--full`, JSON/YAML list output is intentionally flattened to record attributes plus `id`.

## Fetch all records

```bash
# Bounded export (recommended for agents)
crmservice list accounts --all --max-results 500 -o jsonl

# Unlimited (use only when necessary)
crmservice search accounts '{"$eq":["account_type","Customer"]}' --all --max-results 0 -o jsonl
```

Prefer `count` over fetching all records when you only need a total:

```bash
crmservice count accounts -o json
crmservice count accounts '{"$eq":["account_type","Customer"]}' -o json
```