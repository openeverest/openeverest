# The contract

OpenEverest core and its extensions (providers, backup and monitoring extensions, plugins) share a surface: the CRDs, the HTTP API, the conventions of webhooks and status, and the Go identifiers and rules both sides use. That surface is the **contract**. This package holds its version, the check that decides whether an extension can run on a core, and the keys of the annotations that carry versions. Package documentation: [doc.go](doc.go).

## The contract version

The contract version is a core release number, not a counter of its own, so a message can say "needs OpenEverest 2.1.0 or later" without a lookup table. A pre-release, a minor release and a major release each have their own version as contract version. A patch release of a final release cannot change the contract, so it keeps the contract version of its line, and extensions built against it run on every release of that line (see [Release records](#release-records)).

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
| `zz_generated.version.go` | `Version`, `MinVersion`, `FingerprintScheme`, `Fingerprints`, `UpgradableFrom` |

It does so in two steps.

**1. Fingerprint the contract.** [`hack/gen-contract`](../hack/gen-contract) hashes the contract artifacts, which its `inputs` declare once, grouped into logical inputs. An input is named after what it is, not where it lives, so moving a file does not rename it:

| Input | Covers |
|---|---|
| `crds` | every CRD in `config/crd/bases` |
| `http-api` | `api/openapi/http-api.yaml` and the CRD schemas it refers to, `api/openapi/crds.gen.yaml` |
| `go-api` | the exported Go API of `contract/...` |
| `changes` | the entries of [`CHANGES`](CHANGES) |

When the JSON Schema of the `uiSchema` lands, it joins the inputs under a new scheme.

The BackupStorageClasses this repository ships, such as `s3`, are not inputs: a class is an extension that only ships here for convenience, and covering it would raise every extension's minimum OpenEverest whenever the class changed. The `BackupStorageClass` and `BackupStorage` kinds are core contract, and `crds` covers them.

It first strips what does not change the contract, so a digest changes when its part of the contract does and only then:

- documentation: every `description`, `summary` and `example`, and the `message` of CEL rules, whose `rule` stays; a property *named* `description` stays;
- the printer columns of CRDs, which only `kubectl get` uses;
- the `metadata` of manifests, which holds controller-gen's version and the annotations tools stamp;
- in Go: comments, layout, parameter names, function bodies, unexported identifiers and test files;
- in `CHANGES`: comments and blank lines.

Each artifact then has a canonical form, compact JSON with sorted keys and exact numbers, or one line of tokens per Go declaration, or one line per change. The **canonical stream** of an input is its artifacts in order of name, each as its name, a space, the length of its canonical form, a newline, the form and a newline. The **digest** of the input is `sha256:` and the SHA-256 of that stream, the same on every machine. An input none of whose artifacts exist, as at a tag that predates it, has no digest, except `changes`: a tree without `CHANGES` has the empty log.

**2. Compare with the latest release.** Every core release has a [record](#release-records) in [`releases/`](releases), and the latest record is the one with the highest `version`. The generator compares the digests of the tree with those the latest record holds under the current [fingerprint scheme](#fingerprint-schemes):

- An input the record holds is compared; it differs when its digest does, or when the tree lacks it.
- An input the tree has and the record does not was absent from the tag of the release, so there is nothing to compare it with: it is not compared, and the generator names it in a notice.

While no compared input differs, `Version` is the latest record's `contractVersion` and `UpgradableFrom` its `upgradableFrom`. Once one differs, `Version` is the development value after the latest release, and `UpgradableFrom` is empty: a development build is never an upgrade target. The generator says which inputs differ, and [`make contract-diff`](#seeing-what-changed) shows how. `MinVersion` is the latest record's `minContractVersion` either way; it moves only at a release.

So a pull request that changes the contract after a release moves `VERSION` to the development value, and CI's check that `make gen` leaves no diff makes sure the regenerated files are committed. Further changes keep that value until the next release.

**No silent flips.** When the latest record has no digests under the current scheme, nothing can tell whether the contract changed since that release, and either guess would move the version for no reason or keep it while the contract changed. So `make gen-contract`, and every command that compares, fails everywhere with the commands that add them:

```sh
git fetch upstream --tags   # or origin: the remote of openeverest/openeverest
go run ./hack/gen-contract rebaseline -version 2.1.0
```

**The seed.** The first record, `2.0.0-alpha.1`, is exempt by name. Its tag carries no `contract/` directory, and the contract package was added before any check decided whether a change touched the contract, so digests computed from that tag would not say what `main` changed since. Its record has no `fingerprints`, and while it is the latest record, `Version` is the development value after it, `2.0.0-alpha.1.next`. The next release's record ends the exemption.

## Release records

A record is a file per release, `releases/<version>.yaml`, much as Cluster API providers declare the contract of each release series in `metadata.yaml` and Go records the API of each release in `api/go1.N.txt`: the decided facts of a release belong next to the code, in review, readable from any ref. A release adds its record on its release branch before it is tagged, so the tag carries it.

The paths of these files, `contract/VERSION`, `contract/MIN_VERSION`, `contract/CHANGES` and `contract/releases/<version>.yaml`, are permanent from 2.0.0-alpha.2, the first release whose tag carries them: workflows and tests read them from any branch or tag with `git show <ref>:<path>`, so a move would break every reader of an older ref. `contract/releases` holds only records; `check-records` refuses anything else there, a Go file included.

```yaml
# OpenEverest release record: data for hack/gen-contract, not a Kubernetes resource.
apiVersion: contract.openeverest.io/v1
kind: ReleaseRecord
version: 2.1.0                 # computed
contractVersion: 2.1.0         # computed
minContractVersion: 2.0.0      # decided
type: upgradable               # decided
upgradableFrom: [2.0.3]        # decided
fingerprints:                  # computed; add-only; key = fingerprint scheme
  c1:
    crds: sha256:<hex>
    http-api: sha256:<hex>
    go-api: sha256:<hex>
    changes: sha256:<hex>
```

Decided by whoever cuts the release:

| Field | Holds |
|---|---|
| `type` | `fresh-install`: users install the release fresh; `upgradable`: it can be upgraded to in place from the releases `upgradableFrom` lists |
| `minContractVersion` | the oldest contract version the release serves |
| `upgradableFrom` | the core releases it can be upgraded from in place; empty for a fresh-install release |

Computed by `hack/gen-contract`:

| Field | Holds |
|---|---|
| `version` | the core release |
| `contractVersion` | the contract version extensions built against the release are stamped with: the release itself for a pre-release or a minor or major release; for a patch release of a final release, the `contractVersion` of the release before it on its line, since a patch release cannot change the contract |
| `fingerprints` | the digests of the contract the release was cut with, by scheme, then by input |

The generator rejects a record with another `apiVersion` or `kind`, or a field it does not know. Every `apiVersion` a release has shipped stays readable for good: tags carry their records in the format of their time, and the commands that rebaseline a record or check one against its tag read them. A new format adds a decoder next to the old ones.

Every record must hold: valid SemVer versions without build metadata or development values; `minContractVersion ≤ contractVersion ≤ version`; a `type` that agrees with `upgradableFrom`, which lists older releases once each; and `fingerprints`, which only the seed lacks, so removing them cannot turn a check off. Across records, sorted by `version`, `contractVersion` never goes down. `minContractVersion` may: a fresh-install release serves only its own extensions, and a later upgradable release can serve older ones again, as long as each of its upgrade paths keeps the extensions of the release it starts from working: its `minContractVersion` is at most the `contractVersion` of every release in `upgradableFrom`.

**Why digests per input, named by scheme.** A bare digest of everything would say only that something changed, and the moment the way it is computed changes, every recorded digest becomes incomparable. The record keeps one digest per logical input under the name of the scheme that computed it, as `go.sum` names its hash (`h1:`): a message can name the inputs that differ, a scheme bump adds digests next to the old ones instead of rewriting a release's history, and every digest stays verifiable from the tag it was computed from.

### Fingerprint schemes

A scheme, `c1` today, is the set of inputs, what each strips and the canonical stream. Any change to them that can change the digest of an unchanged contract (an input added, removed or regrouped, a strip rule, the canonical form) must name a new scheme (`c2`, ...) in `hack/gen-contract`. The golden test (`TestGolden`, in `hack/gen-contract/testdata/golden`) pins the canonical stream of every input of a frozen tree, so a change to the normaliser fails until the scheme is bumped and its golden files are written. Then add the new scheme's digests to the latest record:

```sh
go run ./hack/gen-contract rebaseline -version 2.1.0   # -tag v2.1.0 by default
```

`rebaseline` computes the digests from the tree of the release tag, reading only the objects of that tag, so a tag fetched alone at depth 1 is enough. It adds `fingerprints.<scheme>` to the record and changes nothing else; it refuses when the record holds the scheme already with other digests, and it refuses the seed. An input absent from the tag gets no digest, and a missing `CHANGES` the empty log.

Only `main` allocates a scheme. A release branch pins the generator's dependencies, or cherry-picks `main`'s generator together with its scheme, so two branches never mean different things by the same name. A pull request that adds an input group also adds its paths to the contract-adjacent block of [`.github/CODEOWNERS`](../.github/CODEOWNERS); `TestInputsAreContractAdjacent` checks it.

[`.github/workflows/contract-verify.yaml`](../.github/workflows/contract-verify.yaml) recomputes the latest record's digests under the current scheme from its tag, with the branch's own generator (`go run ./hack/gen-contract verify`), and fails with "the fingerprint scheme changed: bump it and run rebaseline" when they differ: a change to the generator, its dependencies or Go then changed how an unchanged contract hashes. It runs on pull requests into `main` and `release-*` that touch `hack/gen-contract`, `go.mod`, `go.sum` or the records, and nightly over `main` and every v2 release branch. While the latest record is the seed, there is nothing to verify.

### Records are frozen

A record says what a release was cut with, so it never changes after the release: a change under `releases/` may only add a record, which must equal the one the tag of its release carries, or add `fingerprints.<scheme>` that equal what `rebaseline` computes from that tag. Deleting or renaming a record, editing a field or a recorded digest, and adding a record whose tag does not exist all fail: there is no exemption for a missing tag, so nobody can hand-write a record. The seed, which no tag carries, is only checked for not being modified once merged.

The records job of [`.github/workflows/contract-decision.yaml`](../.github/workflows/contract-decision.yaml) runs the check, `go run ./hack/gen-contract check-records`, on pull requests into `main` and `release-*`, from the files API, and on pushes to them, from the diff of the push. The push run detects an edit that reached a branch without a pull request; it cannot prevent it.

### Seeing what changed

```sh
make contract-diff                  # against the tag of the latest release
make contract-diff REF=v2.1.0 INPUT=crds
make contract-diff RAW=1 INPUT=crds | sha256sum   # the digest of crds
```

For each input, `make contract-diff` (`go run ./hack/gen-contract diff`) prints whether its digest changed since the ref and a unified diff of the canonical forms, with JSON indented and one Go declaration per line, capped at 200 lines per input (`-max-lines 0` prints everything). `RAW=1` prints the exact canonical stream each digest hashes. The ref must be here; fetch a tag with `git fetch --depth=1 origin refs/tags/v2.1.0:refs/tags/v2.1.0`.

### Release branches

A patch release must not change the contract. `isPatchOf(latest, v)` decides when that applies: the latest record is a final release `X.Y.Z`, without a pre-release, and `v` has the same `X.Y`. Then `make check-contract-release` (`go run ./hack/gen-contract -check-release -branch release-X.Y`, or `-version X.Y.Z`) fails when a compared input differs from the latest record, and prints the diff when the tag of that release is here. Otherwise, after a pre-release or for the first release of a new minor, the release may change the contract, and the check passes with a notice.

## `CHANGES`: changes no artifact shows

Some changes reach extensions without changing any artifact: code that imposes a new requirement on them, or gives a field they already write a new meaning. For example, an API middleware change that runs every Instance write from the UI as a dry run through each provider's webhook first changes no CRD and no API schema, yet breaks every provider whose webhook declares side effects.

Record such a change in [`CHANGES`](CHANGES) as an entry, on a line of its own naming the pull request:

```
- #1234: the API server sends every Instance write from the UI through the provider's webhook as a dry run first, so provider webhooks must declare sideEffects None or NoneOnDryRun.
```

then run `make gen-contract`. The entries are the `changes` input, so the version moves as it does for any other contract change. Comments (`#` alone or followed by a space) and blank lines are not hashed, so the file can explain itself without moving the version. Every other line must be an entry, `- #<pull request>: <text>`, or `make gen-contract` fails, so nothing meant as an entry can fail to count. Entries are hashed in the order of the file; add new ones at the end.

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
