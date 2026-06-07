# Configuration and Setup

## API host

Use the CRM hostname, for example `customer.crmservice.fi`, in `api.url`, `CRMSERVICE_API_URL`, and `--url`.

## Precedence

For every setting: **command-line flags** override **environment variables**, which override the **config file**.

## Config file

By default, the CLI reads `config.yaml` from the operating system's user config directory:

- Linux: `~/.config/crmservice/config.yaml`
- macOS: `~/Library/Application Support/crmservice/config.yaml`
- Windows: `%AppData%\crmservice\config.yaml`

Use `--config` to override this path.

```yaml
api:
  url: "customer.crmservice.fi"
  timeout: 30

output:
  format: "table"
  page_size: 20

cache:
  schema_dir: ""
  ttl_days: 24
  auto_refresh: true

auth:
  token: "your-bearer-token"
```

| Key | Description |
|-----|-------------|
| `api.url` | CRM host (`customer.crmservice.fi`) |
| `api.timeout` | Request timeout in seconds |
| `output.format` | Default output format: `table`, `json`, `yaml`, `jsonl`, or `csv` |
| `output.page_size` | Default page size for paginated commands |
| `cache.schema_dir` | Schema cache directory (`~` paths are expanded; empty uses OS default) |
| `cache.ttl_days` | Schema cache TTL in days |
| `cache.auto_refresh` | Refresh cache when missing or expired |
| `auth.token` | Bearer token |

## Environment variables

| Variable | Description |
|----------|-------------|
| `CRMSERVICE_API_URL` | CRM host (`customer.crmservice.fi`) |
| `CRMSERVICE_AUTH_TOKEN` | Bearer token |
| `CRMSERVICE_OUTPUT_FORMAT` | Default output format |
| `CRMSERVICE_PAGE_SIZE` | Default page size |
| `CRMSERVICE_TIMEOUT` | Request timeout in seconds |
| `CRMSERVICE_CACHE_DIR` | Schema cache directory |
| `CRMSERVICE_CACHE_TTL_DAYS` | Schema cache TTL in days |
| `CRMSERVICE_CACHE_AUTO_REFRESH` | `true` or `false` |

Authentication must always be available via config file, environment variable, or `--token`.

## Global command-line flags

Available on every command:

| Flag | Description |
|------|-------------|
| `--config` | Config file path |
| `--url` | CRM host (overrides config) |
| `--token` | Bearer token (overrides config) |
| `--timeout` | Request timeout in seconds (overrides config) |
| `-o`, `--output` | Default output format (overrides config) |
| `--page-size` | Default page size (overrides config) |
| `--cache-dir` | Schema cache directory (overrides config) |
| `--cache-ttl-days` | Schema cache TTL in days (overrides config) |
| `--cache-auto-refresh` | Refresh schema cache automatically (overrides config) |

Individual commands add their own flags (for example `--filter`, `--all`, `--verbose`, `--full`). Command-level `-o` / `--output` overrides the global default when set.

## Agent defaults

For automated agent work, set these so every command uses machine-readable output and sensible pagination:

```bash
export CRMSERVICE_API_URL=customer.crmservice.fi
export CRMSERVICE_OUTPUT_FORMAT=json
export CRMSERVICE_PAGE_SIZE=100
```

Always pass `-o json` or `-o jsonl` explicitly when a specific format matters (for example JSONL pipelines). `CRMSERVICE_OUTPUT_FORMAT` applies only when `-o` / `--output` is not given.

The CLI default output is `table`, which is unsuitable for agent parsing.

## Preflight (`doctor`)

```bash
crmservice doctor -o json
```

`doctor` reports version, config path, API URL reachability, token presence, authentication, cache writability, and an `ok` boolean. Exit code is non-zero when checks fail.

## Current user (`whoami`)

```bash
crmservice whoami
crmservice whoami --output json
```

## Shell completion

```bash
# Bash
crmservice completion bash > ~/.local/share/bash-completion/completions/crmservice

# Zsh
crmservice completion zsh > ~/.zcompletions/_crmservice

# Fish
crmservice completion fish > ~/.config/fish/completions/crmservice.fish
```

## Bundled skill files

The CLI bundles this skill as multiple files under `~/.agents/skills/crmservice/`:

```bash
crmservice skill path          # path to SKILL.md entry point
crmservice skill print         # print SKILL.md
crmservice skill install       # install full skill tree
crmservice skill install --check -o json
```

`skill install --check` compares the installed skill tree with the bundled copy. Exit code `0` means up to date; non-zero means missing or stale. Run `crmservice skill install` after upgrading the CLI when `--check` reports `stale` or `missing`.

Reference files live alongside `SKILL.md` in `references/` (for example `references/filters.md`).