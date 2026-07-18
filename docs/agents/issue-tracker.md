# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub issues. Use the `gh` CLI for all
operations.

## Repository scope

Infer the repository from `git remote -v`. When run inside this checkout, issue
operations target the `zeta987/new-api` fork unless a command explicitly names
another repository.

Owner-authored pull requests to the fork may be used for development,
integration, and release review. Contributing pull requests to upstream is
outside this self-use workflow.

## Conventions

- **Create an issue**:
  `gh issue create --title "<title>" --body-file <markdown-file>`
- **Read an issue**:
  `gh issue view <number> --comments`
- **List issues**:
  `gh issue list --state open --json number,title,body,labels,comments`
- **Comment on an issue**:
  `gh issue comment <number> --body-file <markdown-file>`
- **Apply or remove labels**:
  `gh issue edit <number> --add-label "<label>"`
  or `gh issue edit <number> --remove-label "<label>"`
- **Close an issue**:
  `gh issue close <number> --comment "<reason>"`

For multiline issue bodies, create the Markdown body with the repository's
approved file-editing mechanism and pass it through `--body-file`. Follow the
repository's PowerShell and approval rules for every command.

## Pull requests as a triage surface

**PRs as a request surface: no.**

This flag controls whether pull requests enter the issue-triage queue. It does
not prevent owner-authored pull requests from being created or merged within
the fork.

GitHub shares one number space across issues and pull requests. Resolve an
ambiguous reference such as `#42` with `gh pr view 42`, then fall back to
`gh issue view 42`.

## When a skill says "publish to the issue tracker"

Create a GitHub issue in the fork.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.

## Wayfinding operations

The wayfinder map is a GitHub issue labelled `wayfinder:map`. Child work uses
GitHub sub-issues when available and a task list fallback otherwise.

Child tickets use `wayfinder:<type>` labels such as `wayfinder:research`,
`wayfinder:prototype`, `wayfinder:grilling`, and `wayfinder:task`.

Use GitHub issue dependencies for blockers when available. A ticket becomes
available when all blockers are closed and it has no assignee.

Claim a ticket with:

`gh issue edit <number> --add-assignee @me`

Resolve it by posting the result, closing the issue, and updating the map's
Decisions-so-far section.
