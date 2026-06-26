# jdash

A terminal user interface (TUI) for Jira, inspired by [gh-dash](https://github.com/dlvhdr/gh-dash).

## Features

**Navigation**
- Multiple configurable sections with JQL filters
- Vim-style keybindings (h/j/k/l, g/G)
- Fast table navigation with live preview pane
- Sprint column with automatic board detection

**Actions**
- Add comments (press `c`)
- Change status (press `s`)
- Move to sprint (press `m`) with fuzzy filtering
- Open in browser (press `o`)
- Local fuzzy filter (press `/`) — instantly filter visible issues
- Edit queries on the fly (press `e`)
- Refresh sections (press `r` for current, `R` for all)

**Configuration**
- Auto-generated config on first run
- Custom sections with JQL queries
- Per-section layouts
- Lazy loading for expensive queries
- Configurable sprint field ID

**Help & Discovery**
- Press `?` to see all keybindings
- Inline hints and context-aware help

## Installation

### Prerequisites

jdash uses [jira-cli](https://github.com/ankitpokhrel/jira-cli) for Jira authentication and configuration.

```bash
# Install jira-cli
brew install jira-cli

# Configure your Jira connection
jira init

# Set your API token (required)
export JIRA_API_TOKEN=your-token-here
```

### Install jdash

```bash
# Clone the repository
git clone https://github.com/tesshuflower/jdash.git
cd jdash

# Build
go build -o jdash

# Run
./jdash
```

On first run, jdash will create `~/.config/jdash/config.yaml` with working defaults.

## Configuration

jdash reads Jira connection settings from jira-cli's config (`~/.config/.jira/.config.yml`) and its own settings from `~/.config/jdash/config.yaml`.

### Default Config

On first run, jdash creates a config with two sections:
- **In Sprint** — Your issues in active sprints
- **No Sprint / Future Sprint** — Your unscheduled or future-sprint issues

The config file includes commented examples for:
- Custom layouts (choose which columns to show)
- Team views (filter by component, project, etc.)
- Lazy loading (don't load until you switch to that section)
- Sprint field customization (if your Jira uses a different custom field)

### Example Custom Config

```yaml
sections:
  - title: In Sprint
    filters: assignee = currentUser() AND sprint in openSprints()
    layout: [key, summary, status, assignee, sprint]

  - title: High Priority Bugs
    filters: assignee = currentUser() AND type = Bug AND priority in (Highest, High)
    layout: [key, summary, status, priority, updated]

  - title: Team Sprint
    filters: sprint in openSprints() AND component = "MyComponent"
    lazy: true  # Don't load until first visit

# Global layout (used when section doesn't specify its own)
layout: [key, type, summary, status, assignee, component, sprint, updated]

# Sprint field ID (defaults to customfield_10020)
sprint_field: customfield_10020
```

**Available layout fields:** key, type, summary, status, assignee, component, sprint, updated, created, priority, reporter, labels, resolution, fixversion, parent

## Keybindings

Press `?` to see all keybindings. Common keys:

**Navigation**
- `h`/`l` or `←`/`→` — Switch sections
- `j`/`k` or `↑`/`↓` — Navigate issues
- `g`/`G` — First/last issue
- `tab`/`shift+tab` — Next/previous section

**Actions**
- `c` — Add comment
- `s` — Change status
- `m` — Move to sprint (with filtering: press `/`, type to filter, `Enter` to select)
- `o` — Open issue in browser
- `O` — Open browser to create new issue

**Filter & Search**
- `/` — Fuzzy filter issues in current section (matches all visible columns)
- `e` — Edit JQL query for current section (temporary, not saved)

**Other**
- `r` — Refresh current section
- `R` — Refresh all sections
- `?` — Toggle help
- `q` — Quit

## Sprint Support

jdash automatically detects which board an issue belongs to by parsing the sprint custom field. When you move an issue to a sprint:
- Issues **with a sprint**: Shows sprints from that issue's board (fast)
- Issues **without a sprint**: Shows sprints from all boards in your project (slower, but complete)

Press `/` while in the sprint selector to filter the list, then `j`/`k` to navigate.

## Troubleshooting

**"No issues found"**
- Check your JQL in `~/.config/jdash/config.yaml`
- Verify `JIRA_API_TOKEN` is set
- Check jira-cli config: `jira me`

**"Sprint column is empty"**
- Your Jira instance might use a different sprint field ID
- Find yours: `curl -u "email:token" "https://your-jira/rest/api/3/field" | grep -i sprint`
- Set `sprint_field: customfield_XXXXX` in your config

**"currentUser() doesn't work"**
- Some Jira instances don't support `currentUser()` in JQL
- The auto-generated config uses your actual email instead

## Documentation

- [CONTEXT.md](CONTEXT.md) — Domain glossary and design decisions
- [docs/adr/](docs/adr/) — Architecture decision records
- [docs/GRILLING-SESSION-SUMMARY.md](docs/GRILLING-SESSION-SUMMARY.md) — Planning session recap

## License

See [LICENSE](LICENSE)
