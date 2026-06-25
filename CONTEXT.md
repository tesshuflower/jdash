# jdash Domain Model

## What is jdash?

jdash (short for "jira-dash") is a terminal user interface (TUI) for Jira, inspired by [gh-dash](https://github.com/dlvhdr/gh-dash). It functions primarily as a **dashboard** — providing quick visibility into Jira issues — with **launcher** capabilities for opening issues in the browser and performing common actions without leaving the terminal.

## Core Concepts

### Issue
A Jira issue. jdash adopts Jira's terminology directly: Issue, Epic, Story, Task, Bug, Subtask. The type of issue matters for display purposes (showing whether something is a Story vs Epic vs Task), but from jdash's perspective, all are Issues with different type attributes.

### Section
A filtered view of Issues displayed in the TUI, stacked vertically. Users navigate between sections using keyboard shortcuts (vim-style j/k navigation). Each Section has:
- **Title** — display name shown in the TUI
- **Filters** — JQL (Jira Query Language) that determines which Issues appear in this Section

Example configuration:
```yaml
sections:
  - title: Assigned to Me
    filters: assignee = currentUser()
  - title: Current Sprint  
    filters: sprint in openSprints()
```

### JQL (Jira Query Language)
Jira's query language for filtering issues. Used in Section filters to define which Issues appear. JQL supports field comparisons (`assignee = currentUser()`), boolean operators (`AND`, `OR`), and functions (`openSprints()`, `currentUser()`). jdash passes JQL directly to Jira's API without translation.

### Component
A Jira field used to organize issues within a project by functional area or team. Components are project-specific and configured in Jira. Example: "ACM", "Hive", "Authentication". An issue can have zero, one, or multiple components. Important for scoping "team" views in multi-component projects.

### Status Category
Jira's high-level grouping of statuses into three categories: **To Do**, **In Progress**, and **Done**. While status names are user-defined (e.g., "Code Review", "Blocked", "In QA"), every status maps to one of these three categories. jdash uses status categories for coloring — all "In Progress" category statuses get the same color, regardless of their specific names.

### Fix Version
A Jira field indicating which product release will include the fix/feature for an issue. Projects define their own set of fix versions (e.g., "2.1.0", "2.2.0", "Q1 2026"). Issues can have zero or multiple fix versions. jdash allows changing an issue's fix version from the TUI.

## Jira Integration

jdash integrates with Jira's data model, not an abstraction over it. This means:
- **Statuses** are loaded from Jira (not hardcoded)
- **Sprints** are loaded from Jira
- **Fix Versions** are loaded from Jira
- **Fields** are Jira's fields (not mapped to jdash concepts)

The user's Jira instance defines what statuses, workflows, and fields exist. jdash adapts to that schema.

## Scope

### MVP (v0.1)

**Read operations:**
- View Issues in configured sections
- Navigate between sections (vim-style)
- View Issue details in preview pane
- Edit section query temporarily (press `/`, edit JQL, Enter to refresh — similar to gh-dash)

**Write operations (MVP):**
- Move to sprint
- Change status (workflow transition)
- Add comment

**Launcher operations:**
- Open Issue in browser (`o` key)
- Open browser to create new Issue (popup, not in-TUI creation)

### Post-MVP

**Write operations (future):**
- Change fix version
- Add PR link
- Assign/reassign

**Search enhancements (future):**
- Save edited query as new section from UI
- Search history

**Out of scope (indefinitely):**
- Creating Issues within jdash TUI (use browser popup instead)
- Full workflow builder/configuration
- Deleting Issues
- Complex custom field editing beyond simple text/select fields

## Edge Case Handling

### Move to Sprint
- Show only sprints from the issue's board (issues can only move to sprints on their own board)
- Requires: fetch board for the issue, then fetch sprints for that board
- Future sprints appear first, current/active sprint marked clearly
- If issue belongs to a Kanban board (no sprint support), show error: "This board doesn't support sprints"

### Change Status
- Show all statuses available in the Jira instance
- If user selects an invalid transition (workflow doesn't allow it), Jira API will reject it and jdash displays the error
- Trade-off: Simpler implementation vs fetching valid transitions per issue (extra API call)

### Add Comment
- Accept Markdown input from user
- Convert to Atlassian Document Format (ADF) using jira-cli's `pkg/md` and `pkg/adf` packages
- Jira's API requires ADF; plain text and Markdown must be converted

### Network Failures
- Display error banner when API calls fail
- Keep UI functional and allow retry
- Do not quit or freeze the application
- Show last-fetched data if available (stale but viewable)

## Customization Priorities

jdash's customization model prioritized by importance:

### 1. Section Configuration (Highest Priority)
Users define which Sections appear and what JQL queries power them. This is the core value — quick access to the Issues that matter to your workflow.

Default sections provided out-of-box, but users customize in config.

### 2. Column Layout (Medium Priority)
Users configure which fields display in the Issue table.

**Default columns:** `[key, type, summary, status, assignee, component, sprint, updated]`

**Supported fields:**
- `key` — Issue key (e.g., ACM-12345)
- `type` — Issue type (Story, Bug, Epic, Task, etc.)
- `summary` — Issue summary/title
- `status` — Current status
- `assignee` — Assigned user
- `component` — First component (if multiple)
- `sprint` — Sprint name (if issue is in a sprint, shows most recent)
- `updated` — Last updated date (YYYY-MM-DD)
- `created` — Created date (YYYY-MM-DD)
- `priority` — Priority level
- `reporter` — User who created the issue
- `labels` — Comma-separated labels
- `resolution` — Resolution status
- `fixversion` — First fix version (if multiple)
- `parent` — Parent issue key (for subtasks)

**Sprint field configuration:** Sprint data is extracted from a Jira custom field (defaults to `customfield_10020`, the most common value). If your Jira instance uses a different custom field ID for sprints, set `sprint_field: customfield_XXXXX` in `~/.config/jdash/config.yaml`. To find your sprint field ID, run:
```bash
curl -u "email:api-token" "https://your-jira/rest/api/3/field" | grep -i sprint
```

**Other custom fields:** Custom fields beyond sprint are not currently supported. The jira-cli library doesn't expose a generic mechanism for custom field access. Sprint support is implemented via raw API parsing.

### 3. Keybindings (Low Priority)
Start with sensible defaults similar to gh-dash (vim-style navigation). Customization deferred to future versions.

### 4. Colors/Theme (Low Priority)
Start with status category-based coloring (To Do, In Progress, Done map to different colors). Custom color mapping deferred to future versions.

## Configuration

### Jira Connection
jdash reads Jira server and authentication settings from jira-cli's config at `~/.config/.jira/.config.yml`. Users must install and configure jira-cli before using jdash (see ADR-0002).

**Note on `currentUser()` JQL function:** Some Jira instances don't support `currentUser()` in JQL queries. jdash works around this by reading the login email from jira-cli config and using it directly in queries (e.g., `assignee = "user@example.com"` instead of `assignee = currentUser()`).

### jdash Config File
jdash's own configuration lives at `~/.config/jdash/config.yaml` (follows XDG Base Directory specification).

When jdash runs without a config file, it provides these defaults:

**Default sections (when no config file exists):**
```yaml
sections:
  - title: In Sprint
    filters: assignee = "user@example.com" AND sprint in openSprints()
  - title: No Sprint / Future Sprint
    filters: assignee = "user@example.com" AND (sprint is EMPTY OR sprint in futureSprints()) AND resolution = Unresolved
```

Where `user@example.com` is automatically substituted with your login email from jira-cli config.

**Example custom config** (`~/.config/jdash/config.yaml`):
```yaml
sections:
  - title: In Sprint
    filters: assignee = currentUser() AND sprint in openSprints()
  - title: No Sprint Assigned
    filters: assignee = currentUser() AND sprint is EMPTY AND resolution = Unresolved
  - title: Team Sprint (ACM)
    filters: sprint in openSprints() AND component = "ACM"
```

**Why no "team" view by default?** Jira teams organize differently (by component, project, board, or labels). A generic `sprint in openSprints()` returns too much noise across all accessible projects. Users add team-scoped sections in their config once they know their organization's structure.

**Default layout:**
```yaml
layout: [key, type, summary, status, assignee, component, sprint, updated]
```

**Empty states:** When a section has no issues, display a placeholder message similar to gh-dash (e.g., "No issues found").

## Prototype Plan

Before full implementation, build a prototype to validate technical feasibility. Focus on three high-risk areas:

### A) jira-cli Package Integration
**Goal:** Prove we can use jira-cli as a library dependency

**Tasks:**
- Import `github.com/ankitpokhrel/jira-cli/pkg/jira` and `api` packages
- Read jira-cli config from `~/.config/jira-cli/jira.yml`
- Initialize a jira client with existing auth
- Execute a simple JQL query (`assignee = currentUser() AND resolution = Unresolved`)
- Parse response into Go structs
- Display raw issue data (key, summary, status) in terminal

**Success criteria:** Can fetch and parse real issues from your Jira instance

### B) Bubbletea + Data Display
**Goal:** Prove we can build the UI with Bubbletea

**Tasks:**
- Create basic Bubbletea app (model-update-view)
- Render a table of issues using Bubbles table component
- Implement vim-style navigation (j/k to move between issues)
- Show selected issue details in a bottom preview pane (split layout)
- Style with Lipgloss (borders, colors for status categories)

**Success criteria:** Can navigate a list of real issues with keyboard, see details update in preview pane

### E) Real-World Performance
**Goal:** Prove it performs well with realistic data

**Tasks:**
- Fetch 100+ issues from your actual Jira (current sprint or assigned to you)
- Measure load time and render time
- Test navigation responsiveness with large dataset
- Profile memory usage with large issue lists

**Success criteria:** Load time <3 seconds for 100 issues, navigation feels instant, no memory issues

### Out of Prototype Scope
- Multiple sections (just one hardcoded section)
- Configuration files (hardcoded JQL)
- Write operations (read-only)
- Query editing
- Error handling polish
- Custom layouts

**Prototype success = greenlight for full MVP development**
