# The contract

OpenEverest core and its extensions (providers, backup and monitoring extensions, plugins) share a surface: the CRDs, the HTTP API, the conventions of webhooks and status, and the Go identifiers and rules both sides use. That surface is the **contract**. This package holds its version, the check that decides whether an extension can run on a core, and the keys of the annotations that carry versions. Package documentation: [doc.go](doc.go).

## The contract version

The contract version is the SemVer of the last core release that changed the contract. It is a core release number, not a counter of its own, so a message can say "needs OpenEverest 2.1.0 or later" without a lookup table, and a release that does not change the contract strands no extension.

A core serves extensions built for any contract version in its window `[MinVersion, Version]`. Extensions never pick a number: their tooling stamps each build with the `Version` of the core it was built against, and every enforcement point checks `MinVersion ≤ stamp ≤ Version` by plain SemVer 2.0.0 precedence (`Check`).

Between releases, the version is a development value that sorts after the last release and before any release `main` can build next, so nobody has to guess the next release's name:

| Last release | Development value | Example | Why |
|---|---|---|---|
| a pre-release R | `R.next` | `2.0.0-alpha.1.next` | SemVer sorts nothing between two pre-releases but R with more identifiers, which is also how Go pseudo-versions (`v2.0.0-alpha.1.0.<time>-<hash>`) and Kubernetes CI builds (`v1.34.0-alpha.1.123+<sha>`) name what follows a pre-release. |
| a release X.Y.Z | `X.(Y+1).0-alpha.0` | `2.1.0-alpha.0` | Patch releases come from release branches, so `main` builds the next minor release; Kubernetes tags its main branch `vX.(Y+1).0-alpha.0` when it cuts a release branch. It sorts above every X.Y patch release and below `X.(Y+1).0-alpha.1`. |

`IsDevelopment` recognises a development value by its last pre-release identifier being `next`, or its pre-release being exactly `alpha.0`, and nothing else, so no release can be named either way. The word is `next`, not `dev`: `dev` means *before* a release in Python's and Maven's versions, and core dev builds such as `2.0.0-dev.4` are versions users can install, which sort above every `2.0.0` alpha. A word after a release would not work either: `2.1.0-next` sorts above `2.1.0-alpha.1`.

An extension stamped with a development value was built against an unreleased core and must not be published. Messages to users never show these values: they say "an unreleased OpenEverest build".

## How the version moves

Nobody bumps the version by hand. `make gen-contract`, part of `make gen`, works it out and writes it to these generated files, which must not be edited:

| File | Holds |
|---|---|
| `VERSION` | `Version`, for tools that read it from a git ref without building Go |
| `MIN_VERSION` | `MinVersion` |
| `zz_generated.version.go` | `Version`, `MinVersion`, `Fingerprint`, `UpgradableFrom` |

It does so in two steps.

**1. Fingerprint the contract.** [`hack/gen-contract`](../hack/gen-contract) hashes the contract artifacts, listed once in its `inputs`:

- every CRD in `config/crd/bases`;
- the HTTP API: `api/openapi/http-api.yaml` and the CRD schemas it refers to, `api/openapi/crds.gen.yaml`;
- the built-in BackupStorageClasses in `config/backupstorageclasses`;
- the exported Go API of `contract/...` and `backupstorageclasses/...`;
- [`CHANGES`](CHANGES).

It first strips what does not change the contract, so the fingerprint changes when the contract does and only then:

- documentation: every `description`, `summary` and `example`, and the `message` of CEL rules, whose `rule` stays; a property *named* `description` stays;
- the printer columns of CRDs, which only `kubectl get` uses;
- the `metadata` of manifests, which holds controller-gen's version and the annotations tools stamp;
- the display name, description and `uiSchema` of built-in classes;
- in Go: comments, layout, parameter names, function bodies, unexported identifiers and test files.

The result is canonical JSON with sorted keys, or one line of tokens per Go declaration, so it is the same on every machine.

**2. Compare with the latest release.** Every core release has a record in [`releases/`](releases), named after the release:

```yaml
version: 2.0.0-alpha.1          # the core release
contractVersion: 2.0.0-alpha.1  # last release that changed the contract
minContractVersion: 2.0.0-alpha.1
type: fresh-install             # fresh-install | upgradable
upgradableFrom: []              # core releases; empty for fresh-install
fingerprint: ""                 # empty: release predates fingerprinting
```

The latest record is the one with the highest `version`. While the fingerprint equals the latest record's, `Version` is that record's `contractVersion` and `UpgradableFrom` its `upgradableFrom`. Once the fingerprint differs, `Version` is the development value after the latest release, and `UpgradableFrom` is empty: a development build is never an upgrade target. `MinVersion` is the latest record's `minContractVersion` either way; it moves only at a release.

So a pull request that changes the contract after a release moves `VERSION` to the development value, and CI's check that `make gen` leaves no diff makes sure the regenerated files are committed. Further changes keep that value until the next release.

The first record, `2.0.0-alpha.1`, predates fingerprinting, so its fingerprint is empty and never matches: until the next release records one, `Version` is `2.0.0-alpha.1.next`.

On a release branch, `make check-contract-release` (`go run ./hack/gen-contract -check-release`) fails when the fingerprint differs from the latest record's, since a patch release must not change the contract. A record without a fingerprint (the seed) has nothing to compare with, so the check passes until the first fingerprinted release.

When the JSON Schema of the `uiSchema` lands, it joins the fingerprint's inputs.

## `CHANGES`: changes no artifact shows

Some changes reach extensions without changing any artifact: code that imposes a new requirement on them, or gives a field they already write a new meaning. For example, an API middleware change that runs every Instance write from the UI as a dry run through each provider's webhook first changes no CRD and no API schema, yet breaks every provider whose webhook declares side effects.

Record such a change in [`CHANGES`](CHANGES), on a line of its own naming the pull request, for example:

```
#1234: the API server sends every Instance write from the UI through the provider's webhook as a dry run first, so provider webhooks must declare sideEffects None or NoneOnDryRun.
```

then run `make gen-contract`. Every line of the file is part of the fingerprint, so the version moves as it does for any other contract change. The file has no comment syntax, so nothing written in it can fail to count; blank lines and trailing spaces do not count.

## The decision labels

The fingerprint cannot see a change like the one above, so a person has to say whether there is one. While the contract is unchanged since the last release (`VERSION` on the base branch is a release value, not a development value), a pull request that touches a **contract-adjacent** path needs exactly one of these labels:

| Label | Means | Requires |
|---|---|---|
| `contract-change` | An extension built against the base branch could break or need to change. | `VERSION` in the pull request is the development value: if no artifact changed, record the change in `CHANGES` and run `make gen-contract`. |
| `no-contract-change` | It does not. | `VERSION` in the pull request is still the release value. A pull request that changes a contract artifact changes the contract, whatever its label. |

Once the contract has changed since the last release, the next release changes it anyway, so no decision is needed until then.

The contract-adjacent paths are listed once, between `# BEGIN contract-adjacent` and `# END contract-adjacent` in [`.github/CODEOWNERS`](../.github/CODEOWNERS): the contract and its artifacts, the SDK extensions link, API middleware, admission (webhooks, admission policies, the API's validation handler and the packages they decide through), the readers of the status extensions write, and the finalizers core holds on objects extensions install. Each line says why its paths are there.

[`.github/workflows/contract-decision.yaml`](../.github/workflows/contract-decision.yaml) runs the check on pull requests into `main` and `release-*`, again whenever a label changes. Its logic is [`hack/contract-decision`](../hack/contract-decision), which runs locally too:

```sh
git diff --name-only origin/main... | go run ./hack/contract-decision \
  -base-version "$(git show origin/main:contract/VERSION)" \
  -head-version "$(cat contract/VERSION)" \
  -label no-contract-change
```
