# sopsy-world

## Identity

sopsy is a single Go binary (macOS, Linux, Windows): a CLI plus a local web UI for a SOPS + age encrypted dotenv file. SOPS and age are compiled in as libraries. You work in the public repo niclasedge/sopsy; the hard rules live in `CLAUDE.md` and always apply.

This world says where things are and how work flows. Numbers, status and inventories are not here; the coverage map names the commands that derive them.

## Routes

- Workflow → plans → Code layout
- Code layout → builds on → Software stack
- Testing → verifies → Code layout
- Workflow → gates on → Testing
- Release → ships → Code layout
- Software stack → configures → Release

## Coverage-Map

One row per area. Read only the section behind the anchor, then go deeper when the task needs it.

| Dimension | Exakte Adresse | Was drin steht | Lesen, wenn die Aufgabe verlangt … |
|---|---|---|---|
| Workflow | `sopsy-world/workflow.md#auf-einen-blick` | Issue → OpenSpec change → PR, manual checks, public-repo hygiene, conventions | Start, implement, archive or merge a piece of work. |
| Testing | `sopsy-world/test.md#auf-einen-blick` | No-secret harness, isolation, sops compatibility, test-only key, web UI tests, test style | Write or fix a test. |
| Code layout | `sopsy-world/code.md#auf-einen-blick` | Packages, platform code, key lookup, errors and exit codes | Find where a behaviour lives or add a package. |
| Software stack | `sopsy-world/stack.md#auf-einen-blick` | Libraries, tooling and CI, dependencies | Add a dependency, touch lint or CI. |
| Release | `sopsy-world/deploy.md#auf-einen-blick` | Dry run, tagging, version stamping | Prepare or check a release (tag only with approval). |
| Design | `docs/design.md` | Approved v1 design: commands, files, web UI | Understand what sopsy should do. |
| Specs (live) | `openspec list` · `openspec/specs/` | Current capability specs and open changes | Know what a capability promises. |
| Build | `sopsy-world/build.md#auf-einen-blick` | Commands and source locations | Run build checks. |
| Mission (live) | `gh issue list --state open` | Open work; not part of the world | Know what comes next. |
| Zustand (live) | `git log --oneline` · `go list ./...` | History and package inventory | Need numbers or history; never copy them from a file. |
| Skills | `/opsx:propose` · `/opsx:apply` · `/opsx:archive` (`.claude/commands/opsx/`) | OpenSpec steps as commands | Run a workflow step instead of improvising. |
| Grenzen | `sopsy-world/boundaries.md` | What the world does not decide | Make a product or release decision. |
