# CRM-service CLI

A command-line tool for interacting with the CRM-service REST API.

## Installation

```bash
go install github.com/example/crmservice@latest
```

## Configuration

### Config File

Create a YAML config file such as `~/.config/crmservice/config.yaml`:

```yaml
api:
  url: "https://customer.example.com/api/v1"
  timeout: 30

output:
  format: "table"
  page_size: 20

cache:
  schema_dir: "~/.cache/crmservice/schema"
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
CRMSERVICE_OUTPUT_FORMAT  Output format (table or json)
CRMSERVICE_PAGE_SIZE      Default page size
CRMSERVICE_TIMEOUT        Request timeout in seconds
```

### Command-line Flags

Flags override config file and environment variables:

```
--url        API base URL
--token      Bearer token
--output     Output format: table, json, yaml, or csv
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

# With a JSON filter
crmservice list accounts --filter '{"$and":[{"$eq":["name","Test Corp"]},{"$eq":["status","Active"]}]}'

# Include related data
crmservice list accounts --include contacts

# JSON output
crmservice list accounts --output json
```

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
```

### Update Record

```bash
crmservice update accounts <id> --field name="New Name"
```

### Delete Record

```bash
crmservice delete accounts <id>
```

### Show Fields

```bash
crmservice fields accounts
```

### Search

`search` is an alias for `list --filter`; pass the JSON filter as the second positional argument.

```bash
crmservice search accounts '{"$eq":["name","Test Corp"]}'
crmservice search contacts '{"$or":[{"$eq":["id","123"]},{"$eq":["id","456"]}]}' --fields id,first_name,last_name
```

### Discover Modules

```bash
crmservice modules
```

### Show Current User

```bash
crmservice whoami
crmservice whoami --output json
```

## JSON:API Compliance

The client follows JSON:API specification (v1.0) for:
- Resource objects
- Pagination with `page` and `page_size` parameters
- Sparse fieldsets with `fields` parameter
- Filtering with `filter` parameter
- Compound documents with `include` parameter

## Schema Caching

Schemas are cached per API instance to reduce API calls and improve performance.

Cache location: `~/.cache/crmservice/schema/`

Cache TTL: 24 hours (configurable)
