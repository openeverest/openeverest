# Release Process

This document describes how OpenEverest is versioned and released.

## Versioning

OpenEverest follows [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`.

| Kind | Example | Contents |
| ---- | ------- | -------- |
| Major | `2.0.0` | Breaking changes to APIs, CRDs or upgrade paths. |
| Minor | `1.17.0` | New features, backwards compatible. |
| Patch | `1.17.1` | Bug and security fixes only. |
| Dev build | `2.0.0-dev.4` | Snapshot of ongoing development. Not supported for production. |
| Alpha | `2.0.0-alpha1` | Early preview of a major release. Features and APIs may still change. Not supported for production. |
| Beta | `2.0.0-beta1` | Feature-complete preview of a major release, for wider testing. Not supported for production. |
| Release candidate | `1.17.0-rc1` | Candidate build for testing before a release. |

A major release usually goes through dev builds, then alphas, betas and release candidates before GA. Pre-release versions are published as GitHub pre-releases.

## Release Lines

| Line | Branch | Status |
| ---- | ------ | ------ |
| v2 | `main` | Active development. Dev, alpha and beta builds, then release candidates, until v2.0 GA. |
| v1 | `v1.x` | Receives features until v2.0 GA. After v2.0 GA: bug and security fixes only, for **6 months**, then end of life. |

Each minor release gets a `release-X.Y` branch, which receives patch releases. Security fixes are provided for the versions listed in [SECURITY.md](SECURITY.md#supported-versions).

## Cadence

There is no fixed schedule. Minor releases ship when their features are ready, roughly monthly in practice. Patch releases ship as needed. Security fixes follow the timelines in [SECURITY.md](SECURITY.md#response-process).

## Planning and Tracking

- Long-term direction is on the [public roadmap](https://github.com/orgs/openeverest/projects/1).
- Day-to-day development is tracked on the [OpenEverest Development](https://github.com/orgs/openeverest/projects/3) project board.
- Every release, including pre-releases, has a [milestone](https://github.com/openeverest/openeverest/milestones) named after its version, for example `v2.0.0-alpha1`. Issues and pull requests planned for a release are assigned to its milestone, so anyone can see what lands in it. Work that misses a release moves to the next milestone, and the milestone is closed when the release is published.

## Roles

- **Any maintainer** can propose a release, in a GitHub issue, Slack or a community meeting.
- **Release manager:** a member of the `admins` team (see [GOVERNANCE.md](https://github.com/openeverest/governance/blob/main/GOVERNANCE.md#github-permissions-and-administration)) runs the release workflows and publishes the release.

## Core Release Steps

1. **Prepare.** Make sure CI on the source branch is green, including the nightly [release e2e](.github/workflows/release-e2e.yml) run. Patches are merged to `main` or `v1.x` first and cherry-picked to the `release-X.Y` branch. For v2, once `release-X.Y` has a final release, a pull request into it must not change the contract between core and its extensions, so that patch releases never do: CI runs `make check-contract-release` on it, which shows the diff when it fails (see [contract/MAINTAINING.md](contract/MAINTAINING.md#at-a-release)).
   Check that all issues and pull requests in the release [milestone](https://github.com/openeverest/openeverest/milestones) are done or moved to a later milestone.

   Before a pre-release of a new major line, merge `main` into its release branch, for example `main` into `release-2.0` before `2.0.0-alpha.2`, in a pull request into the release branch. In the merge, take `main`'s side of the files `make gen-contract` writes and run it again. Every v2 release is recorded in `contract/releases`, so the release workflow refuses a branch without the contract, and the nightly contract-verify fails on `release-2.0`, which was cut before the contract, until that merge (see [contract/MAINTAINING.md](contract/MAINTAINING.md#branches-without-the-contract)).
2. **Pre-release.** Run the release workflow with the pre-release version, such as `X.Y.Z-rcN` or, for a major release, `X.0.0-alphaN` / `X.0.0-betaN`:
   - v2: [Release](.github/workflows/release.yml) from `main`;
   - v1: **Release v1** from the `v1.x` branch.

   The workflow creates or updates the `release-X.Y` branch and the tag, builds multi-arch images, and pushes them to GHCR with provenance and SBOM attestations. For v1, it also builds signed `everestctl` binaries and creates a draft GitHub release.

   For v2, also say how users get to the release: `release_type` `fresh-install` (the default) when they must install it fresh, or `upgradable` with `min_contract_version`, the oldest contract version it serves, and `upgradable_from`, the releases it can be upgraded from in place. The releases in `upgradable_from` must be tagged: their records are read from their tags. Before tagging, the workflow checks that a patch release keeps the contract of its line, records the release in [contract/releases](contract/releases), which publishes its window on the CRDs the tag ships, and pushes the branch and the tag together. It then opens a pull request against `main` with the record, with auto-merge on: approve it, since `main` works out its contract version from the latest release it has a record of; if another record pull request merged first, re-run the workflow's "Record the release on main" job. Records are frozen once released, and a re-run of the workflow keeps the record and, if the tag is missing, tags the commit that recorded it. A dev build gets no record. How the contract version moves and how releases are recorded: [contract/MAINTAINING.md](contract/MAINTAINING.md#at-a-release).
3. **Test.** Validate the pre-release (e2e and upgrade from the previous release). Fix issues and cut a new pre-release if needed.
4. **Release.** Run the same workflow with version `X.Y.Z`.
5. **Helm chart.** Run [Everest Chart Release](https://github.com/openeverest/helm-charts/actions/workflows/release.yaml) in `openeverest/helm-charts` with the same version.
6. **Publish.** Write release notes using the [release notes format](https://github.com/openeverest/specs/blob/main/specs/004-release-notes-format.md), publish the GitHub release, and close the milestone. Publish the [documentation](https://github.com/openeverest/docs) by tagging the docs repository with the same version (see its [README](https://github.com/openeverest/docs#releasing-docs)). Publishing the release triggers the [Homebrew tap](https://github.com/openeverest/homebrew-tap) update.
7. **Announce** in [#openeverest-users](https://cloud-native.slack.com/archives/C09RRGZL2UX) on CNCF Slack.

## Providers, Plugins and Other Components

Providers (`provider-*`), plugins (`plugin-*`), monitoring integrations (`monitoring-*`), `provider-sdk` and the Helm charts are versioned and released **independently** of core, each with its own SemVer version.

- Each repository has a manual release workflow that only repository admins can run. It takes `X.Y.Z` or `X.Y.Z-<suffix>`, tags the release, and publishes images or charts to GHCR.
- Dev images are published on every push to `main`.
- Each component should document the core and upstream versions it supports in its README.
- Breaking changes require a major version bump (or a minor bump while the component is `0.x`) and an entry in the release notes.
