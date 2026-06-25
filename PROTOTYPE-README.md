# jdash Prototype

This is a prototype to validate the technical feasibility of jdash before full MVP development.

## What It Validates

1. ✅ **jira-cli package integration** — Uses `pkg/jira` as a library, reads jira-cli config, authenticates
2. ✅ **Bubbletea TUI** — Table view with vim navigation (j/k), preview pane showing issue details
3. ⏳ **Performance** — Test with your real Jira data (up to 100 issues)

## Prerequisites

1. **Install and configure jira-cli:**
   ```bash
   brew install jira-cli
   jira init
   ```

2. **Set your API token:**
   ```bash
   export JIRA_API_TOKEN=your-token-here
   ```

## Running the Prototype

```bash
./jdash
```

Or build and run:
```bash
go run .
```

## What You Should See

- **Table view** showing your assigned unresolved Jira issues
- **Columns**: Key, Type, Summary, Status, Assignee, Component, Updated
- **Navigation**: Use `j`/`k` or arrow keys to move between issues
- **Preview pane** (bottom): Shows details of the selected issue
- **Quit**: Press `q` or `Ctrl+C`

## Features Implemented

✅ **Config loading** — Reads `~/.config/.jira/.config.yml`  
✅ **API token resolution** — From `JIRA_API_TOKEN` env var  
✅ **Issue fetching** — Hardcoded JQL: `assignee = currentUser() AND resolution = Unresolved`  
✅ **Table display** — Bubbles table component with default columns  
✅ **Vim navigation** — j/k to move between issues (built into Bubbles table)  
✅ **Preview pane** — Auto-updates when cursor moves  
✅ **Error handling** — Shows errors gracefully, doesn't crash  
✅ **Empty state** — Shows "No issues found" placeholder  
✅ **Cloud/Local support** — Dispatches to v2 or v3 API based on config  

## Not In Prototype (MVP Features)

❌ Multiple sections (just one hardcoded section)  
❌ jdash config file (`~/.jdash.yml`)  
❌ Query editing (`/` key)  
❌ Write operations (move to sprint, change status, add comment)  
❌ Section switching  
❌ Custom keybindings  
❌ Status category coloring  

## Testing the Prototype

1. **Config loading**: Should detect jira-cli config and connect without errors
2. **Issue display**: Should show real issues from your Jira instance
3. **Navigation**: Press j/k to move, verify preview pane updates
4. **Performance**: How does it feel with your data? Load time? Navigation responsiveness?
5. **Error handling**: 
   - Rename jira-cli config temporarily — should show clear error, not crash
   - Unset `JIRA_API_TOKEN` — should show helpful error message

## Next Steps After Prototype

If the prototype validates feasibility (good performance, no blocking issues):

1. Implement MVP features (see CONTEXT.md)
2. Add jdash config file support (`~/.jdash.yml`)
3. Add multiple configurable sections
4. Implement write operations (move to sprint, change status, add comment)
5. Add query editing with `/` key
6. Polish error handling and loading states

If there are blockers, document them and decide whether to pivot or fix.

## File Structure

```
jdash/
├── main.go                      # Entry point
├── internal/
│   ├── config/
│   │   └── config.go            # jira-cli config reader
│   ├── jira/
│   │   └── client.go            # Jira API wrapper
│   └── ui/
│       └── model.go             # Bubbletea TUI
├── CONTEXT.md                   # Domain model
├── docs/adr/                    # Architecture decisions
└── PROTOTYPE-README.md          # This file
```
