# lsq

A command-line tool for rapid journal entry creation in Logseq, featuring both TUI and external editor support.

## Features

- Terminal User Interface (TUI) with real-time editing
- External editor integration (vim by default)
- Automatic journal file creation
- Support for both Markdown and Org formats
- Configurable file naming format
- Customizable Logseq directory location

## Installation

```bash
go install github.com/jrswab/lsq@latest
```

## Usage

Basic usage:
```bash
lsq
```

This opens today's journal in your default editor (EDITOR environment variable).

### Command Line Options

- `-t`: Use the built-in TUI instead of external editor
- `-d`: Specify Logseq directory name (default: "Logseq")
- `-l`: Specify Logseq config directory name (default: "logseq")
- `-c`: Specify config filename (default: "config.edn")
- `-e`: Set editor environment variable (default: "EDITOR")

### TUI Controls

- `Ctrl+S`: Save current file
- `Ctrl+C`: Quit
- `Ctrl+T`: Cycle through TODO states on current line
- `Ctrl+P`: Cycle through priority states on current line
- Arrow keys: Navigate through text

## Configuration

LSQ reads your Logseq configuration from `config.edn`. Supported settings:

- `meta/version`: Configuration version
- `preferred-format`: File format ("Markdown" or "Org")
- `journal/file-name-format`: Date format for journal files (e.g., "yyyy_MM_dd")

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea): Terminal UI framework
- [EDN](https://olympos.io/encoding/edn): Configuration file parsing

## License

GPL v3
