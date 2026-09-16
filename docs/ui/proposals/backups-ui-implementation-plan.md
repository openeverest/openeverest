# V2 Backups / Restore / PITR — UI Implementation Plan

> **Scope:** Frontend implementation of the V2 backup architecture described in
> [backups-restore-architecture.md](backups-restore-architecture.md).
> Focused on incremental delivery starting with on-demand backups.
>
> **Branch:** `2029-v2-on-demand-backups` (PR #2119)
>
> **BE dependency:** PR #2226 (`backupclass-ui-schema`) — adds `limits`, `pitrConfigSchema`,
> and `uiSchema` to `BackupClassSpec.providerManaged`. Must be merged before UI can consume
> these fields at runtime.

---

## Current State Summary

| Area                        | Status                                                        |
| --------------------------- | ------------------------------------------------------------- |
| On-demand backup modal      | **Done** — v2 API, storage auto-register, backup class select |
| Backups list                | **Done** — v2 API, delete action, status mapping              |
| Backup/BackupClass hooks    | **Done** — v2 API (RBAC commented out)                        |
| Backup API functions        | **Done** — v2 endpoints; v1 legacy retained                   |
| Restores list               | **Done** — v2 list + delete                                   |
| Scheduled backup modal      | **Stub** — returns `null`                                     |
| Scheduled backups list      | **Stub** — returns `null`                                     |
| Restore modal (create)      | **Broken** — `@ts-nocheck`, missing hooks, v1 code            |
| Wizard backup step          | **Missing** — old step in `steps-old/`, no v2 step            |
| UIGenerator backup provider | **Missing** — no `backupClasses`/`backupStorages` registered  |
| PITR hooks/UI               | **Missing** — v1 PITR hook exists, v2 commented out           |
| Storage management (tab)    | **Missing** — `StorageRow`, PITR toggle, remove dialog        |

---

## Phase 1 — On-Demand Backup: UIGenerator + Polish

> **Goal:** Complete the on-demand backup flow with dynamic provider fields.
>
> **Depends on:** PR #2226 merged (BE: `uiSchema` on BackupClass).

### 1.1 UIGenerator integration in on-demand backup modal

- Read `backupClass.spec.uiSchema.backup` from the selected BackupClass
- Embed `<UIGenerator sectionKey="backup" />` below static fields (name, class, storage)
- On submit, pack UIGenerator values into `Backup.spec.config`
- Graceful degradation: no `uiSchema` → no dynamic fields

**Files:**

- `on-demand-backup-modal/on-demand-backup-fields-wrapper.tsx` — add UIGenerator
- `on-demand-backup-modal/on-demand-backup-modal.tsx` — merge dynamic values into `spec.config` on submit

### 1.2 On-demand modal: storage select shows unregistered storages

- Currently modal shows only storages from `instance.spec.backup.storages`
- Should also show **all namespace-level BackupStorages** not yet bound
- Group options: "Registered" / "Available" (or flat list with hint)
- Auto-register unregistered storage on backup create (already partly done)

**Files:**

- `on-demand-backup-fields-wrapper.tsx` — fetch namespace storages, merge with instance storages
- Hooks: may need `useBackupStoragesByNamespace` (v1) or v2 equivalent

### 1.3 "Create backup" dropdown: add "Schedule" option

- Replace plain "Create backup" button with `<MenuButton>` dropdown
- Options: "Now" (opens on-demand modal), "Schedule" (opens scheduled backup modal — wired in Phase 3)

**Files:**

- `backups-list/table-header/backups-list-table-header.tsx` — switch to dropdown button

---

#### 🏷️ Community Candidates (Phase 1)

| Task                                      | Why it's good for community                                                                                                     | Complexity |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| **1.3** — "Create backup" dropdown button | Isolated UI component, no business logic, clear spec (MUI `MenuButton` or `ButtonGroup`). Needs screenshot/mockup as reference. | Small      |

---

## Phase 2 — Backups Tab Layout: Storages + Schedules Tabs

> **Goal:** Add the two-tab layout (Storages / Schedules) to the Backups tab.
>
> **Can start in parallel with Phase 1.**

### 2.1 Backups tab: MUI Tabs (Storages / Schedules)

- Add horizontal MUI Tabs above the backups list
- Tab "Storages" — renders `<StoragesList />` (Phase 2.2)
- Tab "Schedules" — renders `<ScheduledBackupsList />` (Phase 3)
- Backups list table stays **below** tabs (always visible)

**Files:**

- `backups/backups.tsx` — add tab state, render tab panels
- New: `backups/storages-list/storages-list.tsx`

### 2.2 `<StorageRow />` component

- Horizontal bar component (~48–56px) showing: storage name, PITR toggle (disabled initially), schedule count, "Default" chip, delete button
- Data from `instance.spec.backup.storages[]`
- First storage → "Default" chip (filled)
- PITR toggle: read-only display for now (functional in Phase 5)
- Delete: opens `<StorageRemoveConfirmDialog>` (Phase 2.3)

**Files:**

- New: `backups/storages-list/storage-row.tsx`
- New: `backups/storages-list/storage-row.types.ts`

### 2.3 `<StorageRemoveConfirmDialog />`

- Confirmation dialog listing consequences: schedule count, PITR status, backup count
- Disabled when active backup (Running/Pending) targets this storage
- On confirm: PATCH Instance (remove storage from `storages[]`)

**Files:**

- New: `backups/storages-list/storage-remove-confirm-dialog.tsx`

---

#### 🏷️ Community Candidates (Phase 2)

| Task                                                | Why it's good for community                                                                                                                                                   | Complexity   |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| **2.2** — `<StorageRow />` presentational component | Pure presentational component with clear design spec (horizontal bar, name, badge, toggle, icons). Can be developed with Storybook/mock data. No API wiring needed initially. | Small–Medium |
| **2.3** — `<StorageRemoveConfirmDialog />`          | Standard confirmation dialog pattern (MUI Dialog). List of consequence items. Self-contained.                                                                                 | Small        |

---

## Phase 3 — Scheduled Backups

> **Goal:** Implement schedule CRUD on the Backups tab.
>
> **Depends on:** Phase 2.1 (tabs layout), Phase 1.1 (UIGenerator integration pattern — reuse for schedule config).
>
> **Open question:** [Q1] — does `InstanceBackupSchedule` need a `config` field? If yes, requires CRD change before this phase.

### 3.1 `<ScheduledBackupsList />`

- Flat table of ALL schedules across all storages
- Columns: Name, Storage, Schedule (human-readable cron), Retention, Status (Enabled/Disabled), Actions
- Row actions: Edit, Delete
- Data source: `instance.spec.backup.storages[].schedules[]` (flatten + attach storage name)

**Files:**

- `backups/scheduled-backups-list/scheduled-backups-list.tsx` (replace stub)
- `backups/scheduled-backups-list/scheduled-backups-list.constants.ts`
- `backups/scheduled-backups-list/scheduled-backups-list.messages.ts`

### 3.2 `<ScheduledBackupModal />`

- Create / Edit mode
- Fields: Name, Storage (select), Cron (time picker), Retention copies, Enabled toggle
- If [Q1] resolved with `config`: add UIGenerator `backup` section (same as on-demand)
- On submit: PATCH Instance → add/update schedule in `storages[].schedules[]`
- Auto-bind storage if not yet registered (same pattern as on-demand)

**Files:**

- `backups/scheduled-backup-modal/scheduled-backup-modal.tsx` (replace stub)
- `backups/scheduled-backup-modal/scheduled-backup-modal.types.ts`
- `backups/scheduled-backup-modal/scheduled-backup-modal-fields.tsx`
- Reuse: cron picker components from `steps-old/` or build new

### 3.3 Schedule delete

- Delete row action → confirmation → PATCH Instance (remove schedule from storage)
- If last schedule on a PITR-enabled storage → warn "PITR will stop working"

### 3.4 Wire "Schedule" option in dropdown

- Connect "Schedule" option from Phase 1.3 dropdown to open `<ScheduledBackupModal>` in create mode
- Wire edit action from `<ScheduledBackupsList>` row to open modal in edit mode

---

#### 🏷️ Community Candidates (Phase 3)

| Task                                       | Why it's good for community                                                                                                 | Complexity |
| ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------- | ---------- |
| **3.1** — `<ScheduledBackupsList />` table | Standard MUI DataGrid/Table. Clear column spec. Data transformation (flatten schedules across storages) is straightforward. | Medium     |
| **Cron picker component**                  | Reusable time/cron UI component. Could be extracted from `steps-old/` or built fresh. Well-defined input/output contract.   | Medium     |

---

## Phase 4 — Restore

> **Goal:** Restore from backup and PITR restore.
>
> **Can start in parallel with Phase 3** (no dependency).

### 4.1 Restore API + hooks

- Add `createRestoreFn` to `api/restores.ts` — `POST clusters/{cluster}/namespaces/{namespace}/restores`
- Implement `useCreateRestore` hook (replaces missing `useDbClusterRestoreFromBackup` / `useDbClusterRestoreFromPointInTime`)
- Single hook handling both backup restore and PITR restore (differentiated by `spec.dataSource.pitr` presence)

**Files:**

- `api/restores.ts` — add `createRestoreFn`
- `hooks/api/restores/useInstanceRestores.ts` — add `useCreateRestore`

### 4.2 Restore modal rewrite

- Full rewrite of `modals/restore-db-modal/` (currently `@ts-nocheck`, v1 code)
- Static fields: source backup info (name, storage, date, size — read-only)
- PITR section: radio (restore to backup point / restore to point in time) + date picker
- UIGenerator `restore` section (empty for PSMDB initially)
- Warning: "Restoring will replace all data. Cannot be undone."
- On submit: `POST Restore` with `dataSource.backupName` + optional `pitr` + `config`

**Files:**

- `modals/restore-db-modal/restore-db-modal.tsx` — full rewrite
- `modals/restore-db-modal/restore-db-modal.types.ts`
- `modals/restore-db-modal/restore-db-modal.messages.ts`

### 4.3 "Restore" action in backups list

- Add "Restore" row action to backup list menu (currently only "Delete")
- Opens restore modal with selected backup pre-filled
- Only enabled for `Succeeded` backups

**Files:**

- `backups-list/backups-list-menu-actions.tsx` — add Restore action

### 4.4 Restore to new cluster (wizard flow)

- Backup list row action: "Restore to New DB"
- Navigate to `/databases/new` with router state (`backupName`, `pointInTimeDate`)
- Wizard detects restore mode, pre-fills config, adds `spec.dataSource` on submit
- **Depends on Phase 6** (wizard backup step) — defer to Phase 6

---

#### 🏷️ Community Candidates (Phase 4)

| Task                                       | Why it's good for community                                                                                | Complexity |
| ------------------------------------------ | ---------------------------------------------------------------------------------------------------------- | ---------- |
| **4.3** — "Restore" action in backups list | Add menu item + open modal. Very isolated change to one file. Clear pattern from existing "Delete" action. | Small      |

---

## Phase 5 — PITR Management

> **Goal:** Per-storage PITR toggle and config modal.
>
> **Depends on:** Phase 2.2 (`StorageRow`), Phase 1.1 (UIGenerator pattern).
>
> **Can start in parallel with Phase 3/4** after Phase 2 is done.

### 5.1 PITR toggle on `<StorageRow />`

- Toggle ON → OFF: confirmation dialog → PATCH Instance (`pitr.enabled: false`)
- Toggle OFF → ON:
  - If provider has `uiSchema.pitr` (non-empty) → open `<PITRConfigModal>` (5.2)
  - If no pitr config schema → direct PATCH Instance (`pitr.enabled: true`)
  - If modal dismissed → toggle reverts to OFF
- Disabled with tooltip when `maxPITREnabledStorages` limit reached
- Disabled when no active schedules (with warning tooltip)

**Files:**

- `backups/storages-list/storage-row.tsx` — wire toggle logic
- Hook: `useBackupClassLimits()` or read from `backupClass.spec.providerManaged.limits`

### 5.2 `<PITRConfigModal />`

- Renders UIGenerator `pitr` section from `BackupClass.spec.uiSchema.pitr`
- Title: "Configure PITR — {storageName}"
- On save: PATCH Instance → update `storages[i].pitr.config`
- Gear icon on `StorageRow` opens this modal (visible only when PITR enabled + provider has config)

**Files:**

- New: `backups/pitr-config-modal/pitr-config-modal.tsx`
- New: `backups/pitr-config-modal/pitr-config-modal.types.ts`

### 5.3 v2 PITR data hook

- Uncomment/implement v2 `useDbClusterPitr` (currently commented out)
- Uses v2 API path for PITR availability check
- Needed for restore modal PITR date picker range

**Files:**

- `hooks/api/backups/useBackups.ts` — uncomment/fix v2 PITR hook

---

#### 🏷️ Community Candidates (Phase 5)

| Task                            | Why it's good for community                                                                                                                                   | Complexity   |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| **5.2** — `<PITRConfigModal />` | Self-contained modal. UIGenerator does the heavy lifting for field rendering. Modal shell is standard MUI Dialog + form pattern. Needs clear props interface. | Small–Medium |

---

## Phase 6 — Wizard Backup Step

> **Goal:** Add backup configuration step to the instance creation wizard.
>
> **Depends on:** Phase 3 (schedule modal patterns), Phase 5 (PITR patterns).
>
> **This is the most complex phase.** All previous patterns converge here.

### 6.1 `builtIn` mechanism in form engine

- Add `builtIn` key support to Provider uiSchema section processing
- `use-form-engine.ts`: if section has `builtIn: "backups"`, render `<WizardBackupsStep />` instead of UIGenerator
- Provider controls opt-in, ordering, labeling

**Files:**

- `components/ui-generator/` or `pages/database-form/` form engine
- `use-form-engine.ts` — add builtIn component mapping

### 6.2 `<WizardBackupsStep />`

- Enable Backups toggle
- BackupClass select (auto-select if single)
- Scheduled backups: list of `EditableItem` tiles, [+ Create backup schedule] button → `<ScheduleFormDialog>`
- PITR: toggle + storage select (single-PITR provider) or list (multi-PITR)
- No API mutations — reads/writes React Hook Form state
- On wizard submit: `buildSpecBackup()` assembles `spec.backup.storages[]` from selections

**Files:**

- New: `pages/database-form/steps/backup-step/`
- New: `pages/database-form/steps/backup-step/wizard-backups-step.tsx`
- New: `pages/database-form/steps/backup-step/build-spec-backup.ts`
- Shared sub-components from Phase 3/5

### 6.3 Restore to new cluster (wizard restore mode)

- Detect restore mode from router state
- Pre-fill wizard with source cluster config
- On submit: include `spec.dataSource.backupName` + optional `pitr`

**Files:**

- `pages/database-form/` — restore mode detection
- `hooks/api/db-instances/useCreateDbInstance.ts` — handle dataSource in spec

---

## Phase 7 — Cleanup & Hardening

> **Goal:** Remove legacy code, re-enable RBAC, finalize.
>
> **Can be done incrementally as each phase lands.**

### 7.1 Remove legacy v1 backup code

- Remove `legacyGetBackupsFn`, `legacyCreateBackupOnDemand`, `legacyDeleteBackupFn` from `api/backups.ts`
- Remove `useDbBackups`, `useCreateBackupOnDemand` (v1) from hooks
- Remove `steps-old/backups/` directory
- Remove `useCreateDbCluster.ts` (v1)
- Clean up `@ts-nocheck` from restore modal (after Phase 4 rewrite)

### 7.2 Re-enable RBAC

- Uncomment RBAC permission checks in all v2 backup/restore hooks
- Verify RBAC works with new backup/restore/backupclass resources
- Wire `canCreate`/`canDelete`/`canUpdate` to button disabled states

### 7.3 Remove `BackupStatus` type alias

- `backups.types.ts:57` — `TODO remove` — clean up

### 7.4 Verify cron/timezone handling

- `useDbInstanceList.ts:77` — `TODO during adding backups don't forget to check timezone and CRON converting`
- `useUpdateDbInstance.ts:38` — `TODO check cron converter during backups implementation`

---

#### 🏷️ Community Candidates (Phase 7)

| Task                                       | Why it's good for community                                                       | Complexity |
| ------------------------------------------ | --------------------------------------------------------------------------------- | ---------- |
| **7.1** — Remove legacy v1 backup code     | Straightforward deletion. Clear list of files/functions. Good first contribution. | Small      |
| **7.3** — Remove `BackupStatus` type alias | One-liner TODO cleanup.                                                           | Trivial    |

---

## Dependency Graph

```
PR #2226 (BE: uiSchema + limits)
    │
    ▼
Phase 1 ──────────────────────────┐
(On-demand + UIGenerator)         │
    │                             │
    ▼                             ▼
Phase 2 ◄──── can start    Phase 4
(Tabs + StorageRow)         (Restore)
    │                             │
    ├──────────┐                  │
    ▼          ▼                  │
Phase 3    Phase 5                │
(Schedules) (PITR)                │
    │          │                  │
    └────┬─────┘                  │
         ▼                        │
      Phase 6 ◄───────────────────┘
      (Wizard backup step)
         │
         ▼
      Phase 7
      (Cleanup)
```

### Parallelization opportunities

| Track A (main)             | Track B (parallel)              | Track C (community)                   |
| -------------------------- | ------------------------------- | ------------------------------------- |
| Phase 1.1 UIGenerator      | Phase 2.1–2.2 Tabs + StorageRow | Phase 1.3 Dropdown button             |
| Phase 1.2 Storage select   | Phase 4.1–4.2 Restore           | Phase 2.2 StorageRow (presentational) |
| Phase 3.2 Schedule modal   | Phase 5.1–5.2 PITR              | Phase 2.3 Remove confirm dialog       |
| Phase 6 Wizard backup step |                                 | Phase 4.3 Restore menu action         |
|                            |                                 | Phase 5.2 PITRConfigModal             |
|                            |                                 | Phase 7.1 Legacy cleanup              |

---

## Community-Friendly Tasks Summary

Tasks suitable for delegation — small scope, clear spec, minimal overlap with ongoing work:

| #   | Task                                      | Phase | Complexity | Dependencies | Description                                                                                                |
| --- | ----------------------------------------- | ----- | ---------- | ------------ | ---------------------------------------------------------------------------------------------------------- |
| C1  | "Create backup" dropdown button           | 1.3   | Small      | None         | Replace single button with MUI MenuButton/ButtonGroup with "Now" / "Schedule" options                      |
| C2  | `<StorageRow />` presentational component | 2.2   | Small–Med  | None         | Horizontal bar: name, PITR badge, schedule count, Default chip, action icons. Props-driven, no API calls   |
| C3  | `<StorageRemoveConfirmDialog />`          | 2.3   | Small      | None         | MUI Dialog listing consequences (schedule count, PITR status). Standard confirm/cancel pattern             |
| C4  | `<ScheduledBackupsList />` table          | 3.1   | Medium     | Phase 2.1    | MUI table with columns (Name, Storage, Cron, Retention, Status). Data = flattened `storages[].schedules[]` |
| C5  | Cron picker component                     | 3.2   | Medium     | None         | Reusable cron schedule input. Can reference `steps-old/` for prior art                                     |
| C6  | "Restore" action in backup list           | 4.3   | Small      | Phase 4.2    | Add menu item to existing backup row action menu. Pattern: copy "Delete" action                            |
| C7  | `<PITRConfigModal />` shell               | 5.2   | Small–Med  | Phase 1.1    | MUI Dialog + UIGenerator embed. Title, save/cancel, PATCH Instance                                         |
| C8  | Legacy v1 code removal                    | 7.1   | Small      | Phases 1–6   | Delete listed legacy functions/files once v2 equivalents are confirmed working                             |

**Recommended first community tasks:** C1, C2, C3 (no dependencies, clear visual spec).

---

## Notes

- **Mock data:** Until PR #2226 lands and providers populate `BackupClass.spec.uiSchema`, mock BackupClass
  data is needed for UIGenerator development. Consider a `__mocks__/backupClass.ts` fixture.
- **RBAC:** All v2 hooks have RBAC commented out. Re-enable in Phase 7 or per-phase as each feature stabilizes.
- **`uiSchema` placement:** PR #2226 puts `uiSchema` on `BackupClassSpec` (top level), not inside `providerManaged`.
  Architecture doc proposed it inside `providerManaged`. Clarify with recharte — may need adjustment.
- **`config` on schedule:** Q1 from architecture doc remains open. If schedule needs its own `config`
  (backup type, compression), CRD change is required before Phase 3.2 can render UIGenerator in schedule modal.
