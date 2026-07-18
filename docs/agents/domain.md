# Domain Docs

This repository uses a single-context domain documentation layout.

## Before exploring

Read these files when they exist:

- `CONTEXT.md` at the repository root
- Applicable ADRs under `docs/adr/`

If either location does not exist, proceed without reporting it as an error.
The domain-modeling skills create these files when domain vocabulary or
architectural decisions are actually established.

## Layout

```text
/
├── CONTEXT.md
├── docs/
│   └── adr/
└── ...
```

`CONTEXT.md` defines the shared vocabulary for the Go backend, default
frontend, and classic frontend. System-wide architectural decisions belong in
`docs/adr/`.

## Vocabulary

Use terms defined in `CONTEXT.md` when naming issues, specifications,
refactors, hypotheses, and tests. Avoid introducing synonyms for concepts that
already have an established project term.

If a required concept has no established term, record that gap for domain
modeling instead of silently inventing competing vocabulary.

## ADR conflicts

When proposed work conflicts with an existing ADR, identify the affected ADR
and explain why the decision may need to be reconsidered. Do not silently
override an existing architectural decision.
