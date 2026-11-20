# Monitoring E2E — Plan and Coverage

Status: **golden implemented** (`release:monitoring`).
Last updated: 2026-11-13.

Scope: the live, release-lane coverage of monitoring (MonitoringClass /
MonitoringDestination / MonitoringBinding) with the PMM class, and where the
rest of the monitoring behaviour is tested. Mirrors the approach of the
[PITR golden plan](./pitr-golden-e2e-plan.md): one connected live path for
integration confidence, everything else in unit tests.

---

## 0. Environment

The release lane (`.github/workflows/release-e2e.yml`, kind) installs, in
addition to core Everest, the PXC provider and SeaweedFS:

- **The PMM monitoring class** — `make deploy-monitoring-pmm` (root
  `Makefile`): builds `openeverest/monitoring-pmm` (workflow input
  `monitoring_pmm_ref`, default `main`), loads the image into the cluster and
  Helm-installs its chart into `openeverest-system`. The class `pmm` must be
  `Accepted` for the PXC provider (`ProviderManaged`, `RollingRestart`).
- **A PMM 3 server** — `.e2e` `make deploy-monitoring` (chart `percona/pmm`
  1.10.0 = PMM 3.10.0; the class supports PMM 3 only). Service
  `monitoring-service` in `default`, self-signed certificate.

The suite needs no CI secrets: it creates PMM service-account tokens through
the PMM API from inside the `pmm-0` pod (`kubectl exec` + `curl`), see
`.e2e/utils/monitoring.ts`.

Local run (k3d `everest-server-test`):

```sh
make deploy-monitoring-pmm MONITORING_PMM_DIR=<path to monitoring-pmm>
make -C ui/apps/everest/.e2e deploy-seaweedfs create-db-namespaces deploy-monitoring
make -C ui/apps/everest/.e2e test PROJECT=release:monitoring
```

---

## 1. The golden scenario (`release/monitoring.e2e.ts`)

One serial flow in namespace `everest`, a 1-node PXC instance (one proxy)
sized for a CI runner.

| #   | Step                                                                                                                      | What it proves live                                                                    |
| --- | ------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| 1   | Settings → Add destination: PMM, URL, new service-account token, skip TLS verification → `Ready · v3…`                    | class form, credentials Secret creation, class controller probe of a real PMM 3 server |
| 2   | Wizard: select the destination on the Monitoring step → instance `Ready`                                                  | wizard step, `spec.monitoring` entry accepted by core and the provider                 |
| 3   | Overview card: entry `Ready`, mode `Provider-managed`; PMM inventory lists the instance's MySQL service                   | binding status round-trip, data reaches PMM                                            |
| 4   | Settings: Used by `1 instance`; Delete → "Destination in use"; editing the URL warns that the instance restarts           | `status.attachedInstances`, in-use guard, `RollingRestart` from the live class         |
| 5   | An entry whose destination no longer exists (written through the API) is reported, and Edit monitoring removes it (#3597) | orphan entries: overview reason, picker "No longer available", removal on save         |
| 6   | Edit monitoring: deselect → restart and single-replica notice → Save → `Not configured`                                   | disruption notice from the live class and topology, detaching                          |
| 7   | Delete the unused destination; its credentials Secret is kept                                                             | delete after detach, Secret survives (#3596)                                           |
| 8   | Add a destination on the kept Secret ("Use an existing Secret") → `Ready`                                                 | Secret labels let it be found and reused                                               |
| 9   | Edit it → "Enter new credentials" with a new token → the same Secret gets a new `resourceVersion`                         | credentials rotation in place                                                          |

Set-up and teardown delete, through the API and in that order, the instance,
the destinations and the Secret (best effort), so a failed run leaves nothing
behind and a re-run starts clean.

---

## 2. Effect on the other release goldens

- With a class `Accepted`, the PXC wizard has a **Monitoring** step (Basic
  information → Backups → **Monitoring** → Database version → …). `pitr` and
  `restore-new-cluster` pass it with `passMonitoringStep()` (a no-op when the
  step is absent).
- `openeverest-system` (the class namespace) is listed in the namespace pickers,
  so `populateBasicInformation()` matches the namespace option exactly
  (`everest` would also match `openeverest-system`).

---

## 3. Unit vs e2e split

| Behaviour                                                                                 | Layer                                                                                       |
| ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| Destination form: validation, credentials modes (new / existing / keep / update), CA pair | unit: `monitoring-destination-modal.test.tsx`, `monitoring-destination-modal.utils.test.ts` |
| Save order Secret → destination, rollback, 409 "changed meanwhile"                        | unit: `monitoring-destination-requests.test.ts`                                             |
| Restart notice conditions (edit, dirty, `RollingRestart`, attached)                       | unit: `restart-notice.test.tsx`                                                             |
| Picker: limit, one destination per system, deleting, unavailable entries, toggling        | unit: `monitoring-destinations-picker.utils.test.ts`                                        |
| Wizard step shown only with a compatible class                                            | unit: `monitoring-step.test.tsx`                                                            |
| Overview card reasons and statuses                                                        | unit: `monitoring-card.test.tsx`, `monitoring-card.utils.test.ts`                           |
| Edit monitoring disruption and payload                                                    | unit: `monitoring-edit-modal.test.tsx`, `monitoring-edit-modal.utils.test.ts`               |
| Settings list, Used by, delete dialog                                                     | unit: `monitoring-destinations.test.tsx`                                                    |
| Rendering of the monitoring screens                                                       | e2e `pr:visual` (mocked routes)                                                             |
| Class acceptance, probe, binding, data in PMM, in-use guard, Secret reuse and rotation    | **e2e `@release` live, PXC + PMM**                                                          |

---

## 4. Not covered yet

- Restore to a new instance carrying the source's monitoring.
- RBAC: read-only destinations, Secrets without `use` (#3587) — `release/rbac`.
- Other classes (Prometheus, VictoriaMetrics, Coroot): need their operators and
  released class charts in the lane.
- Negative live paths (unreachable server, revoked token → `Not ready` with a
  reason); covered by unit tests on the UI side.
- That the class probes with a rotated token: it does not watch Secrets and
  re-probes a `Ready` destination every 5 minutes.
- PSMDB and PostgreSQL providers: not deployed in the release lane.
- `dev-fe-e2e.yaml` still installs an unused PMM 2 server.
