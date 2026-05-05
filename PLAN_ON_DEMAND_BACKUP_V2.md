//TODO remove after implementation

# Plan: On-Demand Backup v2 Implementation

> **Last updated**: 2025-05-04 (post v2 merge with PR #2030, #2034, #2044)

## TL;DR

Implement on-demand (now) backup for v2 API. BackupClass replaces hard-coded DB-type logic (logical/physical for PSMDB, PITR for PG). UI form is universal — BackupClass config schema drives dynamic fields. No dbType checks in UI. Schedules are out of scope (later).

---

## CHANGE LOG (what changed since initial plan, 2 weeks ago)

### Key PRs merged into v2 since plan was written:

| PR    | Title                                                                                   | Impact on our work                                                                                                                                                                                                                                                                                                   |
| ----- | --------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| #2030 | Hybrid backup architecture: dual execution modes, Restore CRD, BackupProvider interface | **Major**: Backup spec changed (`storageName` instead of `destination.backupStorageName`), new `scheduleName`, `deletionPolicy` fields. New Restore CRD. BackupClass now has `executionMode` (ProviderManaged / Job), `providerManaged.supportsPITR`, `job`/`restoreJob` blocks. Instance has `spec.backup` section. |
| #2034 | Add backup storage API                                                                  | **Major**: Full v2 BackupStorage CRUD API now available at `/clusters/{cluster}/namespaces/{namespace}/backup-storages` with create/list/get/update/delete.                                                                                                                                                          |
| #2044 | Show backup size in UI                                                                  | **Minor, main-only**: Added `size` column to old backup list. Was merged to main/v2 but `size` field does NOT exist in v2 Backup CRD yet. Need to add column when field is available.                                                                                                                                |
| #2045 | Rename backup storage v1                                                                | Renamed old backup storage handler to avoid conflicts with new v2 BackupStorage.                                                                                                                                                                                                                                     |

### What's NEW (not in original plan):

1. **`spec.storageName`** replaces `spec.destination.backupStorageName` — simpler, just a string referencing BackupStorage CR name
2. **`spec.scheduleName`** — empty for on-demand, set by provider for scheduled backups
3. **`spec.deletionPolicy`** — `Delete` (default) or `Retain` — controls data cleanup on CR deletion
4. **`BackupClass.spec.executionMode`** — `ProviderManaged` or `Job` — UI doesn't need to care (both use same Backup API)
5. **`BackupClass.spec.providerManaged.supportsPITR`** — advertises PITR capability
6. **`BackupClass.spec.restoreConfig`** — separate OpenAPI schema for restore config
7. **Restore CRD** — full restore API (`dataSource.backupName`, `dataSource.external`, `dataSource.pitr`)
8. **Instance.spec.backup** — `{enabled, classRef, storages[], schedules[], pitr}` — manages engine-level backup config
9. **BackupStorage v2 CRD** — new `backup.openeverest.io/v1alpha1` BackupStorage (namespaced, S3 only for now)
10. **v2 BackupStorage API** — full CRUD at `/clusters/{cluster}/namespaces/{namespace}/backup-storages`

### What's NOT yet available (still pending/open PRs):

1. **ListBackups for instance** — PR #2040 (Open, not merged)
2. **ListRestores for instance** — PR #2058 (Open, not merged)
3. **Scheduled backups runtime** — PR #2069 merged but provider implementation pending
4. **Backup `size` field** — not in v2 Backup CRD yet (was only in old DatabaseClusterBackup)

---

## API Mapping: Old → New (UPDATED)

| Action               | Old API                                                                          | New API                                                           | Status     |
| -------------------- | -------------------------------------------------------------------------------- | ----------------------------------------------------------------- | ---------- |
| Create backup        | POST /namespaces/{ns}/database-cluster-backups                                   | POST /clusters/{cluster}/namespaces/{ns}/backups                  | ✅ Ready   |
| Get backup           | GET /namespaces/{ns}/database-cluster-backups/{name}                             | GET /clusters/{cluster}/namespaces/{ns}/backups/{backup}          | ✅ Ready   |
| Delete backup        | DELETE /namespaces/{ns}/database-cluster-backups/{name}?cleanupBackupStorage=... | DELETE /clusters/{cluster}/namespaces/{ns}/backups/{backup}       | ✅ Ready   |
| List backups         | GET /namespaces/{ns}/database-clusters/{name}/backups                            | ❌ **MOCK** (PR #2040 open)                                       | ⏳ Pending |
| List backup classes  | N/A                                                                              | GET /clusters/{cluster}/backup-classes                            | ✅ Ready   |
| Get backup class     | N/A                                                                              | GET /clusters/{cluster}/backup-classes/{backupClass}              | ✅ Ready   |
| Create BackupStorage | POST /namespaces/{ns}/backup-storages (old v1)                                   | POST /clusters/{cluster}/namespaces/{ns}/backup-storages          | ✅ Ready   |
| List BackupStorages  | GET /namespaces/{ns}/backup-storages (old v1)                                    | GET /clusters/{cluster}/namespaces/{ns}/backup-storages           | ✅ Ready   |
| Get BackupStorage    | GET /namespaces/{ns}/backup-storages/{name} (old v1)                             | GET /clusters/{cluster}/namespaces/{ns}/backup-storages/{name}    | ✅ Ready   |
| Update BackupStorage | PUT /namespaces/{ns}/backup-storages/{name} (old v1)                             | PUT /clusters/{cluster}/namespaces/{ns}/backup-storages/{name}    | ✅ Ready   |
| Delete BackupStorage | DELETE /namespaces/{ns}/backup-storages/{name} (old v1)                          | DELETE /clusters/{cluster}/namespaces/{ns}/backup-storages/{name} | ✅ Ready   |
| Create Restore       | N/A (old: POST database-cluster-restores)                                        | ❌ Not in HTTP API yet (CRD exists)                               | ⏳ Pending |

## Data Model Changes (UPDATED post PR #2030)

### Backup Spec (final)

```go
BackupSpec {
  InstanceName     string                  // required
  BackupClassName  string                  // required
  StorageName      string                  // required — references BackupStorage CR name in same namespace
  ScheduleName     string                  // optional — empty for on-demand, set by provider mirroring loop
  Config           *runtime.RawExtension   // optional — validated against BackupClass.spec.config.openAPIV3Schema
  DeletionPolicy   BackupDeletionPolicy    // optional — "Delete" (default) or "Retain"
}
```

### Backup Status (final)

```go
BackupStatus {
  ExecutionMode         BackupExecutionMode        // "ProviderManaged" or "Job"
  OperatorBackupRef     *TypedLocalObjectReference // for ProviderManaged (e.g., PerconaServerMongoDBBackup)
  JobName               string                     // for Job mode
  StartedAt             *Time
  CompletedAt           *Time
  LastObservedGeneration int64
  State                 BackupState                // Pending/Running/Succeeded/Failed/Error
  Message               string
  Conditions            []Condition
}
```

### Key changes from original plan:

- ~~`spec.destination.backupStorageName`~~ → **`spec.storageName`** (direct string, not nested)
- ~~`spec.destination.s3`~~ → removed from Backup; inline S3 now lives on BackupStorage CRD
- NEW: `spec.scheduleName` (for scheduled backups)
- NEW: `spec.deletionPolicy` (Delete/Retain)
- NEW: `status.executionMode` (observability)
- NEW: `status.operatorBackupRef` (for ProviderManaged)

## BackupClass Key Facts (UPDATED)

- Cluster-scoped CRD, READ-ONLY API (no create/update for users)
- Created by provider developers / cluster admins as part of provider installation
- `supportedProviders: string[]` — filter by provider when showing in UI
- **`executionMode: ProviderManaged | Job`** — central dispatch field
  - **ProviderManaged**: engine-native agents (PBM, pgBackRest) — provider's reconciler owns backup lifecycle
  - **Job**: external tool via K8s Job (pg_dump, mongodump) — runtime Job controller owns lifecycle
- `config.openAPIV3Schema` — JSON Schema for backup-time config fields
- `restoreConfig.openAPIV3Schema` — JSON Schema for restore-time config fields
- `providerManaged.supportsPITR: bool` — advertises PITR capability
- `displayName` — human-readable label for UI
- `instanceConstraints.requiredFields[]` — JSON paths that must be set on Instance
- If 1 class for provider → auto-select, no field needed
- If multiple → show select with displayName

## DB-Type-Specific Logic — RESOLVED

| Old UI Logic                                | v2 Resolution                                                                                                      |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| PG: auto-enable PITR on backup create       | ✅ **RESOLVED**: Provider responsibility. Instance.spec.backup.pitr.enabled controls this. UI doesn't auto-toggle. |
| PG: auto-disable PITR on last backup delete | ✅ **RESOLVED**: Provider responsibility. UI doesn't auto-toggle.                                                  |
| PG: 3 storage slot limit                    | ✅ **RESOLVED**: BackupClass.spec.instanceConstraints or provider enforcement. No UI limit.                        |
| PSMDB: logical/physical radio               | ✅ **RESOLVED**: Separate BackupClasses or BackupClass.spec.config schema. UI renders from schema.                 |
| PG: schedule limit (3 max)                  | Out of scope (schedules)                                                                                           |

## Implementation Plan (UPDATED)

### Phase 1: API Layer ✅ DONE (already in branch)

1. ✅ **v2 backup types** — `src/shared-types/backups.types.ts`
   - Uses generated types from `@generated/api-types`
   - Backup, BackupStatus enum, BackupClass, BackupList, BackupClassList
   - CreateBackupPayload, GetBackupPayload, DeleteBackupPayload
2. ✅ **v2 API functions** — `src/api/backups.ts`
   - `createBackupOnDemandFn(cluster, namespace, payload)`
   - `getBackupFn(cluster, namespace, backupName)`
   - `deleteBackupFn(cluster, namespace, backupName)`
   - `listBackupClassesFn(cluster)`
   - `getBackupClassFn(cluster, backupClassName)`
   - `listBackupsFn` → **MOCK** returning `{ items: [] }` — awaiting PR #2040
3. ✅ **React Query hooks** — `src/hooks/api/backups/useBackups.ts` & `src/hooks/api/backup-classes/`
   - `useBackupsList(cluster, namespace, instanceName)` — uses mock listBackupsFn
   - `useCreateBackupOnDemand(cluster, namespace)`
   - `useDeleteBackup(cluster, namespace)`
   - `useBackupClassesList(cluster)`
   - `useGetBackupClass(cluster, backupClassName)`
4. ⏳ **RBAC** — commented out, will enable when RBAC for v2 backups is finalized

### Phase 2: UI — Backups Tab on Instance Details (IN PROGRESS)

> **Note**: Currently the page at `pages/db-cluster-details/backups/` uses OLD hooks (`useDbBackups` from `hooks/api/backups-old/`) and OLD types. Need to migrate to v2 hooks.

5. 🔄 **Backups tab** exists but uses old API — needs migration to v2 hooks
6. **BackupsList table** — needs update:
   - Status (Pending/Running/Succeeded/Failed/Error)
   - Name
   - Storage (storageName)
   - Started (status.startedAt)
   - Completed (status.completedAt)
   - ~~Size~~ (NOT in v2 CRD yet — defer, add when field exists)
   - Actions (delete)
7. **Table header** — "Create backup" dropdown → "Now" button
8. ✅ Auto-refresh 10s

### Phase 3: On-Demand Backup Modal (NEEDS REWRITE)

> Current modal uses old types (BackupFormData, DbCluster, old API).
> Need to rewrite with v2 types and simplified spec.

9. **Universal form** (NO dbType checks):
   - `name` — auto-generated `backup-{shortUID}`, editable
   - `backupClassName` — select from filtered BackupClasses (by supportedProviders matching instance provider)
     - If 1 option → auto-select, hide field
     - If multiple → show select with displayName
   - `storageName` — select from Instance.spec.backup.storages[] (registered storages)
     - **IMPORTANT**: For ProviderManaged classes, storage must be registered on the Instance
     - Fallback: list all BackupStorages in namespace
   - `config` fields — **deferred**: render from BackupClass openAPIV3Schema if present
   - ~~`deletionPolicy`~~ — use default (Delete), don't expose in creation modal
10. **Create payload** (UPDATED):
    ```typescript
    {
      metadata: { name: "backup-xxxxx" },
      spec: {
        instanceName: "my-instance",
        backupClassName: "selected-class",
        storageName: "my-storage",    // <-- simplified! just a string
        // config: { ... }            // optional, from BackupClass schema
      }
    }
    ```

### Phase 4: Delete Backup (UPDATED)

11. **Delete confirmation dialog**
12. ✅ **RESOLVED**: `deletionPolicy` on Backup CR controls cleanup (Delete = remove data, Retain = keep data)
    - UI option: show toggle "Also delete backup data from storage" → sets deletionPolicy before delete
    - Or: simple delete (uses CR's existing deletionPolicy, default=Delete)

### Phase 5: Integrate PR #2044 (Backup Size) for v2

13. **When `size` field is added to v2 Backup CRD**:
    - Add `size` column to backups-list table
    - No dbType check needed (was PSMDB-only in old code, v2 should be universal)
    - Format: human-readable bytes (KB/MB/GB)

### Phase 6: E2E Tests

14. **PR-level tests** (with mocks):
    - On-demand backup form validation (name, storage, backup class selection)
    - Backup RBAC (backups + backup-classes as separate resources)
15. **Release-level tests** — adapt after ListBackups endpoint (PR #2040) is merged

---

## WHAT'S ALREADY DONE IN THIS BRANCH

| Item                                                                | Status                       | File                                               |
| ------------------------------------------------------------------- | ---------------------------- | -------------------------------------------------- |
| v2 Backup types (generated)                                         | ✅ Done                      | `src/shared-types/backups.types.ts`                |
| v2 API functions (create/get/delete/listClasses)                    | ✅ Done                      | `src/api/backups.ts`                               |
| v2 Hooks (useBackupsList, useCreateBackupOnDemand, useDeleteBackup) | ✅ Done                      | `src/hooks/api/backups/useBackups.ts`              |
| v2 BackupClasses hooks (useBackupClassesList, useGetBackupClass)    | ✅ Done                      | `src/hooks/api/backup-classes/useBackupClasses.ts` |
| ListBackups mock (returns empty)                                    | ✅ Done                      | `src/api/backups.ts`                               |
| Backups page structure (tab, list, modals)                          | 🔄 Exists but uses OLD hooks | `src/pages/db-cluster-details/backups/`            |

## WHAT REMAINS TO DO

| #   | Task                                                                                | Priority | Blocked by      |
| --- | ----------------------------------------------------------------------------------- | -------- | --------------- |
| 1   | Migrate BackupsList to v2 hooks (`useBackupsList` instead of `useDbBackups`)        | P0       | —               |
| 2   | Rewrite on-demand modal to use v2 payload (`storageName` string, `backupClassName`) | P0       | —               |
| 3   | Remove old dbType-specific logic from modal (PSMDB radio, PG PITR auto-enable)      | P0       | —               |
| 4   | Add BackupClass select to modal (filter by instance provider)                       | P0       | —               |
| 5   | Add storage select from Instance.spec.backup.storages or namespace BackupStorages   | P0       | —               |
| 6   | Update delete dialog (simple delete, respect existing deletionPolicy)               | P1       | —               |
| 7   | Add `size` column when v2 CRD gets the field                                        | P2       | BE adds field   |
| 8   | Enable RBAC checks in hooks (uncomment)                                             | P2       | RBAC finalized  |
| 9   | Replace mock `listBackupsFn` with real endpoint                                     | P1       | PR #2040 merged |
| 10  | Write PR-level E2E tests                                                            | P1       | Tasks 1-5 done  |

---

## Pending Questions (UPDATED)

| #   | Question                                                                                 | Status                                                                                                                           |
| --- | ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| 1   | ~~cleanupBackupStorage~~ — will v2 DELETE have this param?                               | ✅ **RESOLVED**: `spec.deletionPolicy` (Delete/Retain) on Backup CR controls this. No query param needed.                        |
| 2   | ~~PITR auto-enable/disable~~ — provider responsibility?                                  | ✅ **RESOLVED**: Yes, provider manages via Instance.spec.backup.pitr. UI doesn't auto-toggle.                                    |
| 3   | ~~Storage slot limit~~ — provider enforcement?                                           | ✅ **RESOLVED**: BackupClass.instanceConstraints + provider validation. No UI limit.                                             |
| 4   | ListBackups endpoint timeline?                                                           | ⏳ PR #2040 is open. Need to ask for ETA.                                                                                        |
| 5   | **NEW**: Should the UI allow creating backups if Instance.spec.backup is not configured? | ❓ For ProviderManaged, storage must be registered on Instance. For Job mode, any BackupStorage in NS works. Need clarification. |
| 6   | **NEW**: How to get `provider` name from Instance to filter BackupClasses?               | ❓ Instance.spec.provider? Need to check Instance CRD.                                                                           |
| 7   | **NEW**: Should on-demand backup modal show deletionPolicy option?                       | ❓ Probably not for creation (use default=Delete). Maybe on delete action.                                                       |
| 8   | **NEW**: Backup `size` field — when will it be added to v2 Backup CRD?                   | ⏳ Currently only in old DatabaseClusterBackup (main).                                                                           |

---

## Files Reference (UPDATED — what exists NOW)

| Purpose       | File (exists in branch)                                        | Status            |
| ------------- | -------------------------------------------------------------- | ----------------- |
| v2 Types      | `src/shared-types/backups.types.ts`                            | ✅ Done           |
| v2 API        | `src/api/backups.ts`                                           | ✅ Done           |
| v2 Hooks      | `src/hooks/api/backups/useBackups.ts`                          | ✅ Done           |
| BackupClasses | `src/hooks/api/backup-classes/useBackupClasses.ts`             | ✅ Done           |
| Backup tab    | `src/pages/db-cluster-details/backups/backups.tsx`             | 🔄 Uses old hooks |
| Backups list  | `src/pages/db-cluster-details/backups/backups-list/`           | 🔄 Uses old hooks |
| Modal         | `src/pages/db-cluster-details/backups/on-demand-backup-modal/` | 🔄 Uses old API   |
| Old hooks     | `src/hooks/api/backups-old/useBackupsOld.ts`                   | ⚠️ To remove      |
| Old API       | `src/api/backups-old.ts`                                       | ⚠️ To remove      |
| Old types     | `src/shared-types/backupsOld.types.ts`                         | ⚠️ To remove      |

---

## PR #2044 (Backup Size) — Integration Notes

PR #2044 was merged into v2/main but the changes are for the **old** `DatabaseClusterBackup`:

- Added `status.size?: string` to old backup types
- Added `size` column in backups-list (conditional on `dbType === PSMDB`)
- The v2 `Backup` CRD does **NOT** have a `size` field yet

**Action items for our branch**:

- The old backup list code already has this merged (via v2 merge)
- For v2 on-demand backups: **defer** size column until v2 Backup CRD adds the field
- When field is added: show `Size` column universally (no dbType check — v2 is engine-agnostic)

---

## E2E Tests to Restore/Adapt

| Test                      | File                                               | Priority            |
| ------------------------- | -------------------------------------------------- | ------------------- |
| On-demand form validation | .e2e/pr/db-cluster-details/on-demand-backup.e2e.ts | P1                  |
| Backup RBAC               | .e2e/pr/rbac/backups.e2e.ts                        | P1                  |
| Demand backup full cycle  | .e2e/release/demand-backup.e2e.ts                  | P2 (after BE ready) |
| RBAC demand backup        | .e2e/release/rbac/demand-backup.e2e.ts             | P2                  |
| Backup storage settings   | .e2e/pr/settings/backup-storage.e2e.ts             | P3 (unchanged)      |
| v2 backup API test        | api-tests/tests/backup-storage.spec.ts             | P2 (already exists) |
