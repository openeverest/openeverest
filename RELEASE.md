# Release Process

This document describes how OpenEverest is versioned and released.

## Versioning

OpenEverest follows [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`.

| Kind | Example | Contents |
| ---- | ------- | -------- |
| Major | `2.0.0` | Breaking changes to APIs, CRDs or upgrade paths. |
| Minor | `1.17.0` | New features, backwards compatible. |
| Patch | `1.17.1` | Bug and security fixes only. |
| Release candidate | `1.17.0-rc1` | Candidate build for testing before a release. |
| Dev build | `2.0.0-dev.4` | Preview of unreleased work. Not supported for production. |

## Release Lines

| Line | Branch | Status |
| ---- | ------ | ------ |
| v2 | `main` | Active development. Dev builds until v2.0 GA. |
| v1 | `v1.x` | Receives features until v2.0 GA. After v2.0 GA: bug and security fixes only, for **6 months**, then end of life. |

Each minor release gets a `release-X.Y` branch, which receives patch releases. Security fixes are provided for the versions listed in [SECURITY.md](SECURITY.md#supported-versions).

## Cadence

There is no fixed schedule. Minor releases ship when their features are ready, roughly monthly in practice. Patch releases ship as needed. Security fixes follow the timelines in [SECURITY.md](SECURITY.md#response-process).

Planned work is tracked on the [public roadmap](https://github.com/orgs/openeverest/projects/1).

## Roles

- **Any maintainer** can propose a release, in a GitHub issue, Slack or a community meeting.
- **Release manager:** a member of the `admins` team (see [GOVERNANCE.md](https://github.com/openeverest/governance/blob/main/GOVERNANCE.md#github-permissions-and-administration)) runs the release workflows and publishes the release.

## Core Release Steps

1. **Prepare.** Make sure CI on the source branch is green, including the nightly [release e2e](.github/workflows/release-e2e.yml) run. Patches are merged to `main` or `v1.x` first and cherry-picked to the `release-X.Y` branch.
2. **Release candidate.** Run the release workflow with version `X.Y.Z-rcN`:
   - v2: [Release](.github/workflows/release.yml) from `main`;
   - v1: **Release v1** from the `v1.x` branch.

   The workflow creates or updates the `release-X.Y` branch and the tag, builds multi-arch images, and pushes them to GHCR with provenance and SBOM attestations. For v1, it also builds signed `everestctl` binaries and creates a draft GitHub release.
3. **Test.** Validate the RC (e2e and upgrade from the previous release). Fix issues and cut a new RC if needed.
4. **Release.** Run the same workflow with version `X.Y.Z`.
5. **Helm chart.** Run [Everest Chart Release](https://github.com/openeverest/helm-charts/actions/workflows/release.yaml) in `openeverest/helm-charts` with the same version.
6. **Publish.** Write release notes using the [release notes format](https://github.com/openeverest/specs/blob/main/specs/004-release-notes-format.md), publish the GitHub release, and update the [documentation](https://github.com/openeverest/everest-doc). Publishing the release triggers the [Homebrew tap](https://github.com/openeverest/homebrew-tap) update.
7. **Announce** in [#openeverest-users](https://cloud-native.slack.com/archives/C09RRGZL2UX) on CNCF Slack.

## Providers, Plugins and Other Components

Providers (`provider-*`), plugins (`plugin-*`), monitoring integrations (`monitoring-*`), `provider-sdk` and the Helm charts are versioned and released **independently** of core, each with its own SemVer version.

- Each repository has a manual release workflow that only repository admins can run. It takes `X.Y.Z` or `X.Y.Z-<suffix>`, tags the release, and publishes images or charts to GHCR.
- Dev images are published on every push to `main`.
- Each component should document the core and upstream versions it supports in its README.
- Breaking changes require a major version bump (or a minor bump while the component is `0.x`) and an entry in the release notes.
