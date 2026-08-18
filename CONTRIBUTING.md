# Contributing to OpenEverest

Welcome! We are glad that you want to contribute to the OpenEverest project!

[OpenEverest](https://openeverest.io/) is an open source cloud-native database platform that lets developers deploy and manage PostgreSQL, MySQL, MongoDB and other databases on Kubernetes with ease. There are many ways to get involved, and every contribution matters.

Before diving in, please read our [Code of Conduct](https://github.com/openeverest/governance/blob/main/CODE_OF_CONDUCT.md).

The guidelines below are a starting point. We don't want to limit your creativity, passion, and initiative. If you think there are other ways you can contribute, feel free to bring it up in a GitHub Issue or open a Pull Request!

## Ways to contribute

We welcome many types of contributions including:

- New features and enhancements
- Bug reports and fixes
- [Documentation](https://github.com/openeverest/everest-doc)
- Builds, CI/CD improvements
- Issue triage
- Answering questions on [Slack or other community channels](https://openeverest.io/#community) and GitHub Discussions
- Blog posts, social media, and other community advocacy
- [Website and blog posts](https://github.com/openeverest/openeverest.github.io)
- Let us know when your talk on OpenEverest is accepted at a conference!
- Release management
- Problems found while setting up the development environment

For development contributions, please refer to the separate sections below.

## Ask for Help

The best way to reach us with a question when contributing is to join our community channels at [openeverest.io/#community](https://openeverest.io/#community) (Slack and more), or start a new [GitHub Discussion](https://github.com/openeverest/openeverest/discussions).

## Raising Issues

When raising [Issues](https://github.com/openeverest/openeverest/issues), please follow the template and fill the corresponding fields. Details matter.

If you are trying to report a vulnerability, please refer to our [Security Policy](https://github.com/openeverest/openeverest/blob/main/SECURITY.md).

## Working on Issues

### Finding something to work on

Every new issue starts untriaged. A maintainer reads it and either accepts it, asks for more information, or closes it with an explanation. We aim to do this within 5 business days.

Labels tell you where an issue stands:

| Label | What it means for you |
| --- | --- |
| `needs-triage` | Nobody has looked at it yet. Added automatically when the issue is opened. |
| `triage/needs-information` | We're waiting on the reporter before deciding. |
| `triage/accepted` | We agree this should be done. |
| `triage/duplicate` | Already reported — the earliest report is canonical, follow that one. |
| `triage/not-reproducible` | We couldn't reproduce it as reported. A working reproduction reopens the conversation. |
| [`help wanted`](https://github.com/openeverest/openeverest/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22+no%3Aassignee) | Accepted, and we'd welcome community help. **Claim these yourself with `/assign`.** |
| [`good first issue`](https://github.com/openeverest/openeverest/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22+no%3Aassignee) | A `help wanted` issue that needs no deep background. Ideal if this is your first contribution here. |

**The triage labels are maintained by a bot, so they are always current.** `needs-triage` is applied to every issue as it is opened, and is removed only when a maintainer records a decision by applying one of the `triage/*` labels. If a decision label is later taken off, `needs-triage` comes back. Exactly one triage label is on every open issue, so "has anybody looked at this yet?" is answerable at a glance.

They describe where our thinking has got to, not what you are allowed to do. An issue sitting at `needs-triage` means we might decide it isn't a bug, or that we don't want it fixed the way it proposes — which is worth knowing before you spend a weekend on it, and is the only reason to wait.

**This applies to issues you open yourself.** Filing an issue does not assign it to you — comment `/assign` if you want it.

**Small, obvious fixes don't need to wait for any of this.** A wrong error message, a missing nil check, a typo in a log line — open the pull request and reference the issue. Triage exists to stop you sinking days into something we may not want, not to make you queue for permission to fix something that is plainly broken.

**What a `good first issue` should give you.** Before we apply that label, the issue should describe both the problem and the shape of the fix, link the code and the tests you will need to touch, need no unusual setup or deep background, and have somebody willing to answer questions while you work on it. If one of those is missing, say so in the thread — that is a shortcoming in the issue, not in you.

### Claiming an issue

**Comment `/assign` on any open issue.** A bot assigns it to you straight away — no maintainer needs to be involved, and you do not need to wait for the issue to be triaged. Comment `/unassign` at any time to release it. The bot will decline only if somebody else already has it, or if you are at your limit: **at most two open assigned issues at a time**.

**Being assigned is not the same as us agreeing to the fix.** If the issue still says `needs-triage`, we might yet decide it isn't a bug, or that we want it solved differently. For a small, obvious fix — a wrong error message, a missing nil check — go straight ahead. For anything larger, waiting for the triage decision costs you a few days and can save you a weekend. That is a judgement call about your own time, not a rule.

If you have already had a pull request merged here, please consider leaving `good first issue` items for people who haven't — there are rarely many of them, and they are the easiest way into the project for someone new. That is a courtesy rather than a rule, and the bot will not stop you.

**If the same thing gets reported twice, the earliest report wins.** We keep the first one and close later ones against it, whether or not a pull request happens to be attached to the later one. Whoever fixes it references the original issue.

### While you are working

Please do not open a pull request for an issue that is assigned to somebody else. If you have already written the fix, say so in the issue thread rather than opening it: if the assignee has gone quiet, or is happy to hand it over, a maintainer will sort it out there.

Reference the issue with `Fixes #123` in your pull request description.

If an assigned issue shows no visible progress for two weeks, a bot will ask whether you are still on it, and will unassign it about a week later if there is no reply. No hard feelings — comment `/assign` and it is yours again.

Two things stop that clock, and you can ask for either in the issue thread:

- **`status/blocked`** — you are waiting on us rather than the other way round: a design review, an architectural decision, an upstream fix. Reminders stop until it is resolved.
- **`lifecycle/frozen`** — the work is simply a long one and will legitimately take more than a few weeks.

Before investing significant time in an implementation, consider sharing your design ideas in the issue thread or in our [community channels](https://openeverest.io/#community) (Slack and more). Early feedback from maintainers and other contributors can save effort, surface existing work, and help your PR land faster.

## Pull requests

### Keep them small

Small pull requests get reviewed faster and are more likely to be correct. Reviewer attention is the scarce resource here: someone with twenty minutes will pick up a hundred-line change and postpone a thousand-line one, possibly several times over.

- **One concern per pull request.** While implementing something you will find bad names, missing tests, weak types. Please do fix them — in a separate pull request. Unrelated changes in the same diff bury the change that actually matters.
- **Land preparatory work first.** If your change needs a refactor to fit, send the refactor on its own. It is easier to review, and it stops you rebasing it forever.
- **Don't sweep the whole repository.** A one-line change repeated across forty files needs sign-off from everyone who owns those files, and the review cost rarely justifies the benefit. If you want to do one of these, split it by area and open the first one to find out whether we agree before doing the rest.
- **Style-only and linter-only changes need agreement first.** Ask in the issue or in our [community channels](https://openeverest.io/#community) before opening them. A linter we have not enabled is usually a decision rather than an oversight, and reformatting churn makes every open pull request conflict.
- **Trivial edits cost a review too.** A single typo fix is rarely worth one on its own — if you are fixing one, read the rest of the file and fix everything you find there.

Open it as a **draft** while you are still working, and mark it ready for review when you want eyes on it. Explain the *why* in the description; the *what* is already in the diff.

### Getting it reviewed

Give it time before chasing it: a pull request opened this morning has not gone quiet, it has just been opened. Once a week or so has passed with no response at all:

1. Check CI is green and there are no merge conflicts — nobody starts a review on a red pull request.
2. Check the description says why the change is needed, not just what it does.
3. Comment on the pull request asking for a review. This is welcome, not nagging.
4. Bring it to our [community channels](https://openeverest.io/#community), or to a [community meeting](https://github.com/openeverest#openeverest-community-meetings).

We would much rather be nudged than have your work sit there while you assume we have quietly said no.

### It's fine to push back

Reviewers get things wrong. If you have a good reason for doing something a particular way, say so — you might be overruled, but you might also be right, and we would rather have the argument than have you silently apply a change you think is worse.

## AI-assisted contributions

Use whatever tools help you. We care about the result, not how you produced it. Two rules make that workable.

**You must understand what you submit.** If you cannot explain why a change is correct, or defend it in review, it is not ready. The same applies to issues: if you cannot explain why something is a bug, it is not ready to file.

**Never present output you did not produce.** Logs, stack traces, race detector output, benchmark numbers and error messages must be things you actually observed. Reasoning from reading the code is completely legitimate — say that is what you did. *"I haven't run this, here is the reasoning"* is a good report. Invented output presented as observed is not, and it is worse than no evidence at all: it manufactures confidence that a problem was reproduced, and it forces a reviewer to audit the report before they can start on the claim itself.

Please say when a contribution is substantially AI-generated. Reports and pull requests containing unverified claims presented as fact will be closed without detailed review, and repeated submissions of that kind may lead to a temporary interaction limit.

## Contributing to the source code

### Which branch to target

| Branch | Contents | Target it for |
| --- | --- | --- |
| `main` | OpenEverest v2 (Developer Preview) | New features and fixes |
| `v1.x` | OpenEverest v1 (current release) | Fixes for the 1.x line |

Unless an issue says otherwise, open your pull request against `main`.

`main` carried v1 until 18 August 2026, when the two lines swapped branches: v2
moved from `release-2.0` to `main`, and v1 moved to `v1.x`. Both lines are
still developed. This changes nothing about the
[v1 lifecycle](https://openeverest.io/blog/v2-developer-preview-release/#timeline):
v1 remains the released version, is still maintained, and only enters
maintenance mode three months after v2 reaches GA.

#### If you cloned or forked before 18 August 2026

Your `main` still holds v1 code, so a branch cut from it targets the wrong
codebase. Check with:

```sh
git branch -vv | grep -E '\[origin/(main|release-2\.0)'
```

A local `main` reporting a large `ahead N, behind M` is v1 code tracking v2.
Do not `git pull` it — that merges v2 into v1.

**Working from a fork.** Your fork was not renamed. If you only work on v2 and
have nothing unpushed on `main`, re-fork, or press **Sync fork** on your fork's
`main` and choose *Discard commits*. To keep working on v1 as well, rename in
your fork first — that is what makes the push below a plain create rather than
a force-push:

1. Fork on GitHub: Settings → Branches → rename `main` to `v1.x`.
2. Then locally:

```sh
git remote add upstream https://github.com/openeverest/openeverest.git  # if missing
git fetch --all --prune
git branch -m main v1.x
git branch -u origin/v1.x v1.x
git checkout -b main upstream/main
git push -u origin main
git remote set-head origin -a
```

3. Fork on GitHub: Settings → General → set the default branch back to `main`.

**Pushing directly to this repository.** Rename `main` first to free the name:

```sh
git fetch origin --prune
git branch -m main v1.x
git branch -u origin/v1.x v1.x
git branch -m release-2.0 main   # skip if you never had it
git branch -u origin/main main
git remote set-head origin -a
```

[openeverest/helm-charts](https://github.com/openeverest/helm-charts) moved the
same way, with `v2` in place of `release-2.0`.

### Backend

The backend is written in Go. To set up a full local development environment — including a local Kubernetes cluster, the Everest operator, and all dependent services — follow the [Backend Development Guide](https://github.com/openeverest/openeverest/blob/main/dev/README.md).

### Frontend

The frontend is a TypeScript/React monorepo managed with PNPM and Turborepo. For details on the UI stack, local development setup, and available scripts, see the [Frontend Development Guide](https://github.com/openeverest/openeverest/blob/main/ui/README.md).

### Signing Your Work (Developer Certificate of Origin)

Each commit must be signed off. By doing so, you confirm that you have the right to license your contribution under the project's license. See [Developer Certificate of Origin](https://developercertificate.org/).

Use `-s` if you have `user.name` and `user.email` configured in Git:

```bash
git commit -s -m "your commit message"
```

Or add it manually in the commit message:

```
your commit message

Signed-off-by: Your Name <your.email@example.org>
```

To always sign off automatically, set a Git alias:

```bash
git config --global alias.ci "commit -s"
git ci -m "your commit message"
```

## UI Changes

When a pull request touches the UI, include visual context so reviewers can understand the change without having to run the app locally.

### Screenshots

Attach screenshots when your change affects layout, styling, component appearance, or static content. Use a **before/after** format:

- **Before** — screenshot of the existing behavior
- **After** — screenshot with your change applied

### Video demos

Record a short screen capture for changes that involve motion or interaction: multi-step workflows, form flows, animations, hover states, or anything that is easier to understand in motion than in a still image. Keep recordings short and focused — trim dead time at the start and end.

GitHub accepts GIF and MP4 attachments directly in PR descriptions.

## Local quality checks

Before opening a PR, run local checks to keep CI green.

### Copyright headers

Every `*.go`, `*.ts`, and `*.tsx` source file must carry an Apache 2.0 copyright header.

To check files you changed in your branch run from the repository root:

```bash
make copyright-check
```

To automatically add missing headers to files you changed in your branch, run:

```bash
make copyright-headers
```

CI runs the check-only mode and reports files that are missing headers.

The command detects files that were added or modified relative to `main` (using `git merge-base`) plus any new untracked source files, and inserts the header where it is missing.

Files that contain `This file was auto-generated` are skipped automatically.
You can also exclude files or folders using `.copyrightignore` in the repository root.

You can also target specific files explicitly:

```bash
make copyright-check FILES="path/to/file.go path/to/file.ts"
make copyright-headers FILES="path/to/file.go path/to/file.ts"
```

For paths that contain spaces, pass a newline-delimited file list:

```bash
printf '%s\n' "path with spaces/file.ts" "another/path.go" > /tmp/changed_files.txt
make copyright-check FILES_FILE=/tmp/changed_files.txt
make copyright-headers FILES_FILE=/tmp/changed_files.txt
```

Or override the base branch:

```bash
make copyright-check BASE_BRANCH=develop
make copyright-headers BASE_BRANCH=develop
```

## Testing

When contributing new features or bug fixes, please include appropriate tests to ensure code quality and prevent regressions.

### Test Types

- **Unit tests**: For Go backend code, add or update `*_test.go` files alongside your changes
- **API integration tests**: For API changes, add tests in the `api-tests/` directory
- **CLI integration tests**: For CLI changes, add tests in the `cli-tests/` directory

### Running Tests Locally

Before submitting a PR, run the relevant tests:

```bash
# Unit tests (fast, no dependencies)
make test

# API integration tests (requires local Kubernetes cluster)
make k3d-cluster-up
make -C api-tests test

# CLI integration tests (requires local Kubernetes cluster)
make -C cli-tests test-cli
```

### CI Requirements

All pull requests must pass the automated test suite before merge. The CI pipeline runs:

- Unit tests (`make test`)
- API integration tests
- CLI integration tests
- Linting and code quality checks

## Community Meetings

We extend a warm welcome to everyone to join our community meetings. For details on schedules and how to participate, [see here](https://github.com/openeverest#openeverest-community-meetings)
