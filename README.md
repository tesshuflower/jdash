# jdash

A terminal user interface (TUI) for Jira, inspired by [gh-dash](https://github.com/dlvhdr/gh-dash).

## Status: Prototype

Currently in prototype phase. See [PROTOTYPE-README.md](PROTOTYPE-README.md) for details on running and testing.

## What is jdash?

jdash is a Jira dashboard for the terminal that lets you:
- View issues in configurable filtered sections
- Navigate with vim-style keybindings
- Perform common actions without leaving the terminal
- Open issues in your browser for complex edits

### Planned MVP Features

- **Read operations**: View issues, navigate sections, edit queries temporarily
- **Write operations**: Move to sprint, change status, add comments
- **Launcher**: Open issues in browser
- **Configuration**: Define custom sections with JQL queries

See [CONTEXT.md](CONTEXT.md) for the complete domain model and [docs/adr/](docs/adr/) for architectural decisions.

## Installation (Prototype)

```bash
# Prerequisites
brew install jira-cli
jira init
export JIRA_API_TOKEN=your-token

# Build
go build -o jdash

# Run
./jdash
```

## Documentation

- [CONTEXT.md](CONTEXT.md) — Domain glossary and design decisions
- [PROTOTYPE-README.md](PROTOTYPE-README.md) — How to run and test the prototype
- [docs/adr/](docs/adr/) — Architecture decision records
- [docs/GRILLING-SESSION-SUMMARY.md](docs/GRILLING-SESSION-SUMMARY.md) — Planning session recap

## License

See [LICENSE](LICENSE)