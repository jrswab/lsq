# lsq

A command-line tool for rapid journal entry creation in Logseq, featuring both TUI and external editor support.

## Features
- External editor integration ($EDITOR by default)
- Terminal User Interface (TUI) with real-time editing
- Automatic journal file creation
- Support for both Markdown and Org formats
- Configurable file naming format
- Customizable Logseq directory location

### TUI Specific Features
- File search functionality with prefix matching
- TODO & priority cycling through keyboard shortcuts
- Line indentation & unindentation
- Auto-save when switching files through search

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
If no editor is defined in $EDITOR then `Vim` is will be used.

### Command Line Options

- `-a`: Append text directly to the current journal page
- `-c`: Specify config filename. (default: "config.edn")
- `-d`: Specify Logseq directory path. (default: "~/Logseq", expample: "/home/jaron/Document/Notes" you can also used "~/" instead of full home path.)
- `-e`: Set editor to use while editing files. (If the flag is not provided `$EDITOR` is used. If `$EDITOR` is not set, Vim is used.)
- `-f`: Search pages and aliases. Must be followed by a string.
- `-l`: Specify Logseq config directory name. (default: "logseq")
- `-o`: Automatically open the first result from the search.
- `-p`: Open a specific page from the Logseq pages directory.
- `-s`: Specify the journal date to open. (Must be `yyy-MM-dd` formatted)
~~- `-t`: Use the built-in TUI instead of external editor.~~

## TUI (Deprecated)
As lsq moves toward v1.0.0, I've decided to focus on perfecting the core CLI experience. The TUI interface is now deprecated in favor of enhanced external editor integration and improved command-line workflows. This aligns with the project goal of providing the fastest, most reliable journaling experience possible. While the TUI was fast and operated well, it's outside of the current scope of this project. However, this does not mean that TUI is gone forever and if the community wants a TUI after v1.0.0 is released, I'd be happy to work on it again.

### TUI Controls (Deprecated)

~~- `Ctrl+S`: Save current file~~
~~- `Ctrl+C`: Quit~~
~~- `Ctrl+T`: Cycle through TODO states on current line~~
~~- `Ctrl+P`: Cycle through priority states on current line~~
~~- `Ctrl+F`: Open search modal~~
~~- `tab`: Indent the entire line from anywhere on the line.~~
~~- `shift+tab`: Unindent the line from anywhere on te line.~~
~~- Arrow keys: Navigate through text~~

### TUI Search Modal Controls (Deprecated)

~~- Type to search through files~~
~~- `↑/↓`: Navigate through results~~
~~- `Enter`: Open selected file (current files saves on open)~~
~~- `Esc`: Close search modal~~

## Configuration

LSQ reads your Logseq configuration from `config.edn`. Supported settings:

- `meta/version`: Configuration version
- `preferred-format`: File format ("Markdown" or "Org")
- `journal/file-name-format`: Date format for journal files (e.g., "yyyy_MM_dd")

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea): Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss): Terminal UI styling
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
