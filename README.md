# rulebound

Generate static style guide websites from Vale linting packages.

## What it does

rulebound takes a directory of Vale YAML rule definitions, parses them, and produces a complete static website documenting every rule. The generated site includes taxonomy pages organized by category, rule type, and severity level, along with responsive design, sidebar navigation, and optional Pagefind client-side search.

## Quick start

```sh
rulebound build ./my-vale-package --output ./public/
```

## Requirements

- **Go 1.22+** (to build from source)
- **Hugo >= 0.128.0** (extended edition recommended)
- **Pagefind** (optional, for client-side search indexing)

## Installation

From source:

```sh
go install github.com/larah/rulebound@latest
```

Or clone the repository and use the Makefile:

```sh
make build    # compile to ./rulebound
make install  # install to $GOPATH/bin
```

## Usage

### `rulebound build <package-path>`

Build a static style guide website from a Vale rule package.

| Flag | Short | Default | Description |
| --- | --- | --- | --- |
| `--output` | `-o` | `./public/` | Output directory for the generated site |
| `--config` | `-c` | auto-detect | Path to `rulebound.yml` configuration file |
| `--hugo` |  | auto-detect | Path to Hugo binary |
| `--strict` |  | `false` | Treat parse warnings as errors |
| `--verbose` | `-v` | `false` | Print verbose output |

### `rulebound --version`

Print the current version and exit.

## Configuration

Place a `rulebound.yml` file in the root of your Vale package directory, or pass one explicitly with `--config`. The configuration file is optional; sensible defaults are applied when it is absent.

```yaml
title: My Style Guide
description: Documentation for our Vale linting rules
baseURL: https://styleguide.example.com/
categories:
  punctuation:
    - Punctuation.Comma
    - Punctuation.Period
  casing:
    - Casing.HeadingTitle
```

| Field | Default | Description |
| --- | --- | --- |
| `title` | Package directory name | Human-readable name of the style guide |
| `description` | (empty) | Short description displayed on the site |
| `baseURL` | `/` | Base URL for the generated Hugo site |
| `categories` | Group by rule type | Map of category names to lists of rule identifiers. A rule may appear in multiple categories. |

## How it works

1. **Parse** -- Reads all Vale YAML rule files in the package directory (supports all 11 extension types).
2. **Companion docs** -- Reads companion `.md` files alongside each rule for custom documentation content.
3. **Generate** -- Produces Hugo content files with frontmatter and taxonomy terms for each rule.
4. **Scaffold** -- Creates a complete Hugo project in a temporary directory with an embedded theme.
5. **Build** -- Runs Hugo to compile the static site into the output directory.
6. **Search index** -- Optionally runs Pagefind to generate a client-side search index.

## Supported Vale rule types

rulebound parses all 11 Vale extension types:

- `existence`
- `substitution`
- `occurrence`
- `repetition`
- `consistency`
- `conditional`
- `capitalization`
- `metric`
- `script`
- `sequence`
- `spelling`

## Exit codes

| Code | Meaning                           |
| ---- | --------------------------------- |
| 0    | Success                           |
| 1    | General error                     |
| 2    | Configuration error               |
| 3    | Hugo not found or version too old |
| 4    | Hugo build failure                |

## Project structure

```
cmd/           CLI commands (root, build)
internal/
  config/      rulebound.yml parsing
  parser/      Vale YAML rule parser
  generator/   Hugo content generation
  hugo/        Hugo scaffolding, build, theme
```

## Development

```sh
make test       # run all tests
make build      # compile binary
go test ./...   # run tests directly
```

## License

See [LICENSE](LICENSE) for details.
