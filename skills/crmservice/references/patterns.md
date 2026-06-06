# Agent Patterns and Recipes

Always inspect module fields first when constructing writes or non-trivial filters: `crmservice fields <module> -o json`.

## Safe create / update flow

```bash
# 0. Preflight: config, URL, token, auth, cache
crmservice doctor -o json

# 1. Inspect real API field names, labels, and datatypes
crmservice fields accounts -o json | jq -r '.[] | [.name,.label,.type,.nullable] | @tsv'

# 2. Confirm target fields exist
crmservice fields accounts -o json \
  | jq -e 'map(.name) as $names | ["name","account_type"] | all(. as $f | $names | index($f))'

# 3. Optional filter dry-run (list/search/count validate automatically)
crmservice filter validate accounts '{"$eq":["name","Acme Corp"]}' -o json

# 4. Preview creates without sending them
printf '%s\n' '{"name":"Acme Corp","account_type":"Customer"}' \
  | crmservice bulk-create accounts --dry-run -o jsonl

# 5. Send after validation/preview
printf '%s\n' '{"name":"Acme Corp","account_type":"Customer"}' \
  | crmservice bulk-create accounts --summary -o json

# 6. For updates, require id and dry-run first
crmservice search accounts '{"$eq":["name","Acme Corp"]}' --all --max-results 500 -o jsonl \
  | jq -c '{id, account_type: "Customer"}' \
  | crmservice bulk-update accounts --dry-run -o jsonl
```

## Most recently created / updated

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

## Search by name (partial)

Use `$cts` for contains searches. On normal CRM text fields this is the most convenient partial-name search.

```bash
crmservice search accounts '{"$cts":["name","acme"]}' --fields "id,name,entity_no" -o json
crmservice search contacts '{"$or":[{"$cts":["first_name","john"]},{"$cts":["last_name","john"]}]}' \
  --fields "id,first_name,last_name,email" -o json
```

If you specifically need SQL wildcard matching, use `$like` with `%`:

```bash
crmservice search accounts '{"$like":["name","%acme%"]}' --fields "id,name" -o json
```

## Owner / creator fields

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

See [reads.md](reads.md) for `--include` relationship caveats.

## jq pipelines

```bash
# Pick only a few fields from JSONL output
crmservice list accounts --all --max-results 500 -o jsonl \
  | jq -c '{id, name, entity_no, account_type}'

# Select records with missing/blank values
crmservice list contacts --all --max-results 500 -o jsonl \
  | jq -c 'select((.email // "") == "") | {id, first_name, last_name}'

# Build a safe bulk-update stream: preserve id, modify one attribute
crmservice search accounts '{"$eq":["account_type","Prospect"]}' --all --max-results 500 -o jsonl \
  | jq -c '{id, account_type: "Customer"}' \
  | crmservice bulk-update accounts --dry-run -o jsonl

# Remove read-only/system fields before bulk-create into another instance
crmservice --url a.crmservice.fi list accounts --all --max-results 500 -o jsonl \
  | jq -c 'del(.id, .created_at, .updated_at)' \
  | crmservice --url b.crmservice.fi bulk-create accounts --dry-run -o jsonl
```

More output and jq tips: [output.md](output.md).