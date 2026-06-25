# jdash Grilling Session Summary

**Date:** 2026-06-25  
**Purpose:** Domain modeling and architecture decisions for jdash (Jira TUI dashboard)

## What We Built

### Documentation Created
1. **CONTEXT.md** — Domain glossary and design decisions
2. **ADR-0001** — Import jira-cli packages for Jira API integration
3. **ADR-0002** — Require jira-cli for configuration
4. **ADR-0003** — Use Bubbletea for TUI framework

### Key Decisions Stabilized

**What jdash is:**
- TUI dashboard for Jira (inspired by gh-dash)
- Sprint-focused workflow
- Read operations + limited write operations
- Launcher for browser-based complex edits

**Architecture:**
- Go + Bubbletea (Elm architecture TUI)
- Import jira-cli packages (`pkg/jira`, `api`) for Jira API
- Read jira-cli config (`~/.config/jira-cli/jira.yml`) for auth
- Config at `~/.jdash.yml` for sections/layout

**MVP Scope (v0.1):**
- Read: View issues, navigate sections, preview pane, temp query editing (`/`)
- Write: Move to sprint, change status, add comment
- Launch: Open issue in browser

**Default Configuration:**
- Sections: "In Sprint", "No Sprint Assigned" (with team section example commented)
- Layout: `[key, type, summary, status, assignee, component, sprint, updated]`

**Edge Cases:**
- Move to sprint: Show only sprints from issue's board, future-first ordering
- Change status: Show all statuses, let Jira reject invalid transitions
- Add comment: Accept Markdown, convert to ADF
- Network failures: Show error banner, keep UI functional

## Next Steps: Prototype

**Goal:** Validate technical feasibility before full implementation

**Focus Areas:**
1. **jira-cli integration** — Can we use it as a library? Does auth work?
2. **Bubbletea UI** — Can we render tables, navigate, show preview pane?
3. **Performance** — Does it handle 100+ issues smoothly?

**Prototype deliverable:**
- Single hardcoded section showing real Jira issues
- Table view with vim navigation (j/k)
- Preview pane showing selected issue details
- Performance metrics for realistic dataset

**Prototype success = proceed to MVP development**

## Domain Model Highlights

**Core entities:**
- **Issue** — Jira work item (Story, Epic, Task, Bug, etc.)
- **Section** — Filtered view (title + JQL query)
- **JQL** — Jira Query Language (passed directly to API)
- **Component** — Jira's functional area/team field
- **Status Category** — To Do / In Progress / Done (for coloring)
- **Fix Version** — Release version field

**Integration philosophy:**
- Adapt to Jira's schema (don't abstract it)
- Load statuses, sprints, fix versions from Jira
- Use Jira's terminology throughout

## Deferred to Post-MVP

- Custom commands
- Keybinding customization
- Color/theme customization
- Save edited queries as sections
- Write operations: fix version, PR link, assign/reassign
- Issue creation (use browser popup indefinitely)

## Repository State

```
jdash/
├── CONTEXT.md              # Domain glossary
├── docs/
│   ├── adr/
│   │   ├── 0001-import-jira-cli-packages.md
│   │   ├── 0002-require-jira-cli-for-configuration.md
│   │   └── 0003-use-bubbletea-for-tui.md
│   └── GRILLING-SESSION-SUMMARY.md  # This file
├── LICENSE
└── README.md
```

No code yet — documentation-first approach complete. Ready for prototype.
