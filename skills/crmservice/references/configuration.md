# Configuration and Setup

## Config file

By default, the CLI reads `config.yaml` from the operating system's user config directory:

- Linux: `~/.config/crmservice/config.yaml`
- macOS: `~/Library/Application Support/crmservice/config.yaml`
- Windows: `%AppData%\crmservice\config.yaml`

Use `--config` only to override this default path.

## Environment variables

- `CRMSERVICE_API_URL` — API base URL
- `CRMSERVICE_AUTH_TOKEN` — Bearer token for authentication
- `CRMSERVICE_OUTPUT_FORMAT` — Default output format (`table`, `json`, `yaml`, `jsonl`, or `csv`)
- `CRMSERVICE_PAGE_SIZE` — Default page size for paginated commands
- `CRMSERVICE_TIMEOUT` — Request timeout in seconds
- `CRMSERVICE_CACHE_DIR` — Schema cache directory (useful in CI and agent sandboxes)

Authentication must always be available via config file, environment variable, or `--token`.

## Agent defaults

For automated agent work, set these so every command uses machine-readable output and sensible pagination:

```bash
export CRMSERVICE_OUTPUT_FORMAT=json
export CRMSERVICE_PAGE_SIZE=100
```

Always pass `-o json` or `-o jsonl` explicitly when a specific format matters (for example JSONL pipelines). `CRMSERVICE_OUTPUT_FORMAT` applies only when `-o` / `--output` is not given.

The CLI default output is `table`, which is unsuitable for agent parsing.

## Global flags

All commands support:

- `--config string` — Config file path
- `--token string` — Bearer token (overrides config)
- `--url string` — API base URL (overrides config)
- `--verbose int` — 0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed logging

Most data commands also support `-o, --output` and `--full`.

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