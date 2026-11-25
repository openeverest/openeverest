# The contract

OpenEverest core and its extensions (providers, backup and monitoring extensions, plugins) share a surface: the CRDs, the HTTP API, the conventions of webhooks and status, and the Go identifiers and rules both sides use. That surface is the **contract**. This package holds its version, the check that decides whether an extension can run on a core, and the keys of the annotations that carry versions. The package documentation, in [doc.go](doc.go), describes each of them.

## The window

The contract version is a core release number, so a message can say "needs OpenEverest 2.1.0 or later" without a lookup table. A core serves extensions built for any contract version in its window `[MinVersion, Version]`. A patch release never changes the contract, so it keeps the contract version of its line.

## Development values

Between releases, `Version` is a development value that sorts after the last release and before any release `main` can build next: `R.next` after a pre-release `R`, such as `2.0.0-alpha.1.next`, and `X.(Y+1).0-alpha.0` after a release `X.Y.Z`, such as `2.1.0-alpha.0`. `IsDevelopment` recognises both. An extension stamped with one was built against an unreleased core and must not be published. Messages to users never show these values: they say "an unreleased OpenEverest build".

## Stamping and checking

- **Stamp.** Extension tooling stamps every build with the `Version` of the core it was built against, as `VersionAnnotation` on the objects core interprets and in the extension's `Chart.yaml`. Extensions never pick a number, and there are no version ranges.
- **Check.** Every enforcement point checks `MinVersion ≤ stamp ≤ Version` by SemVer 2.0.0 precedence, with `Check`, or with `Window.Check` on a window `WindowFromAnnotations` reads. A reader that sees several windows takes their `Intersect`.
- **Same answers everywhere.** An implementation in another language, such as a Helm template, the UI or a CEL rule, runs the cases of `OrderVectors`, so it decides exactly as `Check`, `IsDevelopment` and `AcknowledgementHolds` do.

## The files here

| File | Kind | Holds |
|---|---|---|
| `*.go` | hand-edited | the library extensions import |
| `zz_generated.version.go` | generated | `Version`, `MinVersion`, `FingerprintScheme`, `Fingerprints` and `UpgradableFrom` |
| `VERSION`, `MIN_VERSION` | generated | `Version` and `MinVersion`, for tools that read them from a git ref without building Go |
| `testdata/order.json` | hand-edited | the cases of `OrderVectors` |
| `CHANGES` | hand-edited | the contract changes no artifact shows |
| `releases/<version>.yaml` | frozen | the record of each core release |
| `MAINTAINING.md` | hand-edited | how the version moves, and how releases are recorded |

`make gen-contract` writes the generated files: never edit them. A record is added at each release and never changes after. How core works out the version, what the records hold and when a pull request needs a decision label: [MAINTAINING.md](MAINTAINING.md).
