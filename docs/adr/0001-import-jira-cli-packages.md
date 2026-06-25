# Import jira-cli packages for Jira API integration

We will use `github.com/ankitpokhrel/jira-cli/pkg/jira` and related packages as a Go module dependency (not vendored) to communicate with Jira's REST API, rather than shelling out to the jira-cli executable or building our own API client from scratch.

## Considered Options

**A) Shell out to jira-cli** — Execute `jira issue list --jql "..."` and parse text output
- Rejected: Process spawning overhead, text parsing fragility, version coupling

**B) Direct REST API integration** — Build our own HTTP client for Jira's REST API
- Rejected: Reinventing battle-tested code, must handle auth/versioning/retries ourselves

**C) Import jira-cli Go packages** — Use `pkg/jira.Client` as a library dependency
- **Selected**: Proven code (5k+ GitHub stars), handles API versioning (v2/v3), faster than shell execution, MIT licensed, designed for library use

## Why not vendor?

We prefer Go module dependency over vendoring to receive upstream bug fixes and security patches automatically. The jira-cli project is actively maintained. If the dependency becomes problematic, we can vendor or fork later.

## Consequences

- We depend on jira-cli's release cycle and API stability
- Breaking changes in their `pkg/` packages would require jdash updates
- We gain: auth handling (basic, bearer, mTLS), API version negotiation, JQL support, proven error handling
- We avoid: parsing CLI text output, maintaining our own HTTP client, testing against Jira API quirks

## What we import

Primary packages:
- `pkg/jira` — HTTP client, resource operations (Issue, Sprint, Epic, Board, etc.)
- `pkg/jql` — JQL query building helpers
- `pkg/adf` — Atlassian Document Format handling
- `api` — High-level API with v2/v3 version routing

If these become limiting, the `jira.Client` abstraction is narrow enough to swap implementations later.
