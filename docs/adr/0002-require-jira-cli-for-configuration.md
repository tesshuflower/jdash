# Require jira-cli installation for Jira configuration

jdash requires users to have `jira-cli` installed and configured (`jira init`) before using jdash. We read the existing jira-cli config at `~/.config/jira-cli/jira.yml` for server URL, authentication, and connection details, rather than implementing our own initialization wizard.

## Why

We're already importing jira-cli packages for API access (ADR-0001). Requiring the CLI binary as well keeps the first-run experience simple:
- No duplicate initialization flows
- No duplicate config files
- Users who already use jira-cli have zero additional setup
- We don't own the auth/config UX — jira-cli handles Jira's complexity (Cloud vs On-Premise, basic/bearer/mTLS auth, .netrc, keyring, etc.)

## User Flow

1. User installs jdash: `go install github.com/tesshuflower/jdash@latest`
2. User installs jira-cli: `brew install jira-cli` (or equivalent)
3. User runs: `jira init` (interactive wizard creates `~/.config/jira-cli/jira.yml`)
4. User exports `JIRA_API_TOKEN` env var
5. User runs: `jdash` (reads jira-cli's config automatically)

## Consequences

- ✅ Simpler codebase — no init wizard, no auth UI
- ✅ Works immediately for existing jira-cli users
- ✅ One config file for both tools
- ⚠️ Additional dependency — users must install jira-cli binary even though we use it as a library
- ⚠️ If user uninstalls jira-cli (but keeps the config), jdash still works — but we should document the config requirement, not the binary requirement

## Rejected Alternative

**Implement our own init wizard** using jira-cli's config package would give a standalone experience, but we'd be reimplementing and maintaining their initialization logic, handling edge cases (non-English Jira, custom fields, multiple auth types), and creating a second config file standard.

## Future Consideration

If jira-cli becomes unmaintained or users strongly prefer standalone installation, we can vendor the config package and build our own `jdash init` command. The config file format is stable, so migration would be straightforward.
