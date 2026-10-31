# Workflow

## Auf einen Blick

Every piece of work is one GitHub issue **and** one OpenSpec change under
`openspec/changes/<name>/` (proposal, design, specs, tasks). The issue links the
change; the change is the source of truth for scope. It moves from proposal
through a branch and local gates to an archived change and a squash-merged PR.

## Issue change PR

1. **Propose.** New work starts as an issue plus `openspec propose` (or
   `/opsx:propose`). Scope and design are agreed before code is written.
   Every task names its verification (test, command or observable behaviour).
2. **Branch.** `change/<name>` from an up-to-date `main`. One change per
   branch, one branch per PR. Respect the dependency order in the issues.
3. **Implement** the tasks in order. Tick a task (`- [x]`) only after its
   named verification passed. If reality differs from the design, update the
   change's `design.md`/specs in the same branch and say so in the PR — never
   silently drift.
4. **Local gates** (all must pass before pushing): the commands in `CLAUDE.md`.
5. **Archive** the change as the last commit of the branch:
   `openspec archive <name> --yes` — this moves the delta specs into
   `openspec/specs/`, so `main` always has specs that match its code.
6. **Pull request.** Title in Conventional Commits form, e.g.
   `feat(run): start a command with secrets in its environment`. Body:
   `Closes #N`, link to the change, verification evidence per task, and any
   deviation from the design or a task that could only be verified partially
   (with a follow-up issue).
7. **CI gate.** The `test` workflow must be green on ubuntu, macos and windows,
   plus `lint`. Then squash-merge and delete the branch.

Belege: `openspec/changes/archive/`

Wo: `openspec/changes/`, `openspec/specs/`, `openspec/config.yaml`, `.claude/commands/opsx/`, `.claude/skills/`

## Manual checks

Tasks that need a manual check on a platform we cannot reach are not ticked
silently: record what was verified where, and open a follow-up issue for the
rest.

Belege: #17

Wo: `openspec/changes/`

## Public repo hygiene

- `.gitignore` blocks `keys.txt`, `.env`, `*.dec`, `*.plain`; do not weaken it.
- Security-relevant behaviour (web UI hardening, file permissions, key lookup)
  always gets a test that would fail if the protection were removed.

Wo: `.gitignore`

## Conventions

- Code, comments, commit messages, issues and PRs in English.
- Conventional Commits (`feat`, `fix`, `docs`, `test`, `ci`, `chore`, `refactor`),
  scope = capability or package.
- Comments explain *why*; the code says *what*. Match the density of the
  surrounding code.
- A PR does what its change says and nothing else.

Wo: `openspec/config.yaml`
