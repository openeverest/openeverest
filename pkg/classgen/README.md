# classgen

classgen generates what an integration's class manifests take from OpenEverest core: the OpenAPI v3 schemas of class parameters, rendered from Go types (see [doc.go](doc.go)), and the **stamp**, the contract version of the core the integration is built against, which tells OpenEverest and its installers the oldest OpenEverest the build works with. It is meant for monitoring integrations, and for any other extension that ships a class (a MonitoringClass, a BackupStorageClass) from its own Helm chart rather than through provider-sdk, which stamps providers itself.

## Run it with `go tool`

Require classgen as a tool of the module that builds your integration's controller:

```sh
go get -tool github.com/openeverest/openeverest/v2/pkg/classgen/cmd/classgen@<version>
```

That adds a `tool` line to `go.mod`, and `go tool classgen` then builds classgen against the core your module requires: the version you pass is the core your controller links, and Dependabot bumps both at once. What classgen writes depends on that core: the stamp is its contract version, and the schemas are rendered by its code. So every form of the command first checks that the core it links is the one your module requires, and refuses to write anything when it is not. Don't run a separately installed binary or `go run github.com/openeverest/openeverest/v2/pkg/classgen/cmd/classgen@<version>`: classgen refuses them as soon as the version they name drifts from `go.mod`, and `go run` builds classgen with core's own dependencies rather than your module's. When `go.mod` replaces the core with a local directory, or a `go.work` workspace includes it, classgen warns and stamps the contract version of that tree.

Stamp from the module that builds the controller: the chart must carry the contract version the controller links, which is the version the controller's startup gate checks against the OpenEverest in the cluster. A separate contract module that providers import, such as `github.com/openeverest/monitoring-pmm/api`, can run classgen for its schemas too, with classgen as a tool of that module. A tool requirement is part of the module graph, so the core it requires becomes a minimum for every provider that imports the module: require the oldest released core that has the classgen you need.

## Schemas

```go
//go:generate -command classgen go tool classgen --package .
//go:generate classgen --type DestinationParameters --output zz_generated.destination-parameters.yaml
//go:generate classgen --type InstanceParameters --output zz_generated.instance-parameters.yaml
```

## The stamp

```sh
go tool classgen stamp --chart-dir charts/<name>
```

writes the annotation `openeverest.io/contract-version` into the chart's `Chart.yaml`, leaving the rest of the file as it is, and the helper template `templates/_openeverest_contract.tpl`. Never write either by hand: they are generated from the core you link. Include the helper in `metadata.annotations` of the MonitoringClass, and of any other object of the chart that OpenEverest interprets (a BackupStorageClass, BackupClass, Provider, ClusterInstancePreset or Plugin):

```yaml
metadata:
  name: {{ .Values.class.name }}
  annotations:
    {{- include "openeverest.contract.annotations" . | nindent 4 }}
```

The helper renders the stamp from `Chart.yaml`, so every object carries the value of the chart, and an installer can read it before rendering anything. When the value `compatibility.acknowledge.coreVersion` is set, it also renders `openeverest.io/compatibility-acknowledged`: a user sets it to an OpenEverest release to run this build on that release although it does not support it, and the startup gate and OpenEverest read it on the object. Document it in your `values.yaml`:

```yaml
compatibility:
  acknowledge:
    # OpenEverest release on which to run this build although it does not
    # support it, such as 2.0.0. Empty: do not.
    coreVersion: ""
```

After stamping, classgen prints the oldest OpenEverest the build works with:

```
Stamped: charts/monitoring-x/Chart.yaml (openeverest.io/contract-version=2.0.0-alpha.2, was 2.0.0-alpha.1)
This build works with OpenEverest 2.0.0-alpha.2 and later.
```

A build against a commit of OpenEverest's `main` branch whose contract has changed since the last release carries a development value, such as `2.0.0-alpha.1.next`, which no release has. classgen then warns that the build cannot be published to the Hub until that OpenEverest is released. While `main`'s contract is still that of a release, the stamp names that release, so an integration can take a fix from `main` without raising its minimum OpenEverest.

## Keeping the stamp right

```sh
go tool classgen verify --chart-dir charts/<name> [--release]
```

renders the chart with its default values, without a cluster, and fails unless `Chart.yaml` and every object OpenEverest interprets that the chart's own templates render carry the stamp of the core you link, the helper is the one this classgen writes, and the acknowledgement reaches every stamped object. With `--release`, it also fails on a development value and on a `go.mod` that replaces the core: a release tells users the oldest OpenEverest it works with, so that must be one they can install. `VerifyChart` runs the same checks from Go.

Wire it into the Makefile and CI:

- `make generate` runs `go tool classgen stamp`, after the schemas;
- `make verify` runs `make generate`, fails on a diff in `Chart.yaml`, `templates/_openeverest_contract.tpl` and the generated schemas, then runs `go tool classgen verify`. A core bump that was not regenerated, or a hand-edited stamp, fails CI;
- the release workflow runs `go tool classgen verify --release` before packaging the chart.
