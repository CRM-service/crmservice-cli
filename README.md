# CRM-service CLI

A command-line tool for interacting with the CRM-service REST API.

## Installation

```bash
go install github.com/example/crmservice@latest
```

## Configuration

### Config File

Create `~/.config/crmservice/config.yaml`:

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
--output     Output format: table or json
--page-size  Items per page
--timeout    Request timeout
--cache-dir  Schema cache directory
```

## Usage

### List Records

```bash
# List all accounts
crmservice accounts list

# With pagination
crmservice accounts list --page 2 --page-size 50

# With fields selection
crmservice accounts list --fields name,email,phone

# With filters
crmservice accounts list --filter "name=Test" --filter "status=Active"

# Include related data
crmservice accounts list --include contacts

# JSON output
crmservice accounts list --output json
```

### Get Record

```bash
crmservice accounts get <account-id>
crmservice accounts get <id> --fields name,email
```

### Create Record

```bash
crmservice accounts create \
  --field name="Test Corp" \
  --field email="test@example.com" \
  --field phone="123-456-7890"
```

### Update Record

```bash
crmservice accounts update <id> --field name="New Name"
```

### Delete Record

```bash
crmservice accounts delete <id>
```

### Show Fields

```bash
crmservice accounts fields
```

### Search

```bash
crmservice accounts search "query string"
```

### Discover Modules

```bash
crmservice modules
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
