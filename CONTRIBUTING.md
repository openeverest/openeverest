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
- Let us know when your talk about OpenEverest is accepted at a conference!
- Release management
- Problems found while setting up the development environment

For development contributions, please refer to the separate sections below.

## Ask for Help

The best way to reach us with a question when contributing is to join our community channels at [openeverest.io/#community](https://openeverest.io/#community) (Slack and more), or start a new [GitHub Discussion](https://github.com/openeverest/openeverest/discussions).

## Raising Issues

When raising [Issues](https://github.com/openeverest/openeverest/issues), please follow the template and fill the correponding fields. Details matter.

If you are trying to report a vulnerability, please refer to our [Security Policy](https://github.com/openeverest/openeverest/blob/main/SECURITY.md).

## Working on Issues

To avoid duplicated effort and keep collaboration smooth, please follow these steps before starting work on an issue.

**Check for an existing assignee.** If an issue is already assigned to someone, it means they are actively working on it. Please do not start work on or open a PR for an assigned issue. If the issue appears stale (no activity for two weeks or more), you are welcome to leave a comment asking about its status.

**Claim unassigned issues before you start.** If an issue is unassigned and you want to tackle it, leave a comment expressing your intent — for example, *"I'd like to work on this"*. Wait for a maintainer to assign it to you before opening a PR. This prevents two people from solving the same problem in parallel.

**Exception: `good first issue`.** Issues labeled [`good first issue`](https://github.com/openeverest/openeverest/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) are explicitly reserved for new contributors and can be picked up without prior assignment. Feel free to start working and open a PR when ready.

**Discuss your approach with the community.** Before investing significant time in an implementation, consider sharing your design ideas in the issue thread or in our [community channels](https://openeverest.io/#community) (Slack and more). Early feedback from maintainers and other contributors can save effort, surface existing work, and help your PR land faster.

## Contributing to the source code

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
