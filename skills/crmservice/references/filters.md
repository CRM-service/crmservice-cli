# Filters

Filters are JSON expressions used by `crmservice list <module> --filter '<json>'`, `crmservice search <module> '<json>'`, and `crmservice count <module> '<json>'`. Always quote the JSON in the shell with single quotes.

Use field **names** from `crmservice fields <module>` (not labels). Related fields can be addressed as `relation.field` when the backend exposes and permits that relation.

`list`, `search`, and `count` always validate filters locally before calling the API: JSON syntax, operator shape, and field names against the module schema (including one-level relation paths like `account.account_type`). Invalid filters fail with a non-zero exit code and structured stderr matching `-o`.

## Preferred expression shape

**Agents must use explicit operator syntax** for every filter. Do not use bare `{"field":"value"}` shorthand in agent workflows.

Each operator is a JSON object whose key is the operator and whose value is an array of arguments:

```json
{"$eq":["account_type","Customer"]}
```

Combine expressions with logical operators:

```json
{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}
```

## Equality shorthand (humans only)

A filter object may use bare field keys as a shortcut for a single equals comparison:

```json
{"account_type":"Customer"}
```

This is equivalent to `{"$eq":["account_type","Customer"]}` when the API accepts it. `list`, `search`, `count`, and `filter validate <module>` validate bare field names against the module schema (including one-level relation paths like `account.account_type`). Use explicit operators for combined conditions, non-equals comparisons, null checks, and agent-generated filters.

## Operators

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

## Common examples

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

Filter values are case-insensitive for text comparisons.

## Filter tooling

```bash
crmservice filter reference
crmservice filter validate '{"$and":[{"$eq":["account_type","Customer"]},{"$cts":["name","Acme"]}]}' -o json
crmservice filter validate accounts '{"$eq":["account_type","Customer"]}' -o json
```

`filter validate` is an optional standalone check. It writes the result to **stdout** only in the requested output format. Exit code 0 means valid; non-zero means invalid. The stdout payload includes `valid`, `message`, and related fields for inspection.

Use `crmservice filter reference` for an offline operator list. Field types and permissions are still enforced by the backend.