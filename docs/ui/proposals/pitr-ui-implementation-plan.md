# V2 PITR — UI Implementation Plan

> **Scope:** Frontend implementation of Point-in-Time Recovery (PITR) for the V2 backup
> architecture described in
> [backups-restore-architecture.md](backups-restore-architecture.md) and
> [backups-ui-implementation-plan.md](backups-ui-implementation-plan.md).
>
> This document is the **PITR-specific** slice, delivered in independently reviewable phases.

## Table of Contents

- [Guiding Principles](#guiding-principles)
- [Backend Readiness](#backend-readiness)
- [Provider Differences (V1 vs V2)](#provider-differences-v1-vs-v2)
- [UI Surfaces](#ui-surfaces)
- [Phases](#phases)
  - [Phase 1 — Foundation](#phase-1--foundation-non-ui)
  - [Phase 2 — Storages tab + StorageRow](#phase-2--storages-tab--storagerow-display-only-minimal)
  - [Phase 3 — Config-PITR toggle](#phase-3--config-pitr-toggle-on-storagerow)
  - [Phase 4 — PITRConfigModal](#phase-4--pitrconfigmodal)
  - [Phase 5 — Restore-PITR](#phase-5--restore-pitr)
  - [Phase 6 — Wizard PITR (deferred)](#phase-6--wizard-pitr-deferred)
  - [Phase 7 — Cleanup](#phase-7--cleanup)
- [Dependency Graph](#dependency-graph)
- [Scope Boundaries](#scope-boundaries)
- [Decisions Log](#decisions-log)

---

## Guiding Principles

- **Provider-agnostic.** Read behavior from `BackupClass` (`supportsPITR`,
  `limits.maxPITREnabledStorages`, `pitrParametersSchema`, `uiSchema.pitr`) — **never**
  branch on `dbType`. PSMDB is the reference provider, but the code must work for
  PostgreSQL (multi-PITR) and PXC without rework.
- **Per-storage PITR.** V2 moves PITR from instance-level (V1) to per-storage
  (`Instance.spec.backup.storages[N].pitr`).
- **No legacy reuse.** The V1 `DatabaseClusterPitr` schema, `getPitrFn`, and
  `useDbClusterPitr` (the `/database-clusters/{name}/pitr` endpoint) are dead V1 code.
  Do **not** uncomment or build on them — the restore window is derived from instance status.
- **Independently reviewable phases.** Each phase is a self-contained, verifiable slice.

---

## Backend Readiness

Backend PITR infrastructure is complete — **no blocker**.

| Capability                   | Source                                                                                                                   | Status |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------ | ------ |
| Per-storage PITR config      | `InstanceBackupStoragePITR{Enabled, Parameters}` — `api/core/v1alpha1/instance_types.go:206`                             | ✅     |
| Restore PITR target          | `DataSourcePITR{Type: date\|latest, Date}` — `api/backup/v1alpha1/restore_types.go:58`                                   | ✅     |
| Provider flags/limits        | `SupportsPITR`, `Limits.MaxPITREnabledStorages`, `PITRParametersSchema` — `api/backup/v1alpha1/backupclass_types.go:130` | ✅     |
| Server-side validation       | `provider-runtime/controller/backup_validation.go` (`ValidateRestorePITR`, limit enforcement)                            | ✅     |
| Restore window (upper bound) | `Instance.status.backup.storages[].latestRestorableTime` — `api/core/v1alpha1/instance_types.go:444`                     | ✅     |

**FE generated types already contain all PITR fields** (`ui/api/crds.gen.types.ts`):
`providerManaged.supportsPITR` (L430), `limits.maxPITREnabledStorages` (L399),
`pitrParametersSchema` (L419), `storages[].pitr.{enabled,parameters}` (L833),
`dataSource.backup.pitr` (L1627), `status.backup.storages[].latestRestorableTime` (L1723).
**No type regeneration required.**

**Not available (out of scope for first iteration):** `earliestRestorableTime`, `gaps`,
`latestBackupName` on status. The restore date-picker derives its window as:

- `minDate` = the selected backup's finish time (PITR restores forward from the base backup)
- `maxDate` = `latestRestorableTime` of the backup's storage

No separate `/pitr` availability endpoint is needed.

---

## Provider Differences (V1 vs V2)

V1 hard-coded PITR behavior per `DbType`. V2 expresses the same differences declaratively
via `BackupClass`.

### V1 behavior (main), verified from source

Common: PITR requires ≥1 backup schedule; toggle disabled otherwise.

| Provider            | Enabling PITR                                                       | PITR storage                                                                                          | Notes                                                           |
| ------------------- | ------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| **MongoDB (PSMDB)** | explicit `backup.pitr.enabled`                                      | **auto** = first schedule's storage (`schedules[0]`); "first storage will be used"; no storage select | —                                                               |
| **MySQL (PXC)**     | explicit toggle                                                     | **user-selectable** separate storage (`AutoCompleteAutoFill`)                                         | only provider with manual storage pick                          |
| **PostgreSQL (PG)** | **auto**: enabled when ≥1 schedule; user can't toggle independently | auto, per-schedule                                                                                    | restore has a PG-specific "stuck in Restoring" limitation alert |

Source: `pages/db-cluster-details/cluster-overview/old-cards/pitr-details/edit-pitr.tsx`,
`pitr-storage.tsx`, `pages/common/pitr.messages.ts`, `modals/restore-db-modal/modal-content.tsx`.

### V2 declarative mapping

| V1 hard-coded rule                     | V2 declarative source                                                                                                                                          |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| PSMDB/PXC single PITR stream; PG multi | `limits.maxPITREnabledStorages` (1 = PSMDB/PXC, N = PG)                                                                                                        |
| Whether PITR is offered at all         | `supportsPITR: bool`                                                                                                                                           |
| PSMDB auto storage / PXC manual        | In V2 PITR is a toggle **on the storage row** → storage choice is implicit (toggle the row you want). Future `pitrStorageMode: auto\|manual` (Future Ideas #8) |
| oplogSpan / compression fields         | `pitrParametersSchema` → rendered by `<UIGenerator sectionKey="pitr">`                                                                                         |
| PITR requires a schedule               | Q5 Option A: toggle disabled when no active schedules + alert                                                                                                  |

---

## UI Surfaces

1. **Instance Details → Backups → Storages sub-tab → `<StorageRow>`** — PITR toggle + ⚙ gear.
2. **`<PITRConfigModal>`** — provider-specific config via `<UIGenerator sectionKey="pitr">`.
3. **Restore modal** — radio (backup point / point in time) + date picker.
4. **Creation wizard → Backups step → PITR block** — deferred (Phase 6).

---

## Phases

### Phase 1 — Foundation (non-UI)

Independent; can run in parallel with Phase 2.

**Steps**

1. `pitr-availability.utils.ts` — compute the restore window: `minDate` = backup `finishedAt`,
   `maxDate` = `latestRestorableTime` from `instance.status.backup.storages[]` matched by the
   backup's storage.
2. Helpers to read/write `instance.spec.backup.storages[i].pitr` (`enabled` / `parameters`).
3. `useCreateRestoreFromPointInTime` mutation — payload
   `spec.dataSource.backup.pitr = { type: 'date', date }`.

**Files**

- New: `pages/db-cluster-details/backups/pitr/pitr-availability.utils.ts`
- `hooks/api/restores/useDbClusterRestore.ts` — implement `useCreateRestoreFromPointInTime`
  (currently commented at L61–86)

**Verification:** unit tests for the availability util (edge cases: missing
`latestRestorableTime`, non-`Succeeded` backup).

---

### Phase 2 — Storages tab + StorageRow (display-only, minimal)

Foundational UI phase. **Start here.** No PITR logic yet — display only.

**Steps**

1. In `backups.tsx`, add horizontal MUI Tabs **[Storages] [Schedules]** above `<BackupsList>`
   (backups table always visible below).
2. `storages-list/storages-list.tsx` — maps `instance.spec.backup.storages[]`.
3. `storages-list/storage-row.tsx` — horizontal bar: storage name + **Default** chip
   (first / `main: true`). No schedule counter, no delete (deferred to general backups plan).
4. Empty state for the Storages tab: **"No storages configured"** + hint (a storage appears
   automatically when a schedule or backup is created). This is a **new** component, distinct
   from the existing `NoStoragesMessage` (which is about namespace-level BackupStorages).
5. Schedules tab reuses the existing `flattenSchedules` helper for display.

**Files**

- `backups/backups.tsx` — add tab state + panels
- New: `backups/storages-list/storages-list.tsx`
- New: `backups/storages-list/storage-row.tsx` (+ `.types.ts`, `.messages.ts`)
- New: `backups/storages-list/no-instance-storages-message.tsx`

**Verification:** render tests (multiple storages, Default chip on first, empty state); tabs
switch; BackupsList stays visible below.

---

### Phase 3 — Config-PITR toggle on StorageRow

Depends on Phase 1 + Phase 2.

> **Design principle (locked from Phase 2 review):** keep `StorageRow` presentational — it renders
> the name plus action slots only. PITR behaviour (read `pitr.enabled`, gating from `BackupClass`
> limits, RBAC) lives in a hook (e.g. `useStoragePitr(storage)`), provider-agnostic (reads the
> `BackupClass`, never `dbType`). Orchestration (PATCH Instance) stays in `StoragesList` / the
> container. No business logic in the render component.

**Steps**

1. PITR toggle ON/OFF reading `storages[i].pitr.enabled`.
2. OFF → ON: if `pitrParametersSchema` present → open `<PITRConfigModal>` (Phase 4); else direct
   PATCH. Cancel → revert to OFF.
3. ON → OFF: confirmation → PATCH `pitr.enabled: false`.
4. Gating: disabled at `limits.maxPITREnabledStorages` (tooltip); disabled with no active
   schedules + alert (Q5 Option A).
5. ⚙ gear visible when `pitr.enabled` + schema present.

**Files**

- `backups/storages-list/storage-row.tsx`
- `hooks/api/db-instances/useUpdateDbInstance.ts` (`useUpdateDbInstanceWithConflictRetry`,
  PATCH `storages[]` pattern as in `scheduled-backup-modal.tsx:81`)

**Verification:** toggle-state tests, limit tooltip, disabled-without-schedules.

---

### Phase 4 — `<PITRConfigModal>`

Depends on Phase 3.

**Steps**

1. Modal "Configure PITR — {storageName}" with `<UIGenerator sectionKey="pitr" />`.
2. Extend `useBackupClassUiSchema` (`hooks/api/backup-classes/useBackupClasses.ts:67`) to expose
   the `pitr` section.
3. Save → PATCH `storages[i].pitr.parameters` (pack values as in on-demand modal via
   `removeEmptyFieldValues`).
4. Graceful degradation: no schema → no modal (plain on/off from Phase 3).

**Files**

- New: `backups/pitr-config-modal/*`
- Reference: `on-demand-backup-modal.tsx:67`

**Verification:** render fields from mock `uiSchema.pitr`, submit packs `parameters`, degradation.

---

### Phase 5 — Restore-PITR

> **Status: PAUSED** — depends on BE PR
> [#2668 "Rework PITR recovery window and point-in-time data sources"](https://github.com/openeverest/openeverest/pull/2668)
> which reworks the restore CRD and status. That PR is DRAFT and needs companion provider
> PRs (PSMDB/PXC/PG) before it can merge, after which `ui/api/*.gen.ts` types land on
> `release-2.0`. **Do not build against the old `dataSource.backup.pitr` shape** — it is
> superseded by the contract below.

**Confirmed contract (from #2668, FE types already regenerated in the PR)**

Request `dataSource` — the same shape for `Restore.spec.dataSource` and `Instance.spec.dataSource`:

```ts
dataSource: {
  type: 'Backup' | 'PointInTime';
  backup?: { backupRef: { name: string } };          // required when type=Backup
  pointInTime?: {                                     // required when type=PointInTime
    recoveryTarget: 'date' | 'latest';
    date?: string;                                    // required when 'date', FORBIDDEN when 'latest'
    source: {
      instanceRef?: { name: string };                // omit for in-place (defaults to spec.instanceRef)
      storageRef: { name: string };                  // ALWAYS required; storage must have pitr.enabled=true
    };
  };
}
```

Status window — `Instance.status.backup.storages[].pitr`:

```ts
{ earliestRestorableTime?: string; latestRestorableTime?: string;
  state?: 'Available' | 'Unavailable'; reason?: string; message?: string }
```

The window is **authoritative, contiguous and conservative** — every point between
earliest and latest is restorable (providers truncate forward at any discontinuity). ⇒
**no gap logic on FE**; picker bounds = `[earliestRestorableTime, latestRestorableTime]`.
`state` is **binary** (`Available`/`Unavailable`) — no `Degraded`.

**Decisions (2026-07-30)**

- **Same-namespace only.** Refs are `{name}` (no namespace); V1 posts restores to
  `namespaces/{ns}/…`. No namespace selector; restore-to-new-DB stays same-ns.
- **No cross-instance in-place.** In-place restore uses the instance's own history
  (omit `source.instanceRef`). "Use another instance's data" happens only when creating a
  **new** instance (clone/seed), exactly like V1.
- **`recoveryTarget: 'latest'`** = one-click, send **no** `date`. `'date'` = picker over the window.
- **Storage selector** only when 2+ PITR-enabled storages; still send `storageRef` even for 1.
- **Restore `parameters`** (schema-driven, `BackupClass.spec.restoreParametersSchema`):
  pending Diogo — likely reuse source/default params, no re-prompt. Out of scope until confirmed.

**Steps (when unblocked)**

1. Radio "Restore to backup" / "Restore to point in time" in the restore modal.
2. `getPitrWindow(storageStatus)` pure helper → `{ available, min?, max?, message? }` from
   `status…storages[].pitr`; combine with `spec…storages[].pitr.enabled` for the storage list.
3. `DateTimePickerInput` bounded by the window; FE range-check (BE admission does **not**
   range-check the date — double validation stands).
4. `toRestoreDateISO(date)` pure util = `new Date(x).toISOString().split('.')[0] + 'Z'`
   (ported from V1 `restore-db-modal.tsx`; already offset-safe) **+ unit test** (offset
   correctness; no `date` emitted for `latest`).
5. `buildRestoreDataSource(...)` → the discriminated union above; `useCreateRestore`.
6. Gate PITR on `state === 'Available'`; surface BE `message` when `Unavailable`.

**Files**

- `modals/restore-db-modal/restore-db-modal.tsx` / `modal-content.tsx` / `restore-db-modal-schema.ts`
- `hooks/api/restores/*` (`useCreateRestore`)

**Verification:** radio switch; picker bounds from status window; payload
`dataSource.pointInTime.{source.storageRef, recoveryTarget, date}`; `latest` omits date;
`toRestoreDateISO` unit test green.

---

### Phase 6 — Wizard PITR

**Design (locked):** per-storage model from the start (covers single AND multi). The Backups
step gets a PITR block below the schedules: a **per-storage list of toggles** (storages are the
distinct `storageName`s used by `backup.schedules`), each with a toggle + configure (⚙) — the
same UI concept as the Details Storages panel. Single vs multi is only the enable limit
(`maxPITREnabledStorages`), not a different UI.

- Providers never auto-enable (locked): the user controls PITR per storage (declarative). No
  `dbType` special-casing; drive purely off `supportsPITR` + `maxPITREnabledStorages` +
  `uiSchema.pitr`. (If a provider ever wants auto-on, that's a future `pitrStorageMode` flag.)
- The **custom schema (`uiSchema.pitr`) participates only inside the config modal** (per-storage
  parameters via `<UIGenerator sectionKey="pitr">`, reusing Phase 4's `PitrConfigModal`). The
  toggles/list are plain React driven by the class flags. No provider schema → no ⚙, on/off only.

**Form shape:** `backup.pitr: { [storageName]: { enabled: boolean; parameters?: {...} } }`.
`buildBackupSpecFromWizard` attaches each entry to the matching `storages[i].pitr`.

**Slices:** 0. Extract pure gating (`getPitrBlockReason(storages, storageEnabled, max)` → reason code) into
`pitr.utils.ts`; refactor `useStoragePitr` to use it (shared by Details + Wizard). Details green.

1. Form: add `backup.pitr` to schema (`database-form-schema.ts`) + defaults + edit-mode prefill.
2. Wizard PITR list component (per-storage toggles, gating from class, reuses `PitrConfigModal`).
3. `buildBackupSpecFromWizard` attaches `pitr`; update preview; wire the list into `BackupStep`.
4. Tests (single limit gating, multi, no-schema on/off, build output, edit prefill).

**Multi-PITR (PG):** built now (PG provider in development) — same list, limit > 1 allows several
enabled. Reference: `on-demand-backup-modal.tsx`, Phase 4 `PitrConfigModal`.

---

### Phase 7 — Cleanup

Remove legacy: `getPitrFn` (`api/backups.ts:100`), `useDbClusterPitr` (`useBackups.ts:161`),
`DatabaseClusterPitr(Payload)` (`shared-types/backups.types.ts:72`), the `DatabaseClusterPitr`
schema in `api/openapi/http-api.yaml:2645`, and the V1
`pages/db-cluster-details/cluster-overview/old-cards/pitr-details/` directory.

---

## Dependency Graph

```
Phase 1 (Foundation) ─────────────► Phase 3 (PITR toggle) ──► Phase 4 (PITRConfigModal)
        │                                 ▲                              │
        │                                 │                              │
        └──────────► Phase 5 (Restore)    │                              ▼
                                          │                          Phase 7 (Cleanup)
Phase 2 (Storages tab) ───────────────────┘                              ▲
                                                                         │
Phase 5 (Restore) ───────────────────────────────────────────────────────┘

Phase 6 (Wizard PITR) — deferred to general backups wizard step
```

Phase 1 and Phase 2 start in parallel. Phase 5 is not blocked by the Storages tab.

---

## Scope Boundaries

- **Storages tab is minimal / PITR-focused:** tabs + StorageRow (name + Default chip) + PITR
  toggle/gear. **No** schedule counter, **no** remove dialog (those belong to the general
  backups plan).
- Empty state: "No storages configured" + hint.
- Wizard PITR and the `gaps` warning are out of the first iteration.

---

## Decisions Log

- **Q5 (PITR–schedule dependency):** Option A — disable the PITR toggle when there are no
  active schedules; show an alert.
- **Providers:** base on PSMDB, but read behavior from `BackupClass` (provider-agnostic).
- **Restore window:** derived from `latestRestorableTime` + the selected backup's finish time;
  no dedicated availability endpoint.
- **`DatabaseClusterPitr`:** dead V1 legacy — do not build on it; remove in Phase 7.
