# PITR Golden E2E — Immediate Plan

Status: **ready to implement (after current PITR changes are committed)**.
Last updated: 2026-08-06.

Scope: the minimal, high-value test set to verify PITR / restore-from-PITR right
now. Broader test-architecture ideas live in
[test-architecture-roadmap.md](./test-architecture-roadmap.md).

---

## 0. Prerequisites / logistics

- Land the current PITR feature changes first (commit), then implement this on a
  **separate branch** to avoid bloating the feature PR.
- Reference provider: **PXC** (richest `uiSchema` + `pitrParametersSchema`).
- Runs in the existing live lane (`dev-fe-e2e.yaml`: minikube + real Go backend +
  provider via OLM catalog).

---

## 1. What we build now (agreed minimal set)

1. **Golden PITR e2e** (date + latest) on PXC, wired into the live e2e lane.
2. **Fixtures-vs-OpenAPI contract guard** — compile-time first (type the mock
   fixtures with generated OpenAPI types). See roadmap item 3(a).

Deferred to later branches / discussion:

- Restore-to-**new-cluster** (PITR seed) → separate branch.
- BE→FE gate, nightly, validator container → roadmap.

---

## 2. The golden scenario (PXC, one connected flow)

PITR restores roll **forward from a full base backup**, so a base backup is
required as the recovery anchor.

1. **Create cluster via wizard.** Basic → resources → backups step: add a
   schedule, enable **PITR** on the storage (PXC: pick the PITR storage
   location) → finish wizard → wait for **Up**.
2. **Verify PITR enabled** in the cluster object: `storages[].pitr.enabled = true`.
3. **Create a base backup (demand / "now").** This is the recovery anchor.
   (Demand is preferred over waiting for the schedule cron: deterministic +
   faster. A demand backup and a scheduled backup are functionally identical as
   a PITR anchor.)
4. **Insert data → record timestamp T → insert more data.**
5. **Wait for logs to upload** (binlog/oplog window must cover T), then
   **delete the data**.
6. **Restore to date (T):** open restore → "From PITR" → `recoveryTarget=date` →
   pick T within the window → run → `Restoring` → `Up` → restore `Succeeded`.
7. **Verify data == state at T.**
8. **Restore to latest:** repeat with `recoveryTarget=latest` (no date) → green →
   **verify data == latest state**.

### Optional realism assert (not a second restore)

In the **setup** step, after enabling PITR on storage A, optionally assert that
enabling PITR on storage B is **disabled/hidden** (real
`maxPITREnabledStorages=1` flag from the live BackupClass). One cheap assertion,
no second full restore. The full limit logic is primarily covered by unit tests.

---

## 3. Unit vs e2e split for PITR

| Behavior                                                                                            | Layer                                       |
| --------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| `supportsPITR:false` → PITR block hidden                                                            | unit (mock)                                 |
| `maxPITREnabledStorages:1` → second storage PITR disabled + reason                                  | unit (mock); optional 1 assert in e2e setup |
| No `pitrParametersSchema` → no config gear; with schema → gear present, disabled when off           | unit (mock)                                 |
| PG auto-enable PITR with schedules                                                                  | unit (mock)                                 |
| Restore schema: `date` requires date; `latest` forbids date                                         | unit (mock)                                 |
| Window range validation + picker bounds                                                             | unit (mock)                                 |
| `resolveRestoreAction` branches (navigate-new / restore / none)                                     | unit (mock)                                 |
| `toRestoreDateISO` UTC-offset serialization                                                         | unit                                        |
| Restore modal: pick "From PITR", persist selected time, window from `status.backup.storages[].pitr` | e2e `pr` (mocked routes)                    |
| Full data-integrity restore (date + latest)                                                         | **e2e `@release` live, PXC**                |

Principle: engine/flag variations → unit; integration confidence → one golden
live path.

---

## 4. Comparison with v1 PITR tests

v1 covered PITR **only via e2e** (no unit tests). Main files:
`ui/apps/everest/.e2e/release/pitr.e2e.ts`,
`.e2e/pr/db-cluster-details/edit-db-cluster/pitr.e2e.ts`,
`.e2e/pr/db-restore/db-restore-action/db-cluster-restore-action.e2e.ts`.

**Port (keep):**

- base backup → insert data → restore to **date** → verify.
- restore to **latest** → verify.
- per-engine storage difference (PXC picks a storage; Mongo does not) — but we
  run **one reference provider (PXC)** in the golden, not the full matrix.

**Drop / change for v2:**

- v1 read the PITR window from a dedicated `/pitr` endpoint returning
  `{ earliestDate, latestDate, gaps, latestBackupName }`. **v2 has no `/pitr`
  endpoint** — the window comes from `status.backup.storages[].pitr`.
- `gaps` and `latestBackupName` are **removed** in v2 → drop gap-alert and
  latest-backup-name gating cases.
- v1 test ids differ from v2 → re-target selectors.
- v1 ran `3 engines × sizes` → collapse to **one reference provider**.

**Deferred:** v1's "PITR restore to new cluster" (seed) → separate branch.

---

## 5. Wiring

- Re-enable / rewrite the `db-restore` project in
  `ui/apps/everest/.e2e/playwright.config.ts` (currently commented out on the v2
  branch) and add the golden spec there.
- Tag as `@release` so it runs in the live lane; keep it to the two sub-flows
  (date + latest) in one connected test to minimise runtime.

---

## 6. Open questions (verify with BE)

- Is a **schedule** strictly required for the PITR stream to start, or is
  `pitr.enabled` on the storage sufficient with only a demand backup? (We keep
  the schedule in setup for realism regardless.)
- Confirm the exact v2 test ids / selectors for the restore modal PITR controls.
- Confirm `recoveryTarget=latest` requires the storage `pitr.state=Available`
  with `latestRestorableTime` set before the option is offered.
