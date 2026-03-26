# rulebound

Generate static style guide websites from Vale linting packages.

## What it does

rulebound takes a directory of Vale YAML rule definitions, parses them, and produces a static website documenting every rule. The site includes taxonomy pages organized by category, rule type, and severity level, along with responsive design, sidebar navigation, and optional Pagefind client-side search.

Packages can also include editorial guidelines — prose-based Markdown files in a `guidelines/` subdirectory that appear as a separate section alongside the rules.

## Quick start

Generate a style guide site from a Vale package:

```sh
rulebound build ./my-vale-package --output ./public/
```

## Requirements

rulebound requires the following dependencies:

- **Go 1.22+** (to build from source)
- **Hugo >= 0.128.0** (extended edition recommended)
- **Pagefind** (optional, for client-side search indexing)

## Installation

Install from source:

```sh
go install github.com/larah/rulebound@latest
```

Clone the repository and use the Makefile:

```sh
make build    # compile to ./rulebound
make install  # install to $GOPATH/bin
```

## Usage

### `rulebound build <package-path>`

Build a static style guide site from a Vale rule package.

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

Place a `rulebound.yml` file in the root of your Vale package directory, or pass one with `--config`. The configuration file is optional. rulebound applies defaults when it is absent.

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
guidelines:
  section_title: Editorial Guidelines
  order:
    - voice-and-tone
    - inclusive-language
  exclude:
    - draft-notes
```

The following fields are available:

| Field | Default | Description |
| --- | --- | --- |
| `title` | Package directory name | Display name of the style guide |
| `description` | (empty) | Short description displayed on the site |
| `baseURL` | `/` | Base URL for the generated Hugo site |
| `categories` | Group by rule type | Map of category names to lists of rule identifiers. A rule may appear in multiple categories. |
| `guidelines.section_title` | `Guidelines` | Sidebar heading for the guidelines section |
| `guidelines.order` | Alphabetical | Page ordering by filename stem |
| `guidelines.exclude` | (none) | Filename stems to skip (takes precedence over order) |
| `guidelines.enabled` | `true` | Set to `false` to suppress guideline generation even when files exist |

## How it works

rulebound builds the site in six stages:

1. **Parse** -- Reads all Vale YAML rule files in the package directory (supports all 11 extension types). Also reads editorial guidelines from a `guidelines/` subdirectory.
2. **Companion docs** -- Reads companion `.md` files alongside each rule for custom documentation content.
3. **Generate** -- Produces Hugo content files with frontmatter and taxonomy terms for each rule, plus guideline pages with their own layout.
4. **Scaffold** -- Creates a Hugo project in a temporary directory with an embedded theme.
5. **Build** -- Runs Hugo to compile the static site into the output directory.
6. **Search index** -- If Pagefind is installed, runs it to generate a client-side search index.

## Editorial guidelines

To add prose-based writing guidelines alongside your Vale rules, place Markdown files in a `guidelines/` subdirectory of your package:

```
my-vale-package/
├── Avoid.yml
├── Terms.yml
├── rulebound.yml
└── guidelines/
    ├── voice-and-tone.md
    └── inclusive-language.md
```

Each guideline file uses YAML frontmatter:

```markdown
---
title: "Voice and Tone"
description: "How to write in our company voice"
weight: 10
---

Write with clarity and confidence. Avoid jargon.
```

The following frontmatter fields are available:

| Field | Required | Description |
| --- | --- | --- |
| `title` | Yes | Page title displayed in sidebar and heading |
| `description` | No | Short summary shown in the guidelines index |
| `weight` | No | Sort order (lower values appear first, default: 0) |

Guidelines appear in a dedicated sidebar section and have their own index page at `/guidelines/`. rulebound skips files without a `title` in frontmatter, files with malformed YAML, and files with non-`.md` extensions. rulebound ignores subdirectories inside `guidelines/`.

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

rulebound returns the following exit codes:

| Code | Meaning                           |
| ---- | --------------------------------- |
| 0    | Success                           |
| 1    | General error                     |
| 2    | Configuration error               |
| 3    | Hugo not found or version too old |
| 4    | Hugo build failure                |

## Project structure

The repository is organized as follows:

```
cmd/           CLI commands (root, build)
internal/
  config/      rulebound.yml parsing
  parser/      Vale YAML rule parser
  generator/   Hugo content generation
  hugo/        Hugo scaffolding, build, theme
```

## Development

Run tests and build the binary with the following commands:

```sh
make test       # run all tests
make build      # compile binary
go test ./...   # run tests directly
```

## License

Refer to [LICENSE](LICENSE) for details.
