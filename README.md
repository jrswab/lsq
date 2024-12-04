# lsq

A command-line tool for rapid journal entry creation in Logseq, featuring both TUI and external editor support.

## Features

- Terminal User Interface (TUI) with real-time editing
- External editor integration ($EDITOR by default)
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

This opens today's journal in your default editor ($EDITOR environment variable).

### Command Line Options

- `-c`: Specify config filename. (default: "config.edn")
- `-d`: Specify Logseq directory name. (default: "Logseq")
- `-e`: Set editor environment variable. (default: "$EDITOR")
- `-l`: Specify Logseq config directory name. (default: "logseq")
- `-s`: Specify the journal date to open. (Must be `yyy-MM-dd` formatted)
- `-t`: Use the built-in TUI instead of external editor.

### TUI Controls

- `Ctrl+S`: Save current file
- `Ctrl+C`: Quit
- `Ctrl+T`: Cycle through TODO states on current line
- `Ctrl+P`: Cycle through priority states on current line
- `tab`: Indent the entire line from anywhere on the line.
- `shift+tab`: Unindent the line from anywhere on te line.
- Arrow keys: Navigate through text

## Configuration

LSQ reads your Logseq configuration from `config.edn`. Supported settings:

- `meta/version`: Configuration version
- `preferred-format`: File format ("Markdown" or "Org")
- `journal/file-name-format`: Date format for journal files (e.g., "yyyy_MM_dd")

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea): Terminal UI framework
- [EDN](https://olympos.io/encoding/edn): Configuration file parsing

## Contributing

First off, thank you for considering contributing to lsq! 🎉

### Ways to Contribute

- Report bugs
- Suggest new features
- Improve documentation
- Submit pull requests
- Share how you use lsq
- Star the project on GitHub

### Development Setup

1. Fork the repository
2. Clone your fork:
```bash
git clone https://github.com/your-username/lsq.git
```
3. Add the upstream remote:
```bash
git remote add upstream https://github.com/jrswab/lsq.git
```
4. Create a branch for your work:
```bash
git checkout -b your-feature-branch
```

### Pull Request Process

1. Update the README.md with details of any interface changes if applicable
2. Keep PRs focused - one feature or fix per PR
3. Use clear, descriptive commit messages
4. Make sure your branch is up to date with main before submitting
5. Include a clear description of the changes in your PR

### First Time Contributors

New to contributing? Look for issues tagged with `good-first-issue` or `documentation`. These are great starting points!

See [CONTRIBUTORS.md](CONTRIBUTORS.md) for a list of project contributors.

## License

GPL v3
