# Module Relations

CRM modules are linked through named **relations**. The relation **name** (used in `--include` and JSON:API relationship URLs) often differs from the target **module** name. Example: invoices expose a `rows` relation that points at the `invoice_rows` module.

Use this reference with `crmservice fields <module>` to choose the right module, relation path, and filter shape.

> **Agent essentials**
>
> - **Relation name ≠ module name** — e.g. `rows` on `invoices` → `invoice_rows` module; line-item FK is `entity_id`.
> - **Default filter strategy** — use FK fields on the module you search (`account_id`) or search the **child module** directly; avoid `relation.field` paths unless you have no alternative.
> - **Missing a related ID** — two-step: search the related module, build a full `$in` filter with `jq -sc`, then search the target module.
> - **Confirm before filtering** — `crmservice fields <module> -o json` (and `--full` for relation names).
> - **Full table below** — lookup only; do not read it end-to-end. Jump to [Non-standard relation names](#non-standard-relation-names) or search for the parent module.

## CLI filter validation

`list`, `search`, and `count` validate every filter field against the module schema **before** calling the API. A filter must use field names that exist on the module being searched.

**What passes local validation:**

- Direct attribute names on the module (`account_id`, `name`, `status`, …)
- One-level relation paths where the prefix is declared in schema:
  - FK fields with `relationType` / `relationName` (e.g. `account_id` → prefix `account`)
  - hasMany relation names from the schema `relations` block (e.g. `activities` on `accounts`)

**Preferred agent pattern — filter by FK field or query the child module:**

```bash
# hasOne: parent stores the FK (account_id on contacts)
crmservice search contacts '{"$eq":["account_id","ACCOUNT_ID"]}' -o json

# hasMany (direct): child stores the FK — search the child module
crmservice search contacts '{"$eq":["account_id","ACCOUNT_ID"]}' -o jsonl

# Filter by a related attribute when you don't have the ID — two-step
# Build complete filter JSON with jq; never splice IDs into partial JSON via xargs (breaks quoting).
filter=$(crmservice search accounts '{"$eq":["account_type","Customer"]}' --all --max-results 500 -o jsonl \
  | jq -sc 'select(length > 0) | {"$in":["account_id",[.[].id]]}')
[ -n "$filter" ] && crmservice search contacts "$filter" -o jsonl
```

**Shell filter-building rules:**

- **Multiple IDs** → `jq -sc` to slurp jsonl into a full `$in` object (see above).
- **Single ID** → capture in a variable, then `jq -n --arg id "$id" '{"$eq":["field",$id]}'`.
- **Never** `xargs -I{}` with `'{"$eq":["field","{}"]}'` — `{}` stays literal and IDs are not JSON-escaped.
- **Single-record pipes** → use `-o jsonl` and `jq -r '.id'`; `-o json` returns an array (use `.[0].id` if you must).

Do **not** use `relation.field` paths (like `account.account_type`) as your default approach. The CRM REST API supports them, but they are harder to validate and reason about. Use FK fields or child-module queries instead.

## Relation types

| Type | How it links | Typical FK pattern | Filter strategy |
|------|--------------|-------------------|-----------------|
| **hasOne** | Parent stores FK to one related record | `account_id` on parent → `accounts.id` | Filter parent by FK: `{"$eq":["account_id","ID"]}` |
| **hasMany** (direct) | Child stores FK to parent | `account_id` on child → `accounts.id` | Search **child** module: `{"$eq":["account_id","PARENT_ID"]}` |
| **hasMany** (many-to-many) | Linked through a named relation | No direct FK on parent | Search parent with `relation.field` + `--include`, or fetch via `GET /<module>/<id>/<relation>` |

### hasOne example

Parent `contacts` has `account` → `accounts` via `account_id`:

```bash
crmservice search contacts '{"$eq":["account_id","ACCOUNT_ID"]}' -o json
```

### hasMany (direct) example

Parent `accounts` has many `contacts`; child stores `account_id`:

```bash
crmservice search contacts '{"$eq":["account_id","ACCOUNT_ID"]}' -o json
```

### hasMany (many-to-many) example

Parent `accounts` has many `activities` (many-to-many relation):

```bash
crmservice search accounts '{"$eq":["activities.id","ACTIVITY_ID"]}' --include activities -o json
```

## Non-standard relation names

Relation names are API identifiers. They are not always the pluralized module name.

| Parent module | Relation name | Actual module | Notes |
|---------------|---------------|---------------|-------|
| `invoices` | `rows` | `invoice_rows` | Line items; child FK is `entity_id` (not `invoice_id`) |
| `quotes` | `rows` | `quote_rows` | Same `entity_id` pattern |
| `sales_orders` | `rows` | `sales_order_rows` | Same `entity_id` pattern |
| `purchase_orders` | `rows` | `purchase_order_rows` | Same `entity_id` pattern |
| `accounts` | `children` | `accounts` | Sub-accounts via `parent_id` |
| `accounts` | `parent` | `accounts` | Parent account (self-referential hasOne) |
| `users` | `reports_to` | `users` | Manager (hasOne); `subordinates` is the inverse hasMany |
| `users` | `created_*` | various | Records created by user (`creator_id` FK on child) |
| Most entities | `owner` | `users` | FK field is `owner_id` |
| Most entities | `creator` / `updater` | `users` | Audit relations via `creator_id` / `updater_id` |

### Inventory line items (`rows`)

Fetch line items for a parent document by querying the row module with `entity_id`:

```bash
crmservice search invoice_rows '{"$eq":["entity_id","INVOICE_ID"]}' -o jsonl
```

Filter rows by product using the row module FK field:

```bash
crmservice search invoice_rows '{"$eq":["product_id","PRODUCT_ID"]}' -o json
```

To find rows by product name, resolve the product ID first:

```bash
product_id=$(crmservice search products '{"$cts":["name","Widget"]}' --page-size 1 -o jsonl | jq -r '.id')
[ -n "$product_id" ] && crmservice search invoice_rows \
  "$(jq -n --arg id "$product_id" '{"$eq":["product_id",$id]}')" -o jsonl
```

## Discover relations at runtime

Always confirm relation names and FK fields for the target installation:

```bash
# All relations declared on a module (name, type, target module)
crmservice fields invoices --full -o json | jq '.relations | to_entries[] | {name: .key, type: .value.type, module: .value.class}'

# FK fields on a module (field name → related module in relationModule)
crmservice fields contacts -o json | jq '.[] | select(.relationModule != null) | {name, module: .relationModule}'

# Relation metadata on a FK attribute
crmservice fields contacts --full -o json | jq '.attributes.account_id | {relationType, relationName, relationModules}'
```

API endpoints for related data:

- `GET /<module>/<id>/<relation>` — list related records (hasMany) or fetch related record (hasOne)
- `GET /<module>/<id>/relationships/<relation>` — JSON:API relationship identifiers

## Filter patterns (quick reference)

| Goal | Approach | Example |
|------|----------|---------|
| Parent linked to a specific related record (by ID) | FK field on parent | `{"$eq":["account_id","123"]}` on `contacts` |
| Child records belonging to a parent | FK on child module | `{"$eq":["account_id","123"]}` on `contacts` |
| Parent linked by related attribute (no ID yet) | Two-step: search related module, then `$in` on FK | see two-step example above |
| Many-to-many linked records | `relation.id` on parent + `--include` | `{"$eq":["activities.id","456"]}` on `accounts` |
| Line items for a document | Query row module with `entity_id` | `{"$eq":["entity_id","INVOICE_ID"]}` on `invoice_rows` |
| Records owned by a user | `owner_id` FK field | `{"$eq":["owner_id","USER_ID"]}` |

Replace `VALUE`, `PARENT_ID`, `ACCOUNT_ID`, `RELATED_ID`, and `INVOICE_ID` with real values. Run `crmservice fields <module> -o json` to confirm field names before filtering.

See [filters.md](filters.md) for operator syntax and [reads.md](reads.md) for `--include` caveats.

## Full relation reference

Table columns:

- **Parent module** — module the relation is declared on
- **Relation name** — API relation identifier (may differ from module name)
- **Relation module** — API module of the related records
- **Type** — `hasOne`, `hasMany`, or `hasMany (many-to-many)`
- **Filter example** — CLI-validated pattern using FK fields or child-module search; substitute IDs

| Parent module | Relation name | Relation module | Type | Filter example |
|---------------|---------------|-----------------|------|----------------|
| account_scheme_modules | account_scheme | account_schemes | hasOne | `crmservice search account_scheme_modules '{"$eq":["account_scheme_id","RELATED_ID"]}'` |
| account_schemes | module | account_scheme_modules | hasMany | `crmservice search account_scheme_modules '{"$eq":["account_scheme_id","PARENT_ID"]}'` |
| accounts | activities | activities | hasMany (many-to-many) | `crmservice search accounts '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| accounts | campaigns | campaigns | hasMany (many-to-many) | `crmservice search accounts '{"$eq":["campaigns.id","VALUE"]}' --include campaigns` |
| accounts | children | accounts | hasMany | `crmservice search accounts '{"$eq":["parent_id","PARENT_ID"]}'` |
| accounts | communication_actions | communication_actions | hasMany | `crmservice search communication_actions '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | contacts | contacts | hasMany | `crmservice search contacts '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | contracts | contract2s | hasMany | `crmservice search contract2s '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | documents | documents | hasMany (many-to-many) | `crmservice search accounts '{"$eq":["documents.id","VALUE"]}' --include documents` |
| accounts | emails | emails | hasMany | `crmservice search emails '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | expenses | project_expenses | hasMany | `crmservice search project_expenses '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | leads | leads | hasMany | `crmservice search leads '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | multi_account_projects | projects | hasMany (many-to-many) | `crmservice search accounts '{"$eq":["multi_account_projects.id","VALUE"]}' --include multi_account_projects` |
| accounts | organizations | organizations | hasMany | `crmservice search organizations '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | parent | accounts | hasOne | `crmservice search accounts '{"$eq":["parent_id","RELATED_ID"]}'` |
| accounts | payments | payments | hasMany | `crmservice search payments '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | price_books | price_books | hasMany (many-to-many) | `crmservice search accounts '{"$eq":["price_books.id","VALUE"]}' --include price_books` |
| accounts | products | products | hasMany (many-to-many) | `crmservice search accounts '{"$eq":["products.id","VALUE"]}' --include products` |
| accounts | projects | projects | hasMany | `crmservice search projects '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | receipts | receipts | hasMany | `crmservice search receipts '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | relations | relations | hasMany | `crmservice search relations '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["account_id","PARENT_ID"]}'` |
| accounts | target_groups | target_groups | hasMany (many-to-many) | `crmservice search accounts '{"$eq":["target_groups.id","VALUE"]}' --include target_groups` |
| accounts | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["account_id","PARENT_ID"]}'` |
| activities | accounts | accounts | hasMany (many-to-many) | `crmservice search activities '{"$eq":["accounts.id","VALUE"]}' --include accounts` |
| activities | campaign | campaigns | hasOne | `crmservice search activities '{"$eq":["campaign_id","RELATED_ID"]}'` |
| activities | campaigns | campaigns | hasMany (many-to-many) | `crmservice search activities '{"$eq":["campaigns.id","VALUE"]}' --include campaigns` |
| activities | contacts | contacts | hasMany (many-to-many) | `crmservice search activities '{"$eq":["contacts.id","VALUE"]}' --include contacts` |
| activities | expenses | activity_expenses | hasMany | `crmservice search activity_expenses '{"$eq":["activity_id","PARENT_ID"]}'` |
| activities | invoices | invoices | hasMany (many-to-many) | `crmservice search activities '{"$eq":["invoices.id","VALUE"]}' --include invoices` |
| activities | leads | leads | hasMany (many-to-many) | `crmservice search activities '{"$eq":["leads.id","VALUE"]}' --include leads` |
| activities | potentials | potentials | hasMany (many-to-many) | `crmservice search activities '{"$eq":["potentials.id","VALUE"]}' --include potentials` |
| activities | project | projects | hasOne | `crmservice search activities '{"$eq":["project_id","RELATED_ID"]}'` |
| activities | purchase_orders | purchase_orders | hasMany (many-to-many) | `crmservice search activities '{"$eq":["purchase_orders.id","VALUE"]}' --include purchase_orders` |
| activities | quotes | quotes | hasMany (many-to-many) | `crmservice search activities '{"$eq":["quotes.id","VALUE"]}' --include quotes` |
| activities | sales_orders | sales_orders | hasMany (many-to-many) | `crmservice search activities '{"$eq":["sales_orders.id","VALUE"]}' --include sales_orders` |
| activities | space | spaces | hasOne | `crmservice search activities '{"$eq":["space_id","RELATED_ID"]}'` |
| activities | tickets | tickets | hasMany (many-to-many) | `crmservice search activities '{"$eq":["tickets.id","VALUE"]}' --include tickets` |
| activity_expenses | account | accounts | hasOne | `crmservice search activity_expenses '{"$eq":["account_id","RELATED_ID"]}'` |
| activity_expenses | activity | activities | hasOne | `crmservice search activity_expenses '{"$eq":["activity_id","RELATED_ID"]}'` |
| activity_expenses | approver | users | hasOne | `crmservice search activity_expenses '{"$eq":["approver_id","RELATED_ID"]}'` |
| activity_expenses | creator | users | hasOne | `crmservice search activity_expenses '{"$eq":["creator_id","RELATED_ID"]}'` |
| activity_expenses | invoice | invoices | hasOne | `crmservice search activity_expenses '{"$eq":["invoice_id","RELATED_ID"]}'` |
| activity_expenses | product | products | hasOne | `crmservice search activity_expenses '{"$eq":["product_id","RELATED_ID"]}'` |
| activity_expenses | user | users | hasOne | `crmservice search activity_expenses '{"$eq":["user_id","RELATED_ID"]}'` |
| announcements | sales_groups | sales_groups | hasMany (many-to-many) | `crmservice search announcements '{"$eq":["sales_groups.id","VALUE"]}' --include sales_groups` |
| billing_companies | billing_company_banks | billing_company_banks | hasMany | `crmservice search billing_company_banks '{"$eq":["company_id","PARENT_ID"]}'` |
| campaigns | accounts | accounts | hasMany (many-to-many) | `crmservice search campaigns '{"$eq":["accounts.id","VALUE"]}' --include accounts` |
| campaigns | activities | activities | hasMany (many-to-many) | `crmservice search campaigns '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| campaigns | campaign_unsubscriptions | campaign_unsubscriptions | hasMany | `crmservice search campaign_unsubscriptions '{"$eq":["campaign_id","PARENT_ID"]}'` |
| campaigns | communication_actions | communication_actions | hasMany | `crmservice search communication_actions '{"$eq":["campaign_id","PARENT_ID"]}'` |
| campaigns | contacts | contacts | hasMany (many-to-many) | `crmservice search campaigns '{"$eq":["contacts.id","VALUE"]}' --include contacts` |
| campaigns | documents | documents | hasMany (many-to-many) | `crmservice search campaigns '{"$eq":["documents.id","VALUE"]}' --include documents` |
| campaigns | leads | leads | hasMany (many-to-many) | `crmservice search campaigns '{"$eq":["leads.id","VALUE"]}' --include leads` |
| campaigns | payments | payments | hasMany | `crmservice search payments '{"$eq":["campaign_id","PARENT_ID"]}'` |
| campaigns | potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["campaign_id","PARENT_ID"]}'` |
| campaigns | product | products | hasOne | `crmservice search campaigns '{"$eq":["product_id","RELATED_ID"]}'` |
| campaigns | products | products | hasMany (many-to-many) | `crmservice search campaigns '{"$eq":["products.id","VALUE"]}' --include products` |
| campaigns | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["campaign_id","PARENT_ID"]}'` |
| campaigns | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["campaign_id","PARENT_ID"]}'` |
| campaigns | vendor | vendors | hasOne | `crmservice search campaigns '{"$eq":["vendor_id","RELATED_ID"]}'` |
| client_responsability_units | accounts | accounts | hasMany | `crmservice search accounts '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | activities | activities | hasMany | `crmservice search activities '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | campaigns | campaigns | hasMany | `crmservice search campaigns '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | contacts | contacts | hasMany | `crmservice search contacts '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | contracts | contracts | hasMany | `crmservice search contracts '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | documents | documents | hasMany | `crmservice search documents '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | emails | emails | hasMany | `crmservice search emails '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | faqs | faqs | hasMany | `crmservice search faqs '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | leads | leads | hasMany | `crmservice search leads '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | price_books | price_books | hasMany | `crmservice search price_books '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | products | products | hasMany | `crmservice search products '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | projects | projects | hasMany | `crmservice search projects '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | sales_channels | sales_channels | hasMany | `crmservice search sales_channels '{"$eq":["target_id","PARENT_ID"]}'` |
| client_responsability_units | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | target_groups | target_groups | hasMany | `crmservice search target_groups '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | users | users | hasMany | `crmservice search users '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| client_responsability_units | vendors | vendors | hasMany | `crmservice search vendors '{"$eq":["client_responsability_unit_id","PARENT_ID"]}'` |
| communication_actions | account | accounts | hasOne | `crmservice search communication_actions '{"$eq":["account_id","RELATED_ID"]}'` |
| communication_actions | campaign | campaigns | hasOne | `crmservice search communication_actions '{"$eq":["campaign_id","RELATED_ID"]}'` |
| communication_actions | contact | contacts | hasOne | `crmservice search communication_actions '{"$eq":["contact_id","RELATED_ID"]}'` |
| communication_actions | creator | users | hasOne | `crmservice search communication_actions '{"$eq":["creator_id","RELATED_ID"]}'` |
| contacts | account | accounts | hasOne | `crmservice search contacts '{"$eq":["account_id","RELATED_ID"]}'` |
| contacts | activities | activities | hasMany (many-to-many) | `crmservice search contacts '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| contacts | campaigns | campaigns | hasMany (many-to-many) | `crmservice search contacts '{"$eq":["campaigns.id","VALUE"]}' --include campaigns` |
| contacts | communication_actions | communication_actions | hasMany | `crmservice search communication_actions '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | contracts | contract2s | hasMany | `crmservice search contract2s '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | documents | documents | hasMany (many-to-many) | `crmservice search contacts '{"$eq":["documents.id","VALUE"]}' --include documents` |
| contacts | emails | emails | hasMany | `crmservice search emails '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | payments | payments | hasMany | `crmservice search payments '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | portals | portal_users | hasMany | `crmservice search portal_users '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | potential | potentials | hasOne | `crmservice search contacts '{"$eq":["contact_id","RELATED_ID"]}'` |
| contacts | potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | price_books | price_books | hasMany (many-to-many) | `crmservice search contacts '{"$eq":["price_books.id","VALUE"]}' --include price_books` |
| contacts | products | products | hasMany (many-to-many) | `crmservice search contacts '{"$eq":["products.id","VALUE"]}' --include products` |
| contacts | projects | projects | hasMany | `crmservice search projects '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | receipts | receipts | hasMany | `crmservice search receipts '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | registration_groups | mass_event_registration_groups | hasMany | `crmservice search mass_event_registration_groups '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | relations | relations | hasMany | `crmservice search relations '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | target_groups | target_groups | hasMany (many-to-many) | `crmservice search contacts '{"$eq":["target_groups.id","VALUE"]}' --include target_groups` |
| contacts | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["contact_id","PARENT_ID"]}'` |
| contacts | vendors | vendors | hasMany (many-to-many) | `crmservice search contacts '{"$eq":["vendors.id","VALUE"]}' --include vendors` |
| contract2s | account | accounts | hasOne | `crmservice search contract2s '{"$eq":["account_id","RELATED_ID"]}'` |
| contract2s | contact | contacts | hasOne | `crmservice search contract2s '{"$eq":["contact_id","RELATED_ID"]}'` |
| contract2s | rows | contract2_rows | hasMany | `crmservice search contract2_rows '{"$eq":["contract_id","PARENT_ID"]}'` |
| contracts | accounts | accounts | hasMany (many-to-many) | `crmservice search contracts '{"$eq":["accounts.id","VALUE"]}' --include accounts` |
| contracts | projects | projects | hasMany | `crmservice search projects '{"$eq":["contract_id","PARENT_ID"]}'` |
| dashboard_items | dashboard | dashboards | hasOne | `crmservice search dashboard_items '{"$eq":["dashboard_id","RELATED_ID"]}'` |
| dashboards | items | dashboard_items | hasMany | `crmservice search dashboard_items '{"$eq":["dashboard_id","PARENT_ID"]}'` |
| dashboards | sales_groups | sales_groups | hasMany | `crmservice search sales_groups '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| documents | accounts | accounts | hasMany (many-to-many) | `crmservice search documents '{"$eq":["accounts.id","VALUE"]}' --include accounts` |
| documents | campaigns | campaigns | hasMany (many-to-many) | `crmservice search documents '{"$eq":["campaigns.id","VALUE"]}' --include campaigns` |
| documents | contacts | contacts | hasMany (many-to-many) | `crmservice search documents '{"$eq":["contacts.id","VALUE"]}' --include contacts` |
| documents | faqs | faqs | hasMany (many-to-many) | `crmservice search documents '{"$eq":["faqs.id","VALUE"]}' --include faqs` |
| documents | invoices | invoices | hasMany (many-to-many) | `crmservice search documents '{"$eq":["invoices.id","VALUE"]}' --include invoices` |
| documents | leads | leads | hasMany (many-to-many) | `crmservice search documents '{"$eq":["leads.id","VALUE"]}' --include leads` |
| documents | potentials | potentials | hasMany (many-to-many) | `crmservice search documents '{"$eq":["potentials.id","VALUE"]}' --include potentials` |
| documents | products | products | hasMany (many-to-many) | `crmservice search documents '{"$eq":["products.id","VALUE"]}' --include products` |
| documents | projects | projects | hasMany (many-to-many) | `crmservice search documents '{"$eq":["projects.id","VALUE"]}' --include projects` |
| documents | purchase_orders | purchase_orders | hasMany (many-to-many) | `crmservice search documents '{"$eq":["purchase_orders.id","VALUE"]}' --include purchase_orders` |
| documents | quotes | quotes | hasMany (many-to-many) | `crmservice search documents '{"$eq":["quotes.id","VALUE"]}' --include quotes` |
| documents | sales_orders | sales_orders | hasMany (many-to-many) | `crmservice search documents '{"$eq":["sales_orders.id","VALUE"]}' --include sales_orders` |
| documents | tickets | tickets | hasMany (many-to-many) | `crmservice search documents '{"$eq":["tickets.id","VALUE"]}' --include tickets` |
| emails | account | accounts | hasOne | `crmservice search emails '{"$eq":["account_id","RELATED_ID"]}'` |
| emails | contact | contacts | hasOne | `crmservice search emails '{"$eq":["contact_id","RELATED_ID"]}'` |
| emails | potential | potentials | hasOne | `crmservice search emails '{"$eq":["potential_id","RELATED_ID"]}'` |
| emails | project | projects | hasOne | `crmservice search emails '{"$eq":["project_id","RELATED_ID"]}'` |
| emails | quote | quotes | hasOne | `crmservice search emails '{"$eq":["quote_id","RELATED_ID"]}'` |
| entities | client_responsability_unit | client_responsability_units | hasOne | `crmservice search entities '{"$eq":["client_responsability_unit_id","RELATED_ID"]}'` |
| entities | creator | users | hasOne | `crmservice search entities '{"$eq":["creator_id","RELATED_ID"]}'` |
| entities | files | files | hasMany (many-to-many) | `crmservice search entities '{"$eq":["files.id","VALUE"]}' --include files` |
| entities | owner | users | hasOne | `crmservice search entities '{"$eq":["owner_id","RELATED_ID"]}'` |
| entities | sales_channel | sales_channels | hasOne | `crmservice search entities '{"$eq":["sales_channel_id","RELATED_ID"]}'` |
| entities | sales_group | sales_groups | hasOne | `crmservice search entities '{"$eq":["sales_group_id","RELATED_ID"]}'` |
| entities | sales_location | sales_locations | hasOne | `crmservice search entities '{"$eq":["sales_location_id","RELATED_ID"]}'` |
| entities | updater | users | hasOne | `crmservice search entities '{"$eq":["updater_id","RELATED_ID"]}'` |
| faqs | documents | documents | hasMany (many-to-many) | `crmservice search faqs '{"$eq":["documents.id","VALUE"]}' --include documents` |
| faqs | product | products | hasOne | `crmservice search faqs '{"$eq":["product_id","RELATED_ID"]}'` |
| invoice_payments | invoice | invoices | hasOne | `crmservice search invoice_payments '{"$eq":["invoice_id","RELATED_ID"]}'` |
| invoice_rows | invoice | invoices | hasOne | `crmservice search invoice_rows '{"$eq":["entity_id","RELATED_ID"]}'` |
| invoices | account | accounts | hasOne | `crmservice search invoices '{"$eq":["account_id","RELATED_ID"]}'` |
| invoices | activities | activities | hasMany (many-to-many) | `crmservice search invoices '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| invoices | contact | contacts | hasOne | `crmservice search invoices '{"$eq":["contact_id","RELATED_ID"]}'` |
| invoices | customer_contact | contacts | hasOne | `crmservice search invoices '{"$eq":["customer_contact_id","RELATED_ID"]}'` |
| invoices | documents | documents | hasMany (many-to-many) | `crmservice search invoices '{"$eq":["documents.id","VALUE"]}' --include documents` |
| invoices | end_customer | accounts | hasOne | `crmservice search invoices '{"$eq":["end_customer_id","RELATED_ID"]}'` |
| invoices | end_customer_contact | contacts | hasOne | `crmservice search invoices '{"$eq":["end_customer_contact_id","RELATED_ID"]}'` |
| invoices | invoice_payments | invoice_payments | hasMany | `crmservice search invoice_payments '{"$eq":["invoice_id","PARENT_ID"]}'` |
| invoices | mass_event | mass_events | hasOne | `crmservice search invoices '{"$eq":["mass_event_id","RELATED_ID"]}'` |
| invoices | organization | organizations | hasOne | `crmservice search invoices '{"$eq":["organization_id","RELATED_ID"]}'` |
| invoices | original_invoice | invoices | hasOne | `crmservice search invoices '{"$eq":["original_invoice_id","RELATED_ID"]}'` |
| invoices | payments | payments | hasMany | `crmservice search payments '{"$eq":["invoice_id","PARENT_ID"]}'` |
| invoices | project | projects | hasOne | `crmservice search invoices '{"$eq":["project_id","RELATED_ID"]}'` |
| invoices | quote | quotes | hasOne | `crmservice search invoices '{"$eq":["quote_id","RELATED_ID"]}'` |
| invoices | receipts | receipts | hasMany | `crmservice search receipts '{"$eq":["invoice_id","PARENT_ID"]}'` |
| invoices | relations | relations | hasMany | `crmservice search relations '{"$eq":["invoice_id","PARENT_ID"]}'` |
| invoices | rows | invoice_rows | hasMany | `crmservice search invoice_rows '{"$eq":["entity_id","PARENT_ID"]}'` |
| invoices | sales_order | sales_orders | hasOne | `crmservice search invoices '{"$eq":["salesorder_id","RELATED_ID"]}'` |
| leads | account | accounts | hasOne | `crmservice search leads '{"$eq":["account_id","RELATED_ID"]}'` |
| leads | activities | activities | hasMany (many-to-many) | `crmservice search leads '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| leads | campaigns | campaigns | hasMany (many-to-many) | `crmservice search leads '{"$eq":["campaigns.id","VALUE"]}' --include campaigns` |
| leads | documents | documents | hasMany (many-to-many) | `crmservice search leads '{"$eq":["documents.id","VALUE"]}' --include documents` |
| leads | payments | payments | hasMany | `crmservice search payments '{"$eq":["lead_id","PARENT_ID"]}'` |
| leads | products | products | hasMany (many-to-many) | `crmservice search leads '{"$eq":["products.id","VALUE"]}' --include products` |
| leads | receipts | receipts | hasMany | `crmservice search receipts '{"$eq":["lead_id","PARENT_ID"]}'` |
| leads | relations | relations | hasMany | `crmservice search relations '{"$eq":["lead_id","PARENT_ID"]}'` |
| list_viewes | creator | users | hasOne | `crmservice search list_viewes '{"$eq":["creator_id","RELATED_ID"]}'` |
| list_viewes | sales_groups | sales_groups | hasMany | `crmservice search sales_groups '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| list_viewes | sales_order_templates | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["target_template_list_id","PARENT_ID"]}'` |
| mass_event_registration_groups | contact | contacts | hasOne | `crmservice search mass_event_registration_groups '{"$eq":["contact_id","RELATED_ID"]}'` |
| mass_event_registrations | contact | contacts | hasOne | `crmservice search mass_event_registrations '{"$eq":["contact_id","RELATED_ID"]}'` |
| mass_event_registrations | mass_event | mass_events | hasOne | `crmservice search mass_event_registrations '{"$eq":["mass_event_id","RELATED_ID"]}'` |
| mass_event_rows | mass_event | mass_events | hasOne | `crmservice search mass_event_rows '{"$eq":["mass_event_id","RELATED_ID"]}'` |
| mass_event_rows | product | products | hasOne | `crmservice search mass_event_rows '{"$eq":["product_id","RELATED_ID"]}'` |
| mass_event_rows | space | spaces | hasOne | `crmservice search mass_event_rows '{"$eq":["space_id","RELATED_ID"]}'` |
| mass_event_staffs | contact | contacts | hasOne | `crmservice search mass_event_staffs '{"$eq":["contact_id","RELATED_ID"]}'` |
| mass_event_staffs | mass_event | mass_events | hasOne | `crmservice search mass_event_staffs '{"$eq":["mass_event_id","RELATED_ID"]}'` |
| mass_events | header_file | files | hasOne | `crmservice search mass_events '{"$eq":["header_file_id","RELATED_ID"]}'` |
| mass_events | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["mass_event_id","PARENT_ID"]}'` |
| mass_events | organization | organizations | hasOne | `crmservice search mass_events '{"$eq":["organization_id","RELATED_ID"]}'` |
| mass_events | payments | payments | hasMany | `crmservice search payments '{"$eq":["mass_event_id","PARENT_ID"]}'` |
| mass_events | product | products | hasOne | `crmservice search mass_events '{"$eq":["product_id","RELATED_ID"]}'` |
| mass_events | products | mass_event_products | hasMany | `crmservice search mass_event_products '{"$eq":["mass_event_id","PARENT_ID"]}'` |
| mass_events | receipts | receipts | hasMany | `crmservice search receipts '{"$eq":["mass_event_id","PARENT_ID"]}'` |
| mass_events | registrations | mass_event_registrations | hasMany | `crmservice search mass_event_registrations '{"$eq":["mass_event_id","PARENT_ID"]}'` |
| mass_events | rows | mass_event_rows | hasMany | `crmservice search mass_event_rows '{"$eq":["mass_event_id","PARENT_ID"]}'` |
| mass_events | space | spaces | hasOne | `crmservice search mass_events '{"$eq":["space_id","RELATED_ID"]}'` |
| mass_events | staffs | mass_event_staffs | hasMany | `crmservice search mass_event_staffs '{"$eq":["mass_event_id","PARENT_ID"]}'` |
| mass_mailings | campaign | campaigns | hasOne | `crmservice search mass_mailings '{"$eq":["campaign_id","RELATED_ID"]}'` |
| mass_mailings | email_template | email_templates | hasOne | `crmservice search mass_mailings '{"$eq":["email_template_id","RELATED_ID"]}'` |
| menus | items | menu_items | hasMany | `crmservice search menu_items '{"$eq":["parent_id","PARENT_ID"]}'` |
| notifications | creator | users | hasOne | `crmservice search notifications '{"$eq":["creator_id","RELATED_ID"]}'` |
| notifications | target_account | accounts | hasOne | `crmservice search notifications '{"$eq":["target_id","RELATED_ID"]}'` |
| notifications | target_user | users | hasOne | `crmservice search notifications '{"$eq":["target_id","RELATED_ID"]}'` |
| notifications | updater | users | hasOne | `crmservice search notifications '{"$eq":["updater_id","RELATED_ID"]}'` |
| organizations | account | accounts | hasOne | `crmservice search organizations '{"$eq":["account_id","RELATED_ID"]}'` |
| organizations | mass_events | mass_events | hasMany | `crmservice search mass_events '{"$eq":["organization_id","PARENT_ID"]}'` |
| organizations | payments | payments | hasMany | `crmservice search payments '{"$eq":["organization_id","PARENT_ID"]}'` |
| organizations | price_books | price_books | hasMany (many-to-many) | `crmservice search organizations '{"$eq":["price_books.id","VALUE"]}' --include price_books` |
| organizations | product | products | hasOne | `crmservice search organizations '{"$eq":["product_id","RELATED_ID"]}'` |
| organizations | relations | relations | hasMany | `crmservice search relations '{"$eq":["organization_id","PARENT_ID"]}'` |
| organizations | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["organization_id","PARENT_ID"]}'` |
| organizations | users | users | hasMany | `crmservice search users '{"$eq":["organization_id","PARENT_ID"]}'` |
| picklist_value_translations | picklist | picklists | hasOne | `crmservice search picklist_value_translations '{"$eq":["picklist_id","RELATED_ID"]}'` |
| picklist_values | picklist | picklists | hasOne | `crmservice search picklist_values '{"$eq":["picklist_id","RELATED_ID"]}'` |
| picklist_values | translation | picklist_value_translations | hasMany | `crmservice search picklist_value_translations '{"$eq":["id","PARENT_ID"]}'` |
| picklists | values | picklist_values | hasMany | `crmservice search picklist_values '{"$eq":["picklist_id","PARENT_ID"]}'` |
| portal_profiles | portal | portals | hasOne | `crmservice search portal_profiles '{"$eq":["portal_id","RELATED_ID"]}'` |
| portal_users | contact | contacts | hasOne | `crmservice search portal_users '{"$eq":["contact_id","RELATED_ID"]}'` |
| portal_users | portal | portals | hasOne | `crmservice search portal_users '{"$eq":["portal_id","RELATED_ID"]}'` |
| portal_users | profile | portal_profiles | hasOne | `crmservice search portal_users '{"$eq":["profile_id","RELATED_ID"]}'` |
| portals | users | portal_users | hasMany | `crmservice search portal_users '{"$eq":["portal_id","PARENT_ID"]}'` |
| potentials | account | accounts | hasOne | `crmservice search potentials '{"$eq":["account_id","RELATED_ID"]}'` |
| potentials | activities | activities | hasMany (many-to-many) | `crmservice search potentials '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| potentials | campaign | campaigns | hasOne | `crmservice search potentials '{"$eq":["campaign_id","RELATED_ID"]}'` |
| potentials | contact | contacts | hasOne | `crmservice search potentials '{"$eq":["contact_id","RELATED_ID"]}'` |
| potentials | contacts | contacts | hasMany | `crmservice search contacts '{"$eq":["contact_id","PARENT_ID"]}'` |
| potentials | documents | documents | hasMany (many-to-many) | `crmservice search potentials '{"$eq":["documents.id","VALUE"]}' --include documents` |
| potentials | emails | emails | hasMany | `crmservice search emails '{"$eq":["potential_id","PARENT_ID"]}'` |
| potentials | payments | payments | hasMany | `crmservice search payments '{"$eq":["potential_id","PARENT_ID"]}'` |
| potentials | product | products | hasOne | `crmservice search potentials '{"$eq":["product_id","RELATED_ID"]}'` |
| potentials | products | products | hasMany (many-to-many) | `crmservice search potentials '{"$eq":["products.id","VALUE"]}' --include products` |
| potentials | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["potential_id","PARENT_ID"]}'` |
| potentials | relations | relations | hasMany | `crmservice search relations '{"$eq":["potential_id","PARENT_ID"]}'` |
| potentials | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["potential_id","PARENT_ID"]}'` |
| price_books | accounts | accounts | hasMany (many-to-many) | `crmservice search price_books '{"$eq":["accounts.id","VALUE"]}' --include accounts` |
| price_books | contacts | contacts | hasMany (many-to-many) | `crmservice search price_books '{"$eq":["contacts.id","VALUE"]}' --include contacts` |
| price_books | organizations | organizations | hasMany (many-to-many) | `crmservice search price_books '{"$eq":["organizations.id","VALUE"]}' --include organizations` |
| price_books | product_prices | product_prices | hasMany | `crmservice search product_prices '{"$eq":["price_book_id","PARENT_ID"]}'` |
| price_books | products | products | hasMany (many-to-many) | `crmservice search price_books '{"$eq":["products.id","VALUE"]}' --include products` |
| product_bundles | parent | products | hasMany | `crmservice search products '{"$eq":["productid","PARENT_ID"]}'` |
| product_bundles | product | products | hasMany | `crmservice search products '{"$eq":["crmid","PARENT_ID"]}'` |
| products | accounts | accounts | hasMany (many-to-many) | `crmservice search products '{"$eq":["accounts.id","VALUE"]}' --include accounts` |
| products | campaigns | campaigns | hasMany (many-to-many) | `crmservice search products '{"$eq":["campaigns.id","VALUE"]}' --include campaigns` |
| products | contacts | contacts | hasMany (many-to-many) | `crmservice search products '{"$eq":["contacts.id","VALUE"]}' --include contacts` |
| products | contract_rows | contract2_rows | hasMany | `crmservice search contract2_rows '{"$eq":["product_id","PARENT_ID"]}'` |
| products | documents | documents | hasMany (many-to-many) | `crmservice search products '{"$eq":["documents.id","VALUE"]}' --include documents` |
| products | expenses | project_expenses | hasMany | `crmservice search project_expenses '{"$eq":["product_id","PARENT_ID"]}'` |
| products | faqs | faqs | hasMany | `crmservice search faqs '{"$eq":["product_id","PARENT_ID"]}'` |
| products | invoice_rows | invoice_rows | hasMany | `crmservice search invoice_rows '{"$eq":["product_id","PARENT_ID"]}'` |
| products | leads | leads | hasMany (many-to-many) | `crmservice search products '{"$eq":["leads.id","VALUE"]}' --include leads` |
| products | organizations | organizations | hasMany | `crmservice search organizations '{"$eq":["product_id","PARENT_ID"]}'` |
| products | potentials | potentials | hasMany (many-to-many) | `crmservice search products '{"$eq":["potentials.id","VALUE"]}' --include potentials` |
| products | price_books | price_books | hasMany (many-to-many) | `crmservice search products '{"$eq":["price_books.id","VALUE"]}' --include price_books` |
| products | product_taxes | product_taxs | hasMany (many-to-many) | `crmservice search products '{"$eq":["product_taxes.id","VALUE"]}' --include product_taxes` |
| products | products | products | hasMany (many-to-many) | `crmservice search products '{"$eq":["products.id","VALUE"]}' --include products` |
| products | projects | projects | hasMany (many-to-many) | `crmservice search products '{"$eq":["projects.id","VALUE"]}' --include projects` |
| products | purchase_order_rows | purchase_order_rows | hasMany | `crmservice search purchase_order_rows '{"$eq":["product_id","PARENT_ID"]}'` |
| products | quote_rows | quote_rows | hasMany | `crmservice search quote_rows '{"$eq":["product_id","PARENT_ID"]}'` |
| products | sales_order_rows | sales_order_rows | hasMany | `crmservice search sales_order_rows '{"$eq":["product_id","PARENT_ID"]}'` |
| products | sales_orders | sales_orders | hasMany (many-to-many) | `crmservice search products '{"$eq":["sales_orders.id","VALUE"]}' --include sales_orders` |
| products | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["product_id","PARENT_ID"]}'` |
| products | stocks | warehouse_product_stocks | hasMany | `crmservice search warehouse_product_stocks '{"$eq":["product_id","PARENT_ID"]}'` |
| products | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["product_id","PARENT_ID"]}'` |
| products | vendor | vendors | hasOne | `crmservice search products '{"$eq":["vendor_id","RELATED_ID"]}'` |
| products | vendors | vendors | hasMany (many-to-many) | `crmservice search products '{"$eq":["vendors.id","VALUE"]}' --include vendors` |
| project_expenses | account | accounts | hasOne | `crmservice search project_expenses '{"$eq":["account_id","RELATED_ID"]}'` |
| project_expenses | creator | users | hasOne | `crmservice search project_expenses '{"$eq":["creator_id","RELATED_ID"]}'` |
| project_expenses | product | products | hasOne | `crmservice search project_expenses '{"$eq":["product_id","RELATED_ID"]}'` |
| project_expenses | project | projects | hasOne | `crmservice search project_expenses '{"$eq":["project_id","RELATED_ID"]}'` |
| project_expenses | row | project_rows | hasOne | `crmservice search project_expenses '{"$eq":["row_id","RELATED_ID"]}'` |
| project_expenses | user | users | hasOne | `crmservice search project_expenses '{"$eq":["user_id","RELATED_ID"]}'` |
| project_rows | expenses | project_expenses | hasMany | `crmservice search project_expenses '{"$eq":["row_id","PARENT_ID"]}'` |
| project_rows | product | products | hasOne | `crmservice search project_rows '{"$eq":["product_id","RELATED_ID"]}'` |
| project_rows | project | projects | hasOne | `crmservice search project_rows '{"$eq":["project_id","RELATED_ID"]}'` |
| project_rows | staff_assignments | project_staff_assignments | hasMany (many-to-many) | `crmservice search project_rows '{"$eq":["staff_assignments.id","VALUE"]}' --include staff_assignments` |
| project_staff_assignments | project | projects | hasOne | `crmservice search project_staff_assignments '{"$eq":["project_id","RELATED_ID"]}'` |
| project_staff_assignments | row | project_rows | hasOne | `crmservice search project_staff_assignments '{"$eq":["row_id","RELATED_ID"]}'` |
| project_staff_assignments | user | users | hasOne | `crmservice search project_staff_assignments '{"$eq":["user_id","RELATED_ID"]}'` |
| project_staffs | project | projects | hasOne | `crmservice search project_staffs '{"$eq":["project_id","RELATED_ID"]}'` |
| project_staffs | user | users | hasOne | `crmservice search project_staffs '{"$eq":["user_id","RELATED_ID"]}'` |
| projects | account | accounts | hasOne | `crmservice search projects '{"$eq":["account_id","RELATED_ID"]}'` |
| projects | accounts | accounts | hasMany (many-to-many) | `crmservice search projects '{"$eq":["accounts.id","VALUE"]}' --include accounts` |
| projects | activities | activities | hasMany | `crmservice search activities '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | assignments | project_staff_assignments | hasMany (many-to-many) | `crmservice search projects '{"$eq":["assignments.id","VALUE"]}' --include assignments` |
| projects | contact | contacts | hasOne | `crmservice search projects '{"$eq":["contact_id","RELATED_ID"]}'` |
| projects | contract | contracts | hasOne | `crmservice search projects '{"$eq":["contract_id","RELATED_ID"]}'` |
| projects | documents | documents | hasMany (many-to-many) | `crmservice search projects '{"$eq":["documents.id","VALUE"]}' --include documents` |
| projects | emails | emails | hasMany | `crmservice search emails '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | end_account | accounts | hasOne | `crmservice search projects '{"$eq":["end_account_id","RELATED_ID"]}'` |
| projects | expenses | project_expenses | hasMany | `crmservice search project_expenses '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | payments | payments | hasMany | `crmservice search payments '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | product | products | hasOne | `crmservice search projects '{"$eq":["product_id","RELATED_ID"]}'` |
| projects | products | products | hasMany (many-to-many) | `crmservice search projects '{"$eq":["products.id","VALUE"]}' --include products` |
| projects | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | relations | relations | hasMany | `crmservice search relations '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | rows | project_rows | hasMany | `crmservice search project_rows '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["project_id","PARENT_ID"]}'` |
| projects | staffs | users | hasMany (many-to-many) | `crmservice search projects '{"$eq":["staffs.id","VALUE"]}' --include staffs` |
| purchase_order_rows | purchase_order | purchase_orders | hasOne | `crmservice search purchase_order_rows '{"$eq":["entity_id","RELATED_ID"]}'` |
| purchase_orders | account | accounts | hasOne | `crmservice search purchase_orders '{"$eq":["account_id","RELATED_ID"]}'` |
| purchase_orders | activities | activities | hasMany (many-to-many) | `crmservice search purchase_orders '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| purchase_orders | contact | contacts | hasOne | `crmservice search purchase_orders '{"$eq":["contact_id","RELATED_ID"]}'` |
| purchase_orders | documents | documents | hasMany (many-to-many) | `crmservice search purchase_orders '{"$eq":["documents.id","VALUE"]}' --include documents` |
| purchase_orders | project | projects | hasOne | `crmservice search purchase_orders '{"$eq":["project_id","RELATED_ID"]}'` |
| purchase_orders | quote | quotes | hasOne | `crmservice search purchase_orders '{"$eq":["quote_id","RELATED_ID"]}'` |
| purchase_orders | rows | purchase_order_rows | hasMany | `crmservice search purchase_order_rows '{"$eq":["entity_id","PARENT_ID"]}'` |
| purchase_orders | vendor | vendors | hasOne | `crmservice search purchase_orders '{"$eq":["vendor_id","RELATED_ID"]}'` |
| quote_rows | quote | quotes | hasOne | `crmservice search quote_rows '{"$eq":["entity_id","RELATED_ID"]}'` |
| quotes | account | accounts | hasOne | `crmservice search quotes '{"$eq":["account_id","RELATED_ID"]}'` |
| quotes | activities | activities | hasMany (many-to-many) | `crmservice search quotes '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| quotes | campaign | campaigns | hasOne | `crmservice search quotes '{"$eq":["campaign_id","RELATED_ID"]}'` |
| quotes | contact | contacts | hasOne | `crmservice search quotes '{"$eq":["contact_id","RELATED_ID"]}'` |
| quotes | documents | documents | hasMany (many-to-many) | `crmservice search quotes '{"$eq":["documents.id","VALUE"]}' --include documents` |
| quotes | emails | emails | hasMany | `crmservice search emails '{"$eq":["quote_id","PARENT_ID"]}'` |
| quotes | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["quote_id","PARENT_ID"]}'` |
| quotes | payments | payments | hasMany | `crmservice search payments '{"$eq":["quote_id","PARENT_ID"]}'` |
| quotes | potential | potentials | hasOne | `crmservice search quotes '{"$eq":["potential_id","RELATED_ID"]}'` |
| quotes | project | projects | hasOne | `crmservice search quotes '{"$eq":["project_id","RELATED_ID"]}'` |
| quotes | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["quote_id","PARENT_ID"]}'` |
| quotes | relations | relations | hasMany | `crmservice search relations '{"$eq":["quote_id","PARENT_ID"]}'` |
| quotes | rows | quote_rows | hasMany | `crmservice search quote_rows '{"$eq":["entity_id","PARENT_ID"]}'` |
| quotes | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["quote_id","PARENT_ID"]}'` |
| receipt_rows | product | products | hasOne | `crmservice search receipt_rows '{"$eq":["product_id","RELATED_ID"]}'` |
| receipt_rows | receipt | receipts | hasOne | `crmservice search receipt_rows '{"$eq":["receipt_id","RELATED_ID"]}'` |
| receipts | account | accounts | hasOne | `crmservice search receipts '{"$eq":["account_id","RELATED_ID"]}'` |
| receipts | contact | contacts | hasOne | `crmservice search receipts '{"$eq":["contact_id","RELATED_ID"]}'` |
| receipts | end_customer | accounts | hasOne | `crmservice search receipts '{"$eq":["end_customer_id","RELATED_ID"]}'` |
| receipts | invoice | invoices | hasOne | `crmservice search receipts '{"$eq":["invoice_id","RELATED_ID"]}'` |
| receipts | lead | leads | hasOne | `crmservice search receipts '{"$eq":["lead_id","RELATED_ID"]}'` |
| receipts | mass_event | mass_events | hasOne | `crmservice search receipts '{"$eq":["mass_event_id","RELATED_ID"]}'` |
| receipts | rows | receipt_rows | hasMany | `crmservice search receipt_rows '{"$eq":["receipt_id","PARENT_ID"]}'` |
| sales_channels | accounts | accounts | hasMany | `crmservice search accounts '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | activities | activities | hasMany | `crmservice search activities '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | campaigns | campaigns | hasMany | `crmservice search campaigns '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | client_responsability_unit | sales_channels | hasOne | `crmservice search sales_channels '{"$eq":["source_id","RELATED_ID"]}'` |
| sales_channels | contacts | contacts | hasMany | `crmservice search contacts '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | contracts | contracts | hasMany | `crmservice search contracts '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | documents | documents | hasMany | `crmservice search documents '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | emails | emails | hasMany | `crmservice search emails '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | faqs | faqs | hasMany | `crmservice search faqs '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | leads | leads | hasMany | `crmservice search leads '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | price_books | price_books | hasMany | `crmservice search price_books '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | products | products | hasMany | `crmservice search products '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | projects | projects | hasMany | `crmservice search projects '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | sales_groups | sales_groups | hasMany | `crmservice search sales_groups '{"$eq":["target_id","PARENT_ID"]}'` |
| sales_channels | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | target_groups | target_groups | hasMany | `crmservice search target_groups '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | users | users | hasMany | `crmservice search users '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_channels | vendors | vendors | hasMany | `crmservice search vendors '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | accounts | accounts | hasMany | `crmservice search accounts '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | activities | activities | hasMany | `crmservice search activities '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | campaigns | campaigns | hasMany | `crmservice search campaigns '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | contacts | contacts | hasMany | `crmservice search contacts '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | contracts | contracts | hasMany | `crmservice search contracts '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | documents | documents | hasMany | `crmservice search documents '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | emails | emails | hasMany | `crmservice search emails '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | faqs | faqs | hasMany | `crmservice search faqs '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | leads | leads | hasMany | `crmservice search leads '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | list_views | list_viewes | hasMany | `crmservice search list_viewes '{"$eq":["list_view_id","PARENT_ID"]}'` |
| sales_groups | potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | price_books | price_books | hasMany | `crmservice search price_books '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | products | products | hasMany | `crmservice search products '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | projects | projects | hasMany | `crmservice search projects '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | sales_channel | sales_channels | hasOne | `crmservice search sales_groups '{"$eq":["source_id","RELATED_ID"]}'` |
| sales_groups | sales_locations | sales_locations | hasMany | `crmservice search sales_locations '{"$eq":["target_id","PARENT_ID"]}'` |
| sales_groups | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | target_groups | target_groups | hasMany | `crmservice search target_groups '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | users | users | hasMany | `crmservice search users '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_groups | vendors | vendors | hasMany | `crmservice search vendors '{"$eq":["sales_group_id","PARENT_ID"]}'` |
| sales_locations | accounts | accounts | hasMany | `crmservice search accounts '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | activities | activities | hasMany | `crmservice search activities '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | campaigns | campaigns | hasMany | `crmservice search campaigns '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | contacts | contacts | hasMany | `crmservice search contacts '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | contracts | contracts | hasMany | `crmservice search contracts '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | documents | documents | hasMany | `crmservice search documents '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | emails | emails | hasMany | `crmservice search emails '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | faqs | faqs | hasMany | `crmservice search faqs '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | leads | leads | hasMany | `crmservice search leads '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | price_books | price_books | hasMany | `crmservice search price_books '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | products | products | hasMany | `crmservice search products '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | projects | projects | hasMany | `crmservice search projects '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | sales_group | sales_groups | hasOne | `crmservice search sales_locations '{"$eq":["source_id","RELATED_ID"]}'` |
| sales_locations | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | target_groups | target_groups | hasMany | `crmservice search target_groups '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | users | users | hasMany | `crmservice search users '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_locations | vendors | vendors | hasMany | `crmservice search vendors '{"$eq":["sales_location_id","PARENT_ID"]}'` |
| sales_order_rows | sales_order | sales_orders | hasOne | `crmservice search sales_order_rows '{"$eq":["entity_id","RELATED_ID"]}'` |
| sales_orders | account | accounts | hasOne | `crmservice search sales_orders '{"$eq":["account_id","RELATED_ID"]}'` |
| sales_orders | activities | activities | hasMany (many-to-many) | `crmservice search sales_orders '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| sales_orders | campaign | campaigns | hasOne | `crmservice search sales_orders '{"$eq":["campaign_id","RELATED_ID"]}'` |
| sales_orders | contact | contacts | hasOne | `crmservice search sales_orders '{"$eq":["contact_id","RELATED_ID"]}'` |
| sales_orders | documents | documents | hasMany (many-to-many) | `crmservice search sales_orders '{"$eq":["documents.id","VALUE"]}' --include documents` |
| sales_orders | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["salesorder_id","PARENT_ID"]}'` |
| sales_orders | payments | payments | hasMany | `crmservice search payments '{"$eq":["salesorder_id","PARENT_ID"]}'` |
| sales_orders | potential | potentials | hasOne | `crmservice search sales_orders '{"$eq":["potential_id","RELATED_ID"]}'` |
| sales_orders | products | products | hasMany (many-to-many) | `crmservice search sales_orders '{"$eq":["products.id","VALUE"]}' --include products` |
| sales_orders | project | projects | hasOne | `crmservice search sales_orders '{"$eq":["project_id","RELATED_ID"]}'` |
| sales_orders | quote | quotes | hasOne | `crmservice search sales_orders '{"$eq":["quote_id","RELATED_ID"]}'` |
| sales_orders | relations | relations | hasMany | `crmservice search relations '{"$eq":["salesorder_id","PARENT_ID"]}'` |
| sales_orders | rows | sales_order_rows | hasMany | `crmservice search sales_order_rows '{"$eq":["entity_id","PARENT_ID"]}'` |
| sales_orders | target_template_list | list_viewes | hasOne | `crmservice search sales_orders '{"$eq":["target_template_list_id","RELATED_ID"]}'` |
| sales_orders | vendor | vendors | hasOne | `crmservice search sales_orders '{"$eq":["vendor_id","RELATED_ID"]}'` |
| spaces | account | accounts | hasOne | `crmservice search spaces '{"$eq":["account_id","RELATED_ID"]}'` |
| spaces | activities | activities | hasMany | `crmservice search activities '{"$eq":["space_id","PARENT_ID"]}'` |
| spaces | contact | contacts | hasOne | `crmservice search spaces '{"$eq":["contact_id","RELATED_ID"]}'` |
| spaces | contract_rows | contract2_rows | hasMany | `crmservice search contract2_rows '{"$eq":["product_id","PARENT_ID"]}'` |
| spaces | organization | organizations | hasOne | `crmservice search spaces '{"$eq":["organization_id","RELATED_ID"]}'` |
| spaces | product | products | hasOne | `crmservice search spaces '{"$eq":["product_id","RELATED_ID"]}'` |
| spaces | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["space_id","PARENT_ID"]}'` |
| survey_answers | contact | contacts | hasOne | `crmservice search survey_answers '{"$eq":["contact_id","RELATED_ID"]}'` |
| survey_answers | source_entity | entities | hasOne | `crmservice search survey_answers '{"$eq":["source_entity_id","RELATED_ID"]}'` |
| survey_answers | survey | surveys | hasOne | `crmservice search survey_answers '{"$eq":["survey_id","RELATED_ID"]}'` |
| surveys | answers | survey_answers | hasMany | `crmservice search survey_answers '{"$eq":["survey_id","PARENT_ID"]}'` |
| target_groups | accounts | accounts | hasMany (many-to-many) | `crmservice search target_groups '{"$eq":["accounts.id","VALUE"]}' --include accounts` |
| target_groups | contacts | contacts | hasMany (many-to-many) | `crmservice search target_groups '{"$eq":["contacts.id","VALUE"]}' --include contacts` |
| tickets | account | accounts | hasOne | `crmservice search tickets '{"$eq":["account_id","RELATED_ID"]}'` |
| tickets | activities | activities | hasMany (many-to-many) | `crmservice search tickets '{"$eq":["activities.subject","VALUE"]}' --include activities` |
| tickets | contact | contacts | hasOne | `crmservice search tickets '{"$eq":["contact_id","RELATED_ID"]}'` |
| tickets | documents | documents | hasMany (many-to-many) | `crmservice search tickets '{"$eq":["documents.id","VALUE"]}' --include documents` |
| tickets | handler | users | hasOne | `crmservice search tickets '{"$eq":["handler_id","RELATED_ID"]}'` |
| tickets | product | products | hasOne | `crmservice search tickets '{"$eq":["product_id","RELATED_ID"]}'` |
| tickets | project | projects | hasOne | `crmservice search tickets '{"$eq":["project_id","RELATED_ID"]}'` |
| tickets | reseller | accounts | hasOne | `crmservice search tickets '{"$eq":["reseller_id","RELATED_ID"]}'` |
| tickets | space | spaces | hasOne | `crmservice search tickets '{"$eq":["space_id","RELATED_ID"]}'` |
| tickets | vendor | vendors | hasOne | `crmservice search tickets '{"$eq":["vendor_id","RELATED_ID"]}'` |
| triggers | creator | users | hasOne | `crmservice search triggers '{"$eq":["creator_id","RELATED_ID"]}'` |
| user_settings | user | users | hasOne | `crmservice search user_settings '{"$eq":["user_id","RELATED_ID"]}'` |
| users | accounts | accounts | hasMany | `crmservice search accounts '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | activities | activities | hasMany | `crmservice search activities '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | campaigns | campaigns | hasMany | `crmservice search campaigns '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | client_responsability_unit | client_responsability_units | hasOne | `crmservice search users '{"$eq":["client_responsability_unit_id","RELATED_ID"]}'` |
| users | contacts | contacts | hasMany | `crmservice search contacts '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | contracts | contracts | hasMany | `crmservice search contracts '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | created_accounts | accounts | hasMany | `crmservice search accounts '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_activities | activities | hasMany | `crmservice search activities '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_campaigns | campaigns | hasMany | `crmservice search campaigns '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_contacts | contacts | hasMany | `crmservice search contacts '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_contracts | contracts | hasMany | `crmservice search contracts '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_documents | documents | hasMany | `crmservice search documents '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_emails | emails | hasMany | `crmservice search emails '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_faqs | faqs | hasMany | `crmservice search faqs '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_files | files | hasMany | `crmservice search files '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_leads | leads | hasMany | `crmservice search leads '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_price_books | price_books | hasMany | `crmservice search price_books '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_products | products | hasMany | `crmservice search products '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_projects | projects | hasMany | `crmservice search projects '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_target_groups | target_groups | hasMany | `crmservice search target_groups '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | created_vendors | vendors | hasMany | `crmservice search vendors '{"$eq":["creator_id","PARENT_ID"]}'` |
| users | documents | documents | hasMany | `crmservice search documents '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | emails | emails | hasMany | `crmservice search emails '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | faqs | faqs | hasMany | `crmservice search faqs '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | files | files | hasMany | `crmservice search files '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | invoices | invoices | hasMany | `crmservice search invoices '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | leads | leads | hasMany | `crmservice search leads '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | organization | organizations | hasOne | `crmservice search users '{"$eq":["organization_id","RELATED_ID"]}'` |
| users | potentials | potentials | hasMany | `crmservice search potentials '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | price_books | price_books | hasMany | `crmservice search price_books '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | products | products | hasMany | `crmservice search products '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | project_assignments | project_rows | hasMany (many-to-many) | `crmservice search users '{"$eq":["project_assignments.id","VALUE"]}' --include project_assignments` |
| users | projects | projects | hasMany | `crmservice search projects '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | projects_staff | projects | hasMany (many-to-many) | `crmservice search users '{"$eq":["projects_staff.id","VALUE"]}' --include projects_staff` |
| users | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | quotes | quotes | hasMany | `crmservice search quotes '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | reports_to | users | hasOne | `crmservice search users '{"$eq":["reports_to_id","RELATED_ID"]}'` |
| users | sales_channel | sales_channels | hasOne | `crmservice search users '{"$eq":["sales_channel_id","RELATED_ID"]}'` |
| users | sales_group | sales_groups | hasOne | `crmservice search users '{"$eq":["sales_group_id","RELATED_ID"]}'` |
| users | sales_location | sales_locations | hasOne | `crmservice search users '{"$eq":["sales_location_id","RELATED_ID"]}'` |
| users | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | spaces | spaces | hasMany | `crmservice search spaces '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | subordinates | users | hasMany | `crmservice search users '{"$eq":["reports_to_id","PARENT_ID"]}'` |
| users | target_groups | target_groups | hasMany | `crmservice search target_groups '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["owner_id","PARENT_ID"]}'` |
| users | vendors | vendors | hasMany | `crmservice search vendors '{"$eq":["owner_id","PARENT_ID"]}'` |
| vendors | campaigns | campaigns | hasMany | `crmservice search campaigns '{"$eq":["vendor_id","PARENT_ID"]}'` |
| vendors | contacts | contacts | hasMany (many-to-many) | `crmservice search vendors '{"$eq":["contacts.id","VALUE"]}' --include contacts` |
| vendors | products | products | hasMany (many-to-many) | `crmservice search vendors '{"$eq":["products.id","VALUE"]}' --include products` |
| vendors | purchase_orders | purchase_orders | hasMany | `crmservice search purchase_orders '{"$eq":["vendor_id","PARENT_ID"]}'` |
| vendors | sales_orders | sales_orders | hasMany | `crmservice search sales_orders '{"$eq":["vendor_id","PARENT_ID"]}'` |
| vendors | tickets | tickets | hasMany | `crmservice search tickets '{"$eq":["vendor_id","PARENT_ID"]}'` |
| warehouse_product_stock_changes | entity | entities | hasOne | `crmservice search warehouse_product_stock_changes '{"$eq":["entity_id","RELATED_ID"]}'` |
| warehouse_product_stock_changes | product | products | hasOne | `crmservice search warehouse_product_stock_changes '{"$eq":["product_id","RELATED_ID"]}'` |
| warehouse_product_stock_changes | warehouse | warehouses | hasOne | `crmservice search warehouse_product_stock_changes '{"$eq":["warehouse_id","RELATED_ID"]}'` |
| warehouse_product_stocks | product | products | hasOne | `crmservice search warehouse_product_stocks '{"$eq":["product_id","RELATED_ID"]}'` |
| warehouse_product_stocks | warehouse | warehouses | hasOne | `crmservice search warehouse_product_stocks '{"$eq":["warehouse_id","RELATED_ID"]}'` |
| warehouses | stocks | warehouse_product_stocks | hasMany | `crmservice search warehouse_product_stocks '{"$eq":["warehouse_id","PARENT_ID"]}'` |
