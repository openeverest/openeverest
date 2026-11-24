# Maintaining the contract

This is for core contributors, reviewers and release managers: how the contract version moves, how releases are recorded and kept frozen, and when a pull request needs a decision label. Extension authors, who only import the package, need [README.md](README.md) and the package documentation in [doc.go](doc.go).

## The files here

| File | Kind | Holds |
|---|---|---|
| `*.go` | hand-edited | the library extensions import |
| `zz_generated.version.go` | generated | `Version` and `MinVersion` |
| `VERSION`, `MIN_VERSION` | generated | `Version` and `MinVersion`, for tools that read them from a git ref without building Go |
| `vectors.json` | hand-edited | the cases of `OrderVectors`, which every other implementation of `Check`, `IsDevelopment` and `AcknowledgementHolds` runs, such as provider-sdk's Helm template, so that it decides as they do |
| `CHANGES` | hand-edited | the contract changes no artifact shows |
| `releases/<version>.yaml` | frozen | the record of each core release |

`make gen-contract` writes the generated files: never edit them. A record is added at each release and never changes after.

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
| `zz_generated.version.go` | `Version`, `MinVersion` |

The package exports the window and nothing else that is generated, because nothing that imports it needs more. The fingerprints of the tree and their scheme belong to the generator, which computes them on every run, and to the release records, which keep those of each release; generated into the package, they would also make any two pull requests that change the same input conflict. The releases a core can be upgraded from describe the release, not the running core, so they travel with the release's manifests, where the tool that upgrades to it reads them.

A workflow that needs a fact whose reading takes a rule or parsing asks the generator rather than repeat them: `go run ./hack/gen-contract show version|min-version|scheme|development|record-schemes [-version V] [-ref R]` prints the version or the minimum of the working tree or of a ref, the fingerprint scheme the generator computes, whether a version is a development value, or the schemes a release record holds, read from the objects of a ref as `rebaseline` reads a tag. Only on a v1 ref (`v1.x`, `release-1.Y`, a `v1` tag) does a missing file print nothing; on any other it fails, saying how the branch gets the contract (see [Branches without the contract](#branches-without-the-contract)). Reading `VERSION` takes neither, so tests and workflows that only need it read it from a ref with `git show <ref>:contract/VERSION`, one reason the paths are permanent.

`make gen-contract` works the version out in two steps.

**1. Fingerprint the contract.** [`hack/gen-contract`](../hack/gen-contract) hashes the contract artifacts, which its `inputs` declare once, grouped into logical inputs. An input is named after what it is, not where it lives:

| Input | Covers |
|---|---|
| `crds` | every CRD in `config/crd/bases` |
| `http-api` | `api/openapi/http-api.yaml` and the CRD schemas it refers to, `api/openapi/crds.gen.yaml` |
| `go-api` | the exported Go API of `contract/...` |
| `vectors` | the cases of [`vectors.json`](vectors.json), which every other implementation of the rules runs, as `OrderVectors` |
| `changes` | the entries of [`CHANGES`](CHANGES) |

When the JSON Schema of the `uiSchema` lands, it joins the inputs under a new scheme.

The vectors are an input because extensions turn them into checks of their own, such as the Helm templates provider-sdk generates, so a case added or changed changes what extensions decide, even when `check.go` does not change. A change to them therefore moves the version like any other contract change: add vectors on `main`, never on a `release-X.Y` branch, whose patch releases cannot change the contract (see [Release branches](#release-branches)).

The BackupStorageClasses this repository ships, such as `s3`, are not inputs: a class is an extension that only ships here for convenience, and covering it would raise every extension's minimum OpenEverest whenever the class changed. The `BackupStorageClass` and `BackupStorage` kinds are core contract, and `crds` covers them.

It first strips what does not change the contract, so a digest changes when its part of the contract does and only then:

- documentation: every `description`, `summary` and `example`, and the `message` of CEL rules, whose `rule` stays; a property *named* `description` stays;
- the printer columns of CRDs, which only `kubectl get` uses;
- the `metadata` of manifests, which holds controller-gen's version and the annotations tools stamp;
- in Go: comments, layout, parameter names, function bodies, unexported identifiers and test files;
- in `vectors.json`: its layout and the order of the keys of its objects, while the order of its arrays stays;
- in `CHANGES`: comments and blank lines.

Each artifact then has a canonical form, compact JSON with sorted keys and exact numbers, or one line of tokens per Go declaration, or one line per change, and a key that names it by what it is rather than by the path of its file: a CRD by its name (`instances.core.openeverest.io`), the HTTP API and the CRD schemas it refers to as `http-api` and `crd-schemas`, the log as `changes`, and a Go package by its import path (`github.com/openeverest/openeverest/v2/contract`), the module path in `go.mod` and its directory, since the import path is part of its API. Moving a file therefore changes no digest, as long as the input still finds it: name the new path in `inputs` next to the old one, which the tags of earlier releases hold, so that `verify` and `rebaseline` still find it there. Two artifacts with the same key in one input fail. The **canonical stream** of an input is its artifacts in order of key, each as its key, a space, the length of its canonical form, a newline, the form and a newline. The **digest** of the input is `sha256:` and the SHA-256 of that stream, the same on every machine. An input none of whose artifacts exist, as at a tag that predates it, has no digest, except `changes`: a tree without `CHANGES` has the empty log.

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

`rebaseline` computes the digests from the tree of the release tag, reading only the objects of that tag, so a tag fetched alone at depth 1 is enough. It adds `fingerprints.<scheme>` to the record and changes nothing else; it refuses when the record holds the scheme already with other digests, when the record holds an input, under any scheme, that the new scheme does not find at the tag, since comparisons under it would skip that input, and it refuses the seed. An input absent from the tag gets no digest, and a missing `CHANGES` the empty log.

Only `main` allocates a scheme. A release branch pins the generator's dependencies, or cherry-picks `main`'s generator together with its scheme, so two branches never mean different things by the same name. A pull request that adds an input group also adds its paths to the contract-adjacent block of [`.github/CODEOWNERS`](../.github/CODEOWNERS); `TestInputsAreContractAdjacent` checks it.

[`.github/workflows/contract-verify.yaml`](../.github/workflows/contract-verify.yaml) recomputes the latest record's digests under the current scheme from its tag, with the branch's own generator (`go run ./hack/gen-contract verify`), and fails with "the fingerprint scheme changed: bump it and run rebaseline" when they differ: a change to the generator, its dependencies or Go then changed how an unchanged contract hashes. It runs on pull requests into `main` and `release-*`, and on pushes to them, that touch `hack/gen-contract`, `go.mod`, `go.sum` or the records, each time with the generator of the code it runs on; nothing else can change its result, since `go.mod` pins Go and `go.sum` the generator's dependencies. The nightly run verifies `main`, and checks through the API that every v2 release branch carries the contract (see [Branches without the contract](#branches-without-the-contract)): it never checks out another branch, because a scheduled run has the privileges of the default branch, its cache included, which another branch's code must not get. While the latest record is the seed, there is nothing to verify.

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

A patch release must not change the contract. `isPatchOf(latest, v)` decides when that applies: the latest record is a final release `X.Y.Z`, without a pre-release, and `v` has the same `X.Y`. Then `make check-contract-release` (`go run ./hack/gen-contract -check-release -branch release-X.Y`, or `-version X.Y.Z`) fails when a compared input differs from the latest record, and prints the diff when the tag of that release is here. Otherwise, after a pre-release or for the first release of a new minor, the release may change the contract, and the check passes with a notice. CI runs it on every pull request into a `release-*` branch; see [At a release](#at-a-release).

### Branches without the contract

Only v1 predates the contract: `v1.x`, and the release branches and tags of major 1 and below. Every other branch must carry `contract/VERSION` and `contract/releases`, and the checks fail closed on one that does not, rather than skip what they check:

- `contract-decision` fails a pull request whose base branch lacks them, and one that removes them. A pull request whose base branch lacks them and which adds them passes with a notice: it brings the contract to the branch.
- The nightly `contract-verify` fails on a v2 release branch, `release-X.Y` with `X` of 2 or more, that lacks them.

A release branch cut before the contract landed, such as `release-2.0`, gets it when `main` is merged into it, which is a step of preparing its next pre-release (2.0.0-alpha.2 for `release-2.0`): until then the nightly `contract-verify` fails on it, saying so. The merge pull request is the one exempt from the decision. In the merge, take `main`'s side of the files `make gen-contract` writes and run it again; the seed record then makes the branch's version `2.0.0-alpha.1.next`, as on `main`.

## Where core publishes its window

`make gen-contract` also stamps the window on the manifests core ships, after the generators that write them (`make gen-crds-manifests` and `make gen-backupstorageclasses` run it again for that reason):

| Where | Carries |
|---|---|
| Every CRD in `config/crd/bases` | `openeverest.io/contract-version` (`Version`), `openeverest.io/min-contract-version` (`MinVersion`) and `openeverest.io/upgradable-from` (the release record's `upgradableFrom`, comma-separated, empty for a release that can only be installed fresh) |
| The built-in BackupStorageClasses | `openeverest.io/contract-version`, the stamp every extension object carries |

The CRDs are where extension tooling reads the window, because:

- CRDs often have a lifecycle of their own: Helm never upgrades the CRDs in a chart's `crds/` directory, and GitOps setups often apply CRDs separately, so only the CRDs the API server serves tell which contract a cluster actually runs;
- any client that can read a CRD finds the window there, without having to guess the namespace core is installed in;
- the window changes in the same apply as the schemas it describes.

The helm-charts repository builds its CRDs and built-in classes from `config/crd` and `config/backupstorageclasses` with kustomize, so the annotations travel with them, and it copies them into the chart's `Chart.yaml`, where `everestctl upgrade` reads the `upgradable-from` of the release it upgrades to.

`GET /v1/version` does not repeat the window. It would report the constants of the API server's binary, which can disagree with the CRDs the cluster serves, and the releases a core can be upgraded from matter only before an upgrade, from the release being upgraded to, never from the running one.

The stamps are metadata, which the fingerprint ignores, so stamping cannot move the version. A reader that sees several CRDs takes their intersection (`Intersect`): the lowest `contract-version` and the highest `min-contract-version`, so it never claims more than every CRD serves while an upgrade is under way.

## At a release

The [Release workflow](../.github/workflows/release.yml) records every core release on its release branch, before it tags it. Whoever cuts the release says, in the workflow's inputs, how users get to it:

| Inputs | The release | Its record |
|---|---|---|
| `release_type: fresh-install`, the default | Users install it fresh; nothing can be upgraded to it in place. | `minContractVersion` is the release itself, so extensions built for an earlier core must be rebuilt for it, and `upgradableFrom` is empty. |
| `release_type: upgradable`, `min_contract_version`, `upgradable_from` | It can be upgraded to in place from the releases `upgradable_from` lists, comma-separated. | `minContractVersion` is `min_contract_version`, and `upgradableFrom` the list. |

The workflow checks the format of the inputs first, then runs, on the release branch:

```sh
go run ./hack/gen-contract release -version 2.1.0 -type upgradable -min 2.0.0 -upgradable-from 2.0.0,2.0.1
make gen-contract
```

The first command writes `releases/2.1.0.yaml` with the fingerprint of the branch's contract. The second finds that fingerprint in the latest record, so `VERSION` becomes the release's contract version and the CRDs carry its window and `upgradable-from`. The workflow commits both with the version it sets in the `Makefile`, and tags that commit, so the tag ships the window. Nothing is pushed until the record is written. `release` refuses:

- a version that is not SemVer 2.0.0, carries build metadata, or is a development value;
- a release that has a record already, or is not newer than the latest release recorded on the branch, since every record is worked out from the one before;
- `-min` or `-upgradable-from` for a fresh-install release, and an upgradable release without both;
- an upgradable release from a release that has no record on the branch: a release cut on another branch since this one forked needs its record merged first;
- an upgradable release whose minimum is above the contract version of the latest release, or of a release it is upgraded from. Users upgrade core first and their extensions after, so the extensions they run, and those published for the latest release, must keep working on the new core;
- a window that goes below the latest release's.

A dev build, such as `2.0.0-dev.5`, is a snapshot, not a release. It sorts above every alpha and beta of its version, so its record would stay the latest until the first release candidate: it gets none.

**The contract version of a release is the release itself**, even when the fingerprint shows the contract is the one the latest release was cut with. Keeping the latest release's contract version would tell extensions built against the new release that they also run on the one before, and the fingerprint is not evidence of that: it only shows that no artifact changed and that nobody recorded a change in `CHANGES`. The evidence will come from a job that runs the extensions built against each release candidate on the release before it, and records every break it finds in `CHANGES`. From then on, an equal fingerprint means no break, and `release` will keep the latest release's contract version, so that a release that does not change the contract strands no extension.

**`main` learns every release.** `main` works out its version from the latest release it has a record of, so after tagging, the workflow opens a pull request against `main` from the branch `contract/record-v<version>`, with the record and what `make gen-contract` derives from it on `main`. It labels the pull request with the [decision](#the-decision-labels) its `VERSION` implies. Running the workflow again for the same version keeps the record and updates the same branch and pull request.

**A patch release does not change the contract.** CI runs `make check-contract-release` on every pull request into a `release-*` branch. Change the contract on `main`, and ship it in the next minor release. When `main` is merged into a release branch, as it is before each pre-release of a major version, the files `make gen-contract` writes can conflict: take `main`'s and run `make gen-contract` on the result.

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

Once the contract has changed since the last release, the next release changes it anyway, so no decision is needed until then. A pull request into a v1 branch needs none either; one into a v2 branch without the contract fails (see [Branches without the contract](#branches-without-the-contract)).

The contract-adjacent paths are listed once, between `# BEGIN contract-adjacent` and `# END contract-adjacent` in [`.github/CODEOWNERS`](../.github/CODEOWNERS): the contract and its artifacts, the SDK extensions link, API middleware, admission (webhooks, admission policies, the API's validation handler and the packages they decide through), the readers of the status extensions write, and the finalizers core holds on objects extensions install. Each line says why its paths are there.

Markdown files never count as touched, wherever they are, so fixing this file or any other documentation needs no decision; a pull request that changes documentation and code still needs one for its code. That holds only while no Markdown file is part of the contract: `TestNoMarkdownIsContract` in `hack/gen-contract` fails when a fingerprint input, or a `//go:embed` directive of a package under a contract-adjacent path, could match one.

[`.github/workflows/contract-decision.yaml`](../.github/workflows/contract-decision.yaml) runs the check on pull requests into `main` and `release-*`, again whenever a label changes. Its logic is [`hack/contract-decision`](../hack/contract-decision), which runs locally too:

```sh
git diff --name-only origin/main... | go run ./hack/contract-decision \
  -base-ref main -base-version "$(git show origin/main:contract/VERSION)" -base-records \
  -head-version "$(cat contract/VERSION)" -head-records \
  -label no-contract-change
```
