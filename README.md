# JIG - Jira's webinterface Isn't Great

[![Status: WIP](https://img.shields.io/badge/status-WIP-red.svg)](https://github.com/emilsto/jig)

Under development, changes may be breaking at this moment. 

A streamlined command-line tool that bridges Jira and Git workflows, designed for developers who want to manage sprint tasks and create branches without leaving the terminal.

It's crafted for my specific needs with Jira, so that I don't have to open the websites poopy UI. Changes and new features might come, use this if it suits your workflow :)


## What is JIG?

JIG is a productivity tool that eliminates context switching between Jira's web interface and your terminal. It fetches your active sprint tickets, allows you to interact with them directly, and automates common workflows like creating branches, assigning tasks, and managing subtasks.

## Installation

Build it from source:

```bash
go build -o jig
```

Move the binary to your PATH:

```bash
mv jig /usr/local/bin/
```

or go install it:

```bash
go install github.com/emilsto/jig@latest
```

## Configuration

On first run, JIG will prompt you for the following information:

- Jira API Key
- Jira Company Name
- Jira Email

With this, JIG will fetch all projects and boards for your account and prompt you to select the ones you want to use.

You can add other boards to the project later by editing the config.toml file.

### Global Configuration

Create `~/.config/jig/config.toml`:

```toml
[api]
baseurl = "https://your-company.atlassian.net"
agileurl = "https://your-company.atlassian.net/rest/agile/1.0"
email = "your-email@company.com"
apikey = "your-jira-api-token"

[git]
branchbase = "develop"

[[projects]]
name = "Your Project"
id = "PROJ"

[[projects.boards]]
name = "Sprint Board"
id = 123
```

### Per-Directory Configuration

Initialize a `.jigrc` file in your project directory:

```bash
jig init
```

f
This remembers your project and board selection, so JIG automatically knows which sprint to fetch when you run it from that directory.

## Usage

### Basic Commands

```bash
# Run in interactive mode
jig

# Run once and exit (oneshot mode)
jig -o

# Initialize .jigrc for current directory
jig init

# Show help
jig -h

```

### Interactive Mode

When JIG starts, it displays your active sprint tickets once. You can then:

- **Select a ticket** by number to view details
- **Assign to yourself**: `3 -p`
- **Change status**: `3 -s`
- **Create branch**: `3 -g`
- **Create subtask + branch**: `3 -su`
- **Refresh ticket list**: `-l`
- **Show help**: `h`
- **Exit**: `0`

### Example Workflow

```bash
$ jig
Using Project: My Project (ID: PROJ), Board: Sprint Board (ID: 123)

Latest Sprint:
  - ID: 456, Name: Sprint 42, State: active

Active Items (5):
┌────┬──────────┬──────────────────────────────────────┬──────────┬──────────────┐
│ #  │ Key      │ Summary                              │ Status   │ Assignee     │
├────┼──────────┼──────────────────────────────────────┼──────────┼──────────────┤
│ 1  │ PROJ-123 │ Implement user authentication        │ To Do    │ Unassigned   │
│ 2  │ PROJ-124 │ Fix navigation bug                   │ In Review│ John Doe     │
│ 3  │ PROJ-125 │ Add payment integration              │ To Do    │ Unassigned   │
└────┴──────────┴──────────────────────────────────────┴──────────┴──────────────┘

→ Select action (number) + command suffix or 'h' for help: 1 -su
→ Enter subtask summary: Add login form validation
✓ Subtask created successfully: PROJ-126

→ Enter ticket description for branch name: login validation
✓ Branch created: feature/PROJ-126-login-validation
✓ Complete

→ Select action (number) + command suffix or 'h' for help: -l
[refreshes ticket list with new subtask]
```

## Command Reference

### Command-Line Flags

- `-h` - Show help message
- `-e` - Fetch epics from project (planned feature)
- `-o` - Oneshot mode (exit after one action)

### Interactive Commands

- `<number>` - View issue details
- `<number> -p` - Assign issue to yourself
- `<number> -s` - Change issue status
- `<number> -g` - Create git branch for issue
- `<number> -su` - Create subtask with branch
- `-l` - Refresh and list sprint tickets
- `h` - Show interactive help
- `0` - Exit

## Authentication

JIG uses Jira API tokens for authentication. Generate one at:
`https://id.atlassian.com/manage-profile/security/api-tokens`

Add it to your `config.toml` as the `apikey` value.

## Branch Naming Convention

Branches are created with the format:

```
<branchbase>/<JIRA-TICKET-KEY>/<description>
```

Based on your `branchbase` configuration, branches are created from that base branch (e.g., `develop` or `main`).

## Project Structure

- `main.go` - Core application logic and interactive loop
- `jira.go` - Jira API integration
- `git.go` - Git operations
- `config.go` - Configuration management
- `print.go` - Terminal output formatting
- `help.go` - Help text and documentation

## Why JIG?

Traditional Jira workflows require:

1. Opening browser
2. Finding your sprint
3. Clicking through tickets
4. Copy-pasting ticket keys
5. Manually creating branches
6. Switching back to terminal

JIG reduces this to:

1. `jig`
2. Select ticket + action
3. Done

## Requirements

- Go 1.23.1 or higher
- Jira API token
- Git
- A terminal that supports colors and formatting

---

## License

This project is licensed under the **GNU General Public License v3.0**. A copy of the license is available in the `LICENSE` file.

This software is provided "AS IS", without warranty of any kind, express or implied, including but not limited to the warranties of merchantability, fitness for a particular purpose and noninfringement. In no event shall the authors or copyright holders be liable for any claim, damages or other liability, whether in an action of contract, tort or otherwise, arising from, out of or in connection with the software or the use or other dealings in the software.
