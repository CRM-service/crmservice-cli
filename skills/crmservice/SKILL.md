---
name: crmservice
description: REQUIRED for interacting with the CRM-service REST API via the crmservice CLI tool. Use when working with CRM data operations including creating, reading, updating, and deleting records across CRM modules. Triggers: CRM record management, data queries, API interactions, module exploration, record field operations, and CRM data synchronization.
---

# crmservice CLI Tool Skill

This skill provides guidance for using the `crmservice` CLI tool to interact with the CRM-service REST API.

## Overview

CRM-service CLI is a command-line tool for interacting with the CRM-service REST API. It provides commands for managing CRM records, listing modules, searching data, and exploring field schemas.

## Base Commands

| Command | Description |
|---------|-------------|
| `completion` | Generate autocompletion for bash, fish, powershell, or zsh |
| `create` | Create a new record in a module |
| `delete` | Delete a record by ID |
| `fields` | Show available fields for a module |
| `get` | Get a single record by ID |
| `list` | List records with pagination |
| `modules` | List available API modules |
| `search` | Search records by query |
| `update` | Update an existing record |

## Global Flags

All commands support these global flags:

- `--config string`: Config file path
- `--token string`: Bearer token (overrides config)
- `--url string`: API base URL (overrides config)
- `--verbose int`: Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)

## Common Flags

Most data commands support:

- `-o, --output string`: Output format (table, json, yaml, csv)
- `--full`: Include full response (not just attributes)

## Usage Examples

### List Modules
```bash
crmservice modules
crmservice modules --output json
```

### List Records
```bash
crmservice list accounts
crmservice list accounts --page 2 --page-size 50
crmservice list accounts --fields "name,id,status"
crmservice accounts list --filter '{$and:[{$eq:["status","value"]}]}'
crmservice accounts list --filter '{"$or":[{"$eq":["id","123"]},{"$eq":["id","456"]}]}'
```

### Get Record
```bash
crmservice get accounts 123
crmservice get accounts 123 --fields "name,email"
```

### Create Record
```bash
crmservice create accounts --field "name=Acme Corp" --field "status=active"
```

### Update Record
```bash
crmservice update accounts 123 --field "status=inactive" --field "notes=Updated"
```

### Delete Record
```bash
crmservice delete accounts 123
```

### Search Records
```bash
crmservice search accounts '{$and:[{$eq:["status","value"]}]}'
crmservice search contacts '{"$or":[{"$eq":["id","123"]},{"$eq":["id","456"]}]}' --fields "id,entity_no,first_name,last_name"
```

### Show Fields
```bash
crmservice fields accounts
crmservice fields accounts --output json
```

### Completion Setup
```bash
# Bash
crmservice completion bash > ~/.local/share/bash-completion/completions/crmservice

# Zsh
crmservice completion zsh > ~/.zcompletions/_crmservice

# Fish
crmservice completion fish > ~/.config/fish/completions/crmservice.fish
```

## Configuration

Set environment variables or use a config file:

- `CRMSERVICE_API_URL`: API base URL
- `CRMSERVICE_AUTH_TOKEN`: Bearer token for authentication

## Notes

- Module names are case-sensitive
- To get available modules use the modules command
- To get available fields for module use the fields [module] command. This schema also defines the datatype for the field
- All fields have name and label. API call must always use the name
- Schema defines the datatype fields. Invalid values must never be sent to the API
- Field schema may contain custom fields. Name prefixed with `cf_`
- Field values for create/update use `--field "name=value"` syntax
- Output formats support table (default), json, yaml, and csv
- Pagination uses --page and --page-size or --offset
- Authentication must always be available by either: config file, environment variable or command-line argument
