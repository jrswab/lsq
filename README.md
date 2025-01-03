# lsq

A command-line tool for rapid journal entry creation in Logseq, featuring both TUI and external editor support.

## Features
- External editor integration ($EDITOR by default)
- Automatic journal file creation
- Support for both Markdown and Org formats
- Configurable file naming format
- Customizable directory location
- User defined configuration file

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
If no editor is defined in $EDITOR, then `Vim` will be used.

### Command Line Options
- `-a`: Append text directly to the current journal page
- `-d`: Specify main directory path. (example: `/home/jrswab/Documents/Notes`)
- `-e`: Set editor to use while editing files. (Defaults to $EDITOR, then Vim if $EDITOR is not set)
- `-f`: Search pages and aliases. Must be followed by a string.
- `-o`: Automatically open the first result from the search.
- `-p`: Open a specific page from the pages directory.
- `-s`: Specify the journal date to open. (Must be `yyyy-MM-dd` formatted)
- `-v`: Display the version of lsq being executed. (Added in v0.11.0)
- `-y`: Open yesterday's journal file. (Added in v0.11.0)

### Configuration File
This file must be stored in your config directory as `lsq/config.edn`.
On Unix systems, it returns `$XDG_CONFIG_HOME` if non-empty, else `$HOME/.config` will be used.
On macOS, it returns `$HOME/Library/Application Support`.
On Windows, it returns `%AppData%`.
On Plan 9, it returns `$home/lib`.

#### Configuration Behavior
The configuration file will override any lsq defaults which are defined. If a CLI flag is provided, the flag value will override the config file value.

#### Configuration File Example:
```EDN
{
  ;; Either "Markdown" or "Org".
  :file/type "Markdown"
  ;; This will be used for journal file names
  ;; Using the format below and the file type above will produce 2025.01.01.md
  :file/format "yyyy_MM_dd"
  ;; The directory which holds all your notes
  :directory "/home/jaron/Logseq"
}
```

## TUI (Deprecated)
As lsq moves toward v1.0.0, I've decided to focus on perfecting the core CLI experience. The TUI interface is now deprecated in favor of enhanced external editor integration and improved command-line workflows. This aligns with the project goal of providing the fastest, most reliable journaling experience possible. While the TUI was fast and operated well, it's outside of the current scope of this project. However, this does not mean that TUI is gone forever and if the community wants a TUI after v1.0.0 is released, I'd be happy to work on it again.

### TUI Controls (Deprecated)
- `Ctrl+S`: Save current file
- `Ctrl+C`: Quit
- `Ctrl+T`: Cycle through TODO states on current line
- `Ctrl+P`: Cycle through priority states on current line
- `Ctrl+F`: Open search modal
- `tab`: Indent the entire line from anywhere on the line.
- `shift+tab`: Unindent the line from anywhere on the line.
- Arrow keys: Navigate through text

### TUI Search Modal Controls (Deprecated)
- Type to search through files
- `↑/↓`: Navigate through results
- `Enter`: Open selected file (current file saves on open)
- `Esc`: Close search modal

## Dependencies
- [Bubble Tea](https://github.com/charmbracelet/bubbletea): Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss): Terminal UI styling
- [EDN](https://olympos.io/encoding/edn): Configuration file parsing

## Contributing
For information on contributing to lsq check out [CONTRIBUTING.md](https://github.com/jrswab/lsq/blob/master/CONTRIBUTING.md).

## License
GPL v3
