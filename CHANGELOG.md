# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- `-v` flag to display the current version of lsq.

## [0.10.2] - 2025-01-03
### Added
- Console messages when the supplied path for pages and journals does not exist.

### Changed
- No longer creating the main directory path when the full path does not exist.

### Fixed
- Configuration override not being applied on `-d`

## [0.10.1] - 2025-01-03
### Fixed
- Empty line appearing before text when using the append flag.

## [0.10.0] - 2025-01-01
### Added
- lsq configuration file to set user defaults instead of using flags on every run.

### Deprecated
- TUI

### Changed
- README to reflect TUI deprecation
- How the `d` flag operates. The full path must be specified when using this flag. Example: "/home/jaron/documents/notes"
- No longer using the Logseq `config.edn` file.
- `CONTRIBUTIONS.md` to be `CONTRIBUTING.md`
- `README.md`

### Removed
- appCfgFileName (-c flag) since this is set by Logseq.
- appCfgDirName (-l flag) since this is also set by Logseq.

## [0.9.0] - 2024-12-19
### Added
- Open pages with -p in the default editor.

### Changed
- Refactored the `main.go` file.
- Vim is now the default editor when $EDITOR is undefined and TUI is not used.

## [0.8.0] - 2024-12-18
### Added
- Search and list results for cli (`-f`)
- Open first search result automatically from cli to $EDITOR (`-o`)

## [0.7.0] - 2024-12-16
### Added
- Added "-a" flag to append text directly to the current journal page.
- Added linting to to Github Actions workflow

### Changed
- Github Actions workflow structure into divided parts.

## [0.6.0] - 2024-12-14
### Added
- Search for page aliases

## [0.5.0] - 2024-12-07
### Added
- [Charmbracelet Lipgloss](https://github.com/charmbracelet/lipgloss) for modal positioning
- Added search state structure to track search UI state
- Implemented trie-based search functionality
- Keyboard shortcut (Ctrl+F) to activate search
- Added modal search interface.
- Updated footer to show search shortcut
- Recursive file collection for prefix matches

### Changed
- Updated tuiModel to include search state

## [0.4.1] - 2024-12-04
### Changed
- fmt to log in main.go

### Fixed
- Failure to load $EDITOR when -t is omitted.

## [0.4.0] - 2024-12-03
### Added
- Indent the line from any location on the line.
- Unindent the line from any location on the line.

### Changed
- Set editor to Nano when $EDITOR is blank.
- Moved SelectEditor logic into main.go loadEditor function.
- Dried up text manipulation code on key combos.
- Refactored TODO state cycling and todo priority cycling.

## [0.3.0] - 2024-12-01
### Added
- CHANGELOG.md
- CONTRIBUTORS.md
- Contributions section to README.md
- Specify a specific journal page data to open.

### Changed
- Link typos in README.md by [@kandros](https://github.com/jrswab/lsq/commits?author=kandros)
- LoadConfig and GetTodaysJournal functions into system package

## [0.2.0] - 2024-11-28
### Added
- Todo cycling with `ctrl+t`
- Priority cycling with `ctrl+p`

### Removed
- `ctrl+e` to open $EDITOR from TUI

### Changed
- README.md

## [0.1.0] - 2024-11-27
### Added
- Opening today's journal with $EDITOR by default
- Github Actions for CI
- Option to open with custom TUI
- Unit test for ConvertDateFormat

### Changed
- go.yml to use Go version 1.23
- ConvertDataFormat to use a slice for ordering.
- Moved ConvertDateFormat into config package.
- README.md

## [0.0.1] - 2024-11-26
### Added
- README.md
- LICENSE
- .gitignore
