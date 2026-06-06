# Create, Update, and Delete

Always inspect module fields first: `crmservice fields <module> -o json`.

Field values use `--field "name=value"` syntax. Create/update can alternatively read a flat JSON object or a complete JSON:API request body from stdin. Prefer flat JSON for scripts. Do not combine stdin body input with `--field`. Update rejects a stdin `id` that does not match the command argument.

## Create

```bash
crmservice create accounts --field "name=Acme Corp" --field "account_type=Customer"

# Preview the JSON:API request body without sending it
crmservice create accounts --field "name=Acme Corp" --dry-run -o json

# Prefer flat JSON objects via stdin for scripted creates
printf '{"name":"Acme Corp","account_type":"Customer"}' \
  | crmservice create accounts

# Complete JSON:API request bodies are also supported
printf '{"data":{"type":"accounts","attributes":{"name":"Acme Corp","account_type":"Customer"}}}' \
  | crmservice create accounts
```

## Update

```bash
crmservice update accounts 123 --field "account_type=Partner" --field "notes=Updated"

# Preview the JSON:API request body without sending it
crmservice update accounts 123 --field "account_type=Partner" --dry-run -o json

# Prefer flat JSON objects via stdin for scripted updates. If id is present, it must match the argument.
printf '{"id":"123","account_type":"Partner","notes":"Updated"}' \
  | crmservice update accounts 123

# Complete JSON:API request bodies are also supported
printf '{"data":{"type":"accounts","id":"123","attributes":{"account_type":"Partner","notes":"Updated"}}}' \
  | crmservice update accounts 123
```

## Delete

```bash
crmservice delete accounts 123
crmservice delete accounts 123 -o json
```

`-o json` returns `{"deleted":true,"module":"accounts","id":"123"}`.