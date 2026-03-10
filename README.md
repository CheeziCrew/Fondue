# Fondue

![fondue](https://github.com/user-attachments/assets/d03ae127-72ec-4c90-b675-9ac0dd9426b7)

Melt through your microservice dependencies. Interactive TUI for scanning Maven-based Java services, extracting integration configs, and visualizing the dependency graph.

## Features

- **Interactive TUI** — browse services, drill into dependencies, fuzzy search
- **Dependency graph** — outbound integrations + reverse dependency tracking
- **Version staleness** — detects when integration specs drift from target service versions
- **Navigation history** — drill through linked services with full back-navigation
- **Multiple outputs** — TUI, table, JSON, Graphviz DOT
- **Configurable** — JSON config for repo prefixes, standalone repos, scan paths
- **Theme-aware** — uses ANSI 0-15 colors, adapts to your terminal theme (base16/Tinty)

## Install

```bash
go install github.com/CheeziCrew/fondue@latest
```

Or grab a binary from [Releases](https://github.com/CheeziCrew/fondue/releases).

## Usage

```bash
fondue                              # Interactive TUI (default path)
fondue ~/code/myproject             # Scan specific directory
fondue --list                       # Table output
fondue --json | jq '.[] | select(.dependencies | length > 5)'
fondue --dot | dot -Tpng -o graph.png && open graph.png
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--list` | `-l` | Print services as a table |
| `--json` | `-j` | Export as JSON |
| `--dot` | `-d` | Export as Graphviz DOT |
| `--version` | `-v` | Print version |
| `--help` | `-h` | Show help |

### Keybindings

**List view:**

| Key | Action |
|-----|--------|
| `↑/↓` `j/k` | Navigate |
| `/` | Fuzzy search |
| `Enter` | Open service detail |
| `q` `Ctrl+C` | Quit |

**Detail view:**

| Key | Action |
|-----|--------|
| `↑/↓` `j/k` | Navigate dependencies |
| `Enter` | Jump to linked service |
| `Esc` | Back (pops history stack) |
| `q` | Quit |

## Config (optional)

Works out of the box — defaults scan `~/code/scit/` for repos prefixed with `api-service-*` and `pw-*`.

To override, drop a `.fondue.json` in your scan directory or next to the binary. All fields are optional, only specify what you want to change:

```json
{
  "scanPath": "~/code/other-project/",
  "repoPrefixes": ["my-svc-", "backend-"],
  "standaloneRepos": {
    "my-proxy": "proxy"
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `scanPath` | `~/code/scit/` | Where to look for service repos |
| `repoPrefixes` | `["api-service-", "pw-"]` | Directory prefixes to scan (prefix is stripped to derive service name) |
| `standaloneRepos` | See `config.go` | Map of `dirName` → `serviceName` for repos that don't follow prefix conventions |

## How it works

1. Globs for directories matching configured prefixes
2. Verifies each has `src/main/java` (Maven service marker)
3. Walks `integration/` packages for `CLIENT_ID` / `INTEGRATION_NAME` constants
4. Parses `pom.xml` (XML) for project version and `<inputSpec>` references
5. Reads OpenAPI spec headers for integration version matching
6. Builds bidirectional dependency graph
7. Renders everything in a Bubbletea TUI

## License

MIT
