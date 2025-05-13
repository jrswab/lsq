<p align="center">
<img width="25%" src="https://lsq.sh/media/img/lsq_logo_cropped.png" alt="lsq logo">
</p>

# lsq

[![Go Report Card](https://goreportcard.com/badge/github.com/jrswab/lsq)](https://goreportcard.com/report/github.com/jrswab/lsq)
[![Release](https://img.shields.io/github/v/release/jrswab/lsq)](https://github.com/jrswab/lsq/releases)
[![License](https://img.shields.io/github/license/jrswab/lsq)](https://github.com/jrswab/lsq/blob/master/LICENSE)

The ultra-fast CLI companion for [Logseq](https://github.com/logseq/logseq) designed to speed up your note capture directly from the terminal!

## Why lsq?
- ⚡️ Lightning-fast journal additions without leaving your terminal
- ⌨️ Optimized for both quick captures and extended writing sessions
- 🎯 Native support for Logseq's file naming and formatting conventions
- 🔄 Seamless integration with your existing Logseq workflow
- 💻 Built by Logseq users, for Logseq users

## Features That Power Your Workflow
- External editor integration ($EDITOR by default)
- Automatic journal file creation
- Support for both Markdown and Org formats
- Configurable file naming format
- Customizable directory location
- User defined configuration file

## Ready to Start?
1. Install the binary via Go:
```bash
go install github.com/jrswab/lsq@latest
```
2. Make sure you have the location of the Go binaries in your $PATH. Run `go env` and find the variable called `GOPATH`. Then copy that location to your shell's $PATH if it's not already there.

3. Then run:
```bash
lsq
```

## Usage
### Command Line Options
- `-a`: Append text directly to the current journal page
- `-A`: Append the contents of STDIN to the current journal page
- `-d`: Specify main directory path. (example: `/home/jrswab/Documents/Notes`)
- `-e`: Set editor to use while editing files. (Defaults to $EDITOR, then Vim if $EDITOR is not set)
- `-f`: Search pages and aliases. Must be followed by a string.
- `-o`: Automatically open the first result from the search.
- `-p`: Open a specific page from the pages directory.
- `-r`: Search pages and journals via regex pattern. Must be followed by a regex string.
- `-s`: Specify the journal date to open. (Must be `yyyy-MM-dd` formatted)
- `-v`: Display the version of lsq being executed.
- `-y`: Open yesterday's journal file.

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
**Note:** The configured directory must contain both a `journals` and `pages` subdirectory for lsq to function properly. These are automatically created when using Logseq, but will need to be manually created if setting lsq to use a new directory or without Logseq.

### Usage Examples:
```bash
lsq
```
This opens today's journal in your default editor ($EDITOR environment variable).
If no editor is defined in $EDITOR, then `Vim` will be used.

```bash
lsq -p file_name.md -a "text to append"
```
This combination will append the text to the page with file name `file_name.md`.
If `-p` is not provided the appended text will be placed in today's journal entry.

```bash
lsq -f word -o
```
This will search your pages for files containing "word" and open the first result in $EDITOR.
If `-o` is not provided lsq will output all files which contain "word" to STDOUT.

```bash
cat ~/.zshrc | lsq -A
```
This will take the contents of your `~/.zshrc` file and append it to your current
journal. This reads STDIN through to end-of-file, so be sure to `Ctrl-d` if your
contents don't contain an end-of-file.

```bash
run_long_batch_job |& lsq -A -p "long-job.$(date +%s).log"
```
This will run your long-running batch job, and it'll append the contents of STDIN
and STDERR (note the pipe!) to a new page called `long-job.UNIX_TIMESTAMP.log`.

## Contributing
For information on contributing to lsq check out [CONTRIBUTING.md](https://github.com/jrswab/lsq/blob/master/CONTRIBUTING.md).

## License
GPL v3
