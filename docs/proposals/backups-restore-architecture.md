# V2 Backup / Restore / PITR — Architecture

## Table of Contents

- [Goals](#goals)
- [Non-goals](#non-goals)
- [Architecture Decisions](#architecture-decisions)
  - [One BackupClass per provider](#one-backupclass-per-provider)
  - [Three separate config schemas](#three-separate-config-schemas)
  - [Hybrid rendering: static React + dynamic UIGenerator](#hybrid-rendering-static-react--dynamic-uigenerator)
  - [PITR is a per-storage property](#pitr-is-a-per-storage-property)
- [CRD Data Model](#crd-data-model)
  - [BackupClass](#backupclass)
  - [Instance.spec.backup](#instancespecbackup)
  - [Backup CR](#backup-cr)
  - [Restore CR](#restore-cr)
  - [BackupStorage CR](#backupstorage-cr)
- [Backend Changes](#backend-changes)
  - [ProviderManagedSpec extension](#providermanagedspec-extension)
  - [CRD / client regeneration](#crd--client-regeneration)
  - [Provider: BackupClass population](#provider-backupclass-population)
  - [Provider: config consumption](#provider-config-consumption)
- [UI Component Architecture](#ui-component-architecture)
  - [Component tree](#component-tree)
  - [StorageRow visual design (Backups tab only)](#storagerow-visual-design-backups-tab-only)
  - [Storage Edit Modal (Backups tab)](#storage-edit-modal-backups-tab)
  - [Backups tab layout](#backups-tab-layout)
  - [Wizard vs Backups tab — two orchestration modes](#wizard-vs-backups-tab--two-orchestration-modes)
  - [WizardBackupsStep architecture](#wizardbackupsstep-architecture)
  - [Restore mode in wizard](#restore-mode-in-wizard)
  - [Default storage (`main` field)](#default-storage-main-field)
  - [Storage selection flow (Backups tab)](#storage-selection-flow-backups-tab)
  - [Storage removal](#storage-removal)
  - [Static vs dynamic fields](#static-vs-dynamic-fields)
  - [UIGenerator integration pattern](#uigenerator-integration-pattern)
- [User Flows](#user-flows)
  - [Flow 1: Create DB with scheduled backups (wizard)](#flow-1-create-db-with-scheduled-backups-wizard)
  - [Flow 2: Create on-demand backup](#flow-2-create-on-demand-backup)
  - [Flow 3: Create schedule — no storages configured yet](#flow-3-create-schedule--no-storages-configured-yet)
  - [Flow 4: Create schedule — storage already exists](#flow-4-create-schedule--storage-already-exists)
  - [Flow 5: Restore from backup](#flow-5-restore-from-backup)
- [Mock Data](#mock-data)
  - [Mock strategy](#mock-strategy)
- [Implementation Phases](#implementation-phases)
  - [Phase 0: Backend — CRD extension + provider support](#phase-0-backend--crd-extension--provider-support)
  - [Phase 1: Foundation — mocks + UISchema support](#phase-1-foundation--mocks--uischema-support)
  - [Phase 2: On-demand backup modal enhancement](#phase-2-on-demand-backup-modal-enhancement)
  - [Phase 3: Storage management](#phase-3-storage-management)
  - [Phase 4: Scheduled backups](#phase-4-scheduled-backups)
  - [Phase 5: Restore](#phase-5-restore)
  - [Phase 6: Instance creation wizard — backups step](#phase-6-instance-creation-wizard--backups-step)
  - [Phase 7: Cleanup](#phase-7-cleanup)
- [File Inventory](#file-inventory)
- [Verification Criteria](#verification-criteria)
- [Open Questions](#open-questions)
- [Future Ideas / TODO](#future-ideas--todo)

---

## Goals

- Dynamic, provider-agnostic backup/restore UI driven by BackupClass schemas
- Shared presentational components between wizard and Backups tab; separate orchestration layers
- Graceful degradation when BackupClass has no `uiSchema`
- Two-level storage model hidden from user — UI presents a single "pick a storage" flow
- BE: extend `ProviderManagedSpec` with `uiSchema`, `storageLimits`, `pitrConfig`
- BE: providers populate BackupClass and consume `Backup.spec.config`, `pitr.config`

## Non-goals

- Job-mode BackupClass support (only ProviderManaged mode)
- Multi-cluster backup orchestration
- Backup policy / retention policy management (beyond per-schedule retention copies)
- RBAC implementation (re-enabled in cleanup phase)

---

## Architecture Decisions

### One BackupClass per provider

One BackupClass per provider with `backupType` as a field inside `Backup.spec.config` (not separate classes like `psmdb-logical`, `psmdb-physical`).

`Instance.spec.backup.classRef` is a single reference — one class per instance. Backup type is a per-backup decision, not per-instance.

### Three separate config schemas

| Schema         | Stored in                                      | Validated against                                                        | Purpose                                     |
| -------------- | ---------------------------------------------- | ------------------------------------------------------------------------ | ------------------------------------------- |
| Backup config  | `Backup.spec.config`                           | `BackupClass.spec.config.openAPIV3Schema`                                | Per-backup: type, provider-specific options |
| Restore config | `Restore.spec.config`                          | `BackupClass.spec.restoreConfig.openAPIV3Schema`                         | Per-restore: future selective restore       |
| PITR config    | `Instance.spec.backup.storages[N].pitr.config` | `BackupClass.spec.providerManaged.pitrConfig.openAPIV3Schema` (proposed) | Continuous PITR tuning (provider-specific)  |

No shared base needed:

- `backupType` (logical/physical) is **only** in backup config — restore type is auto-derived from source backup
- `compressionType`/`compressionLevel` appear in both backup and PITR but for different purposes (backup data vs oplog chunks)
- PITR target (date/latest) is already first-class in `Restore.spec.dataSource.pitr`
- Each schema has a different lifecycle

### Hybrid rendering: static React + dynamic UIGenerator

```
Static React (hardcoded)              Dynamic (UIGenerator from BackupClass)
───────────────────────────           ──────────────────────────────────────
• PITR toggle (SwitchInput)           • PITR config fields — section="pitr"
• Storage select                      • Backup config fields — section="backup"
• Schedule cron picker                • Restore config fields — section="restore"
• Backup name, restore source
• Modal open/close, date picker
```

Backup/restore forms and modals are universal regardless of provider — the only variable part is provider-specific config fields. On the current iteration we hardcode the static layout and embed UIGenerator for dynamic fields.

**Future extensions** (out of scope): conditional rendering (CEL), `datePicker`/`cronPicker` uiTypes, plugin widgets.

### PITR is a per-storage property

V1: instance-level toggle. V2: **per-storage** (`Instance.spec.backup.storages[N].pitr`).

- PSMDB: max 1 PITR-enabled storage (via `storageLimits.maxWithPITR`)
- PostgreSQL: may support PITR on all storages
- Toggle lives on `<StorageRow />`; gear icon opens `<PITRConfigModal />` when config exists
- Disabled when `maxWithPITR` limit reached
- PITR requires at least one active backup schedule (see [Q5](#q5-pitrbackup-schedule-dependency))
- UIGenerator renders `pitr` section for provider-specific config when enabled

---

## CRD Data Model

### BackupClass

```yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: BackupClass
metadata:
  name: psmdb-managed
spec:
  displayName: "Percona Server for MongoDB Backups"
  description: "Physical and logical backups via PBM"
  executionMode: ProviderManaged
  supportedProviders:
    - percona-server-mongodb

  providerManaged:
    supportsPITR: true
    storageLimits:
      max: 3 # max storages per instance
      maxWithPITR: 1 # max PITR-enabled storages
      maxSchedulesPerStorage: 10
    uiSchema: # ui-generator DSL, RawExtension
      backup: # → On-demand backup modal
        label: "Backup Configuration"
        componentsOrder: ["type", "compressionType", "compressionLevel"]
        components:
          type:
            uiType: select
            label: "Backup Type"
            path: type
            required: true
            fieldParams:
              options:
                - { label: "Logical", value: "logical" }
                - { label: "Physical", value: "physical" }
              defaultValue: "logical"
          compressionType:
            uiType: select
            label: "Compression"
            path: compressionType
            fieldParams:
              options:
                - { label: "None", value: "none" }
                - { label: "Gzip", value: "gzip" }
                - { label: "Snappy", value: "snappy" }
                - { label: "LZ4", value: "lz4" }
                - { label: "Zstandard", value: "zstd" }
              defaultValue: "snappy"
          compressionLevel:
            uiType: number
            label: "Compression Level"
            path: compressionLevel
            fieldParams: { defaultValue: 6 }
            validation: { minimum: 0, maximum: 22 }
      pitr: # → Instance storage PITR config sub-form
        label: "PITR Configuration"
        componentsOrder: ["oplogSpanMin", "compressionType"]
        components:
          oplogSpanMin:
            uiType: number
            label: "Oplog Span (minutes)"
            path: oplogSpanMin
            tooltip: "Interval between oplog chunk boundaries"
            fieldParams: { defaultValue: 10 }
            validation: { minimum: 1 }
          compressionType:
            uiType: select
            label: "Oplog Compression"
            path: compressionType
            fieldParams:
              options:
                - { label: "None", value: "none" }
                - { label: "Snappy", value: "snappy" }
                - { label: "Zstandard", value: "zstd" }
              defaultValue: "snappy"
      restore: # → Restore modal dynamic section
        label: "Restore Configuration"
        components: {} # empty for PSMDB initially
```

### Instance.spec.backup

```yaml
spec:
  backup:
    enabled: true
    classRef:
      name: psmdb-managed
    storages:
      - name: s3-main
        storageRef:
          name: my-s3-storage
        main: true
        pitr:
          enabled: true
          config:
            oplogSpanMin: 10
            compressionType: snappy
        schedules:
          - name: daily-full
            cron: "0 2 * * *"
            retentionCopies: 7
            enabled: true
      - name: s3-archive
        storageRef:
          name: my-archive-storage
        main: false
        pitr:
          enabled: false
        schedules:
          - name: monthly-archive
            cron: "0 4 1 * *"
            retentionCopies: 12
            enabled: true
```

### Backup CR

```yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: Backup
metadata:
  name: my-db-backup-2026-05-11
spec:
  instanceName: my-db
  backupClassName: psmdb-managed
  storageName: s3-main
  config: # from UIGenerator backup section
    type: logical
    compressionType: snappy
    compressionLevel: 6
status:
  state: Succeeded
  size: "2.1Gi"
```

### Restore CR

```yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: Restore
metadata:
  name: restore-from-backup-001
spec:
  instanceName: my-db
  dataSource:
    backupName: my-db-backup-2026-05-11
    pitr: # optional, only if PITR restore
      type: date
      date: "2026-05-11T01:30:00Z"
  config: {} # from UIGenerator restore section
status:
  state: Succeeded
```

### BackupStorage CR

```yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: BackupStorage
metadata:
  name: my-s3-storage
  namespace: my-ns
spec:
  type: s3
  s3:
    bucket: my-backup-bucket
    region: us-east-1
    endpointURL: https://s3.amazonaws.com
    credentialsSecretName: s3-creds-secret
    verifyTLS: true
    forcePathStyle: false
```

---

## Backend Changes

### ProviderManagedSpec extension

Current state in `api/backup/v1alpha1/backupclass_types.go`:

```go
type ProviderManagedSpec struct {
    SupportsPITR bool `json:"supportsPITR,omitempty"`
}
```

Required change:

```go
type ProviderManagedSpec struct {
    SupportsPITR  bool                  `json:"supportsPITR,omitempty"`
    StorageLimits *StorageLimitsSpec    `json:"storageLimits,omitempty"`
    UISchema      *runtime.RawExtension `json:"uiSchema,omitempty"`
    // PITRConfig schema for Instance.spec.backup.storages[N].pitr.config validation
    PITRConfig    BackupClassConfig     `json:"pitrConfig,omitempty"`
}

type StorageLimitsSpec struct {
    // Max is the maximum number of storages per instance.
    Max *int32 `json:"max,omitempty"`
    // MaxWithPITR is the maximum number of PITR-enabled storages.
    MaxWithPITR *int32 `json:"maxWithPITR,omitempty"`
    // MaxSchedulesPerStorage is the maximum number of schedules per storage.
    MaxSchedulesPerStorage *int32 `json:"maxSchedulesPerStorage,omitempty"`
}
```

### CRD / client regeneration

After modifying `ProviderManagedSpec`:

1. `make generate` — regenerates CRD OpenAPI in `api/openapi/crds.gen.yaml`
2. `make generate-client` — regenerates `client/everest-client.gen.go`
3. No server handler changes — handlers pass through K8s objects, new fields flow automatically
4. No new HTTP endpoints — BackupClass is already read-only (`GET /clusters/{cluster}/backup-classes`)

### Provider: BackupClass population

Each provider creates a BackupClass CR at install time (Helm chart).
Full YAML example: see [BackupClass](#backupclass) section above.

Helm chart path: `provider-percona-server-mongodb/charts/.../templates/backup-class.yaml`

### Provider: config consumption

Currently PSMDB provider's `SyncBackup()` ignores `Backup.spec.config`. Required changes in `provider-percona-server-mongodb`:

| Function            | Current                                             | Required                                                              |
| ------------------- | --------------------------------------------------- | --------------------------------------------------------------------- |
| `SyncBackup()`      | Sets only `ClusterName`, `StorageName`              | Read `backup.Spec.Config` → set `psmdbBackup.Spec.Type`, compression  |
| `SyncRestore()`     | Reads only PITR from `restore.Spec.DataSource.PITR` | Read `restore.Spec.Config` (future: selective restore params)         |
| `buildBackupSpec()` | PITR = simple bool from `storage.PITR.Enabled`      | Read `storage.PITR.Config` → set `oplogSpanMin`, compression for PITR |
| `BackupCustomSpec`  | Empty `struct{}`                                    | Not needed — config comes from Backup CR, not provider definition     |

---

## UI Component Architecture

### Component tree

```
Instance Details Page
└── Tab: Backups (backups.tsx)  — data source: API hooks, mutations: PATCH Instance
    │
    ├── <BackupsListTableHeader>
    │   └── [Create backup ▼] ──→ <MenuButton> dropdown
    │       ├── "Now"      → <OnDemandBackupModal>
    │       └── "Schedule" → <ScheduledBackupModal> (with storage select)
    │
    ├── MUI Tabs: [Storages] [Schedules]
    │
    │   Tab: Storages
    │   │  List of instance-bound storages.
    │   │  Each row is a horizontal bar (StorageRow) — see "StorageRow visual design".
    │   ├── <StorageRow> per instance.spec.backup.storages[]
    │   │   ├── Display: name, PITR toggle, schedule count, Default badge (first only)
    │   │   ├── PITR toggle (inline) → PATCH Instance
    │   │   │   ├── If provider has pitr config: ON opens <PITRConfigModal>
    │   │   │   ├── Disabled with tooltip when maxWithPITR limit reached
    │   │   │   └── If modal dismissed without save → toggle stays OFF
    │   │   ├── [⚙] → opens <PITRConfigModal> (shown only if PITR enabled + provider has config)
    │   │   └── [🗑] → <StorageRemoveConfirmDialog>
    │
    │   Tab: Schedules
    │   │  Flat list of ALL schedules across all storages.
    │   └── <ScheduledBackupsList>
    │       Columns: Name | Storage | Schedule | Retention | Status | Actions
    │       Row actions: [Edit] [Delete]
    │
    └── <BackupsList> (existing table)
        └── Row actions: [Restore] [Delete]
```

### StorageRow visual design (Backups tab only)

`<StorageRow>` is used **only in the Backups tab** (instance details page), not in the wizard.
It's a **horizontal bar** on full container width, ~48–56px tall. Similar to v1 `EditableItem`.

```
┌──────────────────────────────────────────────────────────────────────┐
│  s3-prod     PITR: [● ON]   active schedules: 2   Default   [⚙] [🗑] │
└──────────────────────────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────────────────────────┐
│  s3-dev      PITR: [○ OFF]  active schedules: 1                 [🗑] │
└──────────────────────────────────────────────────────────────────────┘

Note: Maximum 3 storages for MongoDb provider
```

**PITR toggle behavior:**

- Toggle OFF → ON: if provider has `uiSchema.pitr.components` (non-empty), opens `<PITRConfigModal>`. If modal dismissed → toggle reverts to OFF.
- Toggle ON → OFF: confirmation → PATCH Instance.
- If provider has no pitr config (just on/off): toggle directly, no modal.
- **Disabled with tooltip** when `maxWithPITR` limit is reached and this storage's PITR is OFF (e.g. "Maximum 1 PITR-enabled storage for this provider").
- **Disabled** when no active schedules exist on any storage (PITR requires schedules — see [Q5](#q5-pitrbackup-schedule-dependency)).

**[⚙] gear button:**

- Shown **only** when provider has `pitr` config schema AND PITR is enabled on this storage
- Opens `<PITRConfigModal>` for editing existing PITR config
- If no pitr config schema → gear is hidden (nothing to configure)

**[🗑] delete:**

- Opens `<StorageRemoveConfirmDialog>` (see [Storage removal](#storage-removal))

### Storage Edit Modal (Backups tab)

Triggered from [⚙] gear on a storage row (when PITR is enabled and provider has config), or from PITR toggle ON → provider has config.

```
┌─────────────────────────────────────────────────────────┐
│  Configure PITR — s3-prod                          [✕]  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  PITR is enabled for this storage.                     │
│                                                         │
│  ┌─ UIGenerator: pitr section ────────────────────┐    │
│  │                                                 │    │
│  │  Oplog Span (min)                               │    │
│  │  ┌──────────────────────────────────────┐       │    │
│  │  │ 10                                   │       │    │
│  │  └──────────────────────────────────────┘       │    │
│  │                                                 │    │
│  │  Compression Type                               │    │
│  │  ┌──────────────────────────────────────┐       │    │
│  │  │ snappy                            ▼  │       │    │
│  │  └──────────────────────────────────────┘       │    │
│  │                                                 │    │
│  │  Compression Level                              │    │
│  │  ┌──────────────────────────────────────┐       │    │
│  │  │ 6                                    │       │    │
│  │  └──────────────────────────────────────┘       │    │
│  │                                                 │    │
│  └─────────────────────────────────────────────────┘    │
│                                                         │
│                          [Cancel]  [Save]               │
└─────────────────────────────────────────────────────────┘
```

- **Title:** "Configure PITR — {storageName}"
- **Content:** UIGenerator renders the `pitr` section from BackupClass uiSchema.
  Fields are provider-defined (e.g. PSMDB: oplogSpanMin, compressionType, compressionLevel).
- **Save:** PATCH Instance → update `storages[i].pitr.config` with UIGenerator values.
- **Cancel:** Dismiss without changes. If opened from PITR toggle ON → toggle reverts to OFF.
- If provider has no `pitr.config` schema: modal is not needed — toggle is simple on/off.

### Backups tab layout

Storages and Schedules are **MUI Tabs** (horizontal tab bar):

```
┌─────────────────────────────────────────────────────────────┐
│                               [Create backup ▼]            │
├──────────┬──────────────────────────────────────────────────┤
│ Storages │ Schedules                                        │  ← MUI Tabs
├──────────┴──────────────────────────────────────────────────┤
│  (active tab content: StorageRows or SchedulesList)       │
└─────────────────────────────────────────────────────────────┘

├── Backups list (always visible below tabs)
│   Status | Name | Size | Started | Finished | Type
│   Row actions: [Restore] [Delete]
```

### Wizard vs Backups tab — two orchestration modes

|                      | Wizard (create/edit DB)                   | Backups tab (existing DB)       |
| -------------------- | ----------------------------------------- | ------------------------------- |
| Instance             | Does not exist yet                        | Exists                          |
| Data source          | React Hook Form state                     | API hooks (useQuery)            |
| Persistence          | All at once on POST Instance              | Each action → PATCH Instance    |
| Storage select scope | ALL namespace-level BackupStorages        | Bound + unbound NS storages     |
| Storage management   | No storage rows — implicit via selections | StorageRow list in Storages tab |
| Shared components    | ScheduleFormFields, PITRConfigFields      | Same                            |
| Orchestration        | `<WizardBackupsStep />`                   | `<BackupsTab />`                |
| spec.backup assembly | `buildSpecBackup()` from form state       | Already persisted in CRD        |

Presentational sub-components are shared. Orchestration is separate.

### WizardBackupsStep architecture

`<WizardBackupsStep />` is a **built-in step** rendered via Provider uiSchema.

> **⚠️ Open question (Q2):** The `builtIn` mechanism below is a starting point. We may
> evolve toward describing sub-elements (PITR, schedules, warnings) via standard ui-schema
> vocabulary. See [Q2](#q2-builtin-mechanism--befe-and-evolution-path).

**`builtIn` key on Section:**

```yaml
# Provider.spec.uiSchema (per topology)
replicaSet:
  sections:
    backups:
      label: "Backups"
      description: "Configure backup schedules and PITR"
      builtIn: "backups" # renders built-in React component, not UIGenerator
  sectionsOrder: [resources, backups, advanced]
```

Provider controls: opt-in/opt-out (omit section), ordering (`sectionsOrder`), labeling.

**form-engine change** (`use-form-engine.ts`):

```typescript
const builtInComponents: Record<string, ComponentType<StepProps>> = {
  backups: WizardBackupsStep,
};

// In step generation:
if (section.builtIn && builtInComponents[section.builtIn]) {
  return {
    id: `section:${key}`,
    label: section.label,
    component: builtInComponents[section.builtIn],
    fields: [],
  };
}
```

**WizardBackupsStep internals:**

- Reads form state via `useFormContext()` — no API mutations
- **Scheduled Backups:** Flat list of `EditableItem` tiles. Click → `ScheduleFormDialog`.
- **PITR:** Provider-aware. Single-PITR: toggle + storage select. Multi-PITR: flat PITR list with [+ Add PITR].
- **Storage select:** ALL namespace-level BackupStorages (instance doesn't exist yet).
- On submit: `buildSpecBackup()` auto-assembles `spec.backup.storages[]` from selections

### Restore mode in wizard

Wizard supports **restore to new cluster** (same as v1).

**Flow:** Instance Details → backup row → [Restore to New DB] → `<RestoreDbModal>` (backup/PITR selection) → "To new cluster" → navigates to `/databases/new` with router state (`selectedDbCluster`, `backupName`, `pointInTimeDate`).

**Wizard in restore mode:**

- Detected via `useDatabasePageMode()` — checks `location.state.selectedDbCluster`
- Source cluster's config loaded as form defaults
- Fields locked: namespace read-only, name may be pre-filled
- On submit: POST Instance with `spec.dataSource.dbClusterBackupName` + optional `pitr`
- User can modify backup/schedule config before creating the new cluster

### Default storage (`main` field)

`main: true` on `InstanceBackupStorage` marks the engine's **default storage**. At most one per instance (not enforced by CEL).

- **Auto-assigned:** First storage bound to instance gets `main: true` automatically
- **1st iteration:** Default is always the first storage. No manual reassignment UI.
- **UI (Backups tab):** Shown as filled "Default" chip on the first storage row

> **Future improvement:** Allow user to reassign default via clickable chip on storage rows. See [Future Ideas](#future-ideas--todo).

### Storage selection flow (Backups tab)

The two-level storage model is **hidden from the user:**

1. Storages are bound **implicitly** when user creates a schedule or on-demand backup targeting an unbound storage
2. Storage row appears automatically in Storages tab after binding
3. No explicit [+ Add Storage] button in first iteration (see [Future Ideas](#future-ideas--todo))
4. Storage select in modals shows existing namespace-level BackupStorages only. Inline creation of new NS-level BackupStorage is deferred (see [Future Ideas](#future-ideas--todo)).

### Storage removal

Removing a storage from an instance = removing entry from `instance.spec.backup.storages[]`.

**Blocking conditions** (Remove button disabled with tooltip):

- Active backup in progress (Running/Pending) to this storage

**Warning conditions** (confirmation dialog lists consequences):

- Storage has N active schedules → "N schedules will be deleted"
- PITR is enabled on this storage → "PITR will be disabled"
- Storage has existing backups → "Existing backups remain accessible but no new backups to this storage"
- This is the only storage → "Backups will be fully disabled"

**Validation flow:**

1. UI pre-checks conditions from local state (schedules count, pitr.enabled, main flag, backup statuses)
2. Shows `<StorageRemoveConfirmDialog>` with consequence list
3. On confirm → PATCH Instance (remove from storages[])
4. BE validates server-side and can reject with error

### Static vs dynamic fields

| Context          | Static fields (React)          | Dynamic fields (UIGenerator)                                                                                                                                    |
| ---------------- | ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| On-demand backup | name, backupClass, storage     | `backup` section: e.g. backupType, compressionType, compressionLevel                                                                                            |
| PITR config      | pitr.enabled toggle            | `pitr` section: e.g. oplogSpanMin, compressionType                                                                                                              |
| Restore          | source info, PITR date picker  | `restore` section: (empty for PSMDB initially)                                                                                                                  |
| Schedule         | name, cron, retention, enabled | `backup` section: same as on-demand (backupType, compression) — requires CRD change, see [Q1](#q1-should-schedule-carry-its-own-config-backup-type-compression) |
| Storage row      | name, Default badge            | _(none — display only + action buttons)_                                                                                                                        |

### UIGenerator integration pattern

Static React modal embeds `<UIGenerator sectionKey="..." />` for provider-specific fields.
UIGenerator renders only fields (no section title/chrome) inside the same `<FormProvider>`.

```tsx
const OnDemandBackupFieldsWrapper = () => {
  const selectedClassName = watch("backupClassName");
  const { data: backupClass } = useGetBackupClass(selectedClassName);
  const sections = backupClass?.spec?.providerManaged?.uiSchema;

  return (
    <>
      <TextInput name="name" label="Backup Name" />
      <SelectInput name="backupClassName" label="Backup Class" />
      <SelectInput name="storageName" label="Storage" />
      {sections?.backup && (
        <UIGenerator
          sectionKey="backup"
          sections={sections}
          formMode="new"
          namespace={namespace}
        />
      )}
    </>
  );
};
```

- On submit, dynamic values are packed into `spec.config`
- No `uiSchema` → no dynamic fields shown (graceful degradation)
- No UIGenerator adaptation needed — already works this way in `SectionEditModal`

---

## User Flows

### Flow 1: Create DB with scheduled backups (wizard)

```
User opens "Create Database" form
│
├── Steps 1-N: Instance config (topology, resources, etc.)
│
└── Step: Backups
    │
    ├── [Enable Backups] toggle → ON
    ├── BackupClass select (auto-select if one)
    │
    ├── Scheduled Backups:
    │   ┌────────────────────────────────────────────────┐
    │   │ daily-full  Every day, 2:00 AM  s3-prod   [✎] │  ← EditableItem tile
    │   └────────────────────────────────────────────────┘
    │   ┌────────────────────────────────────────────────┐
    │   │ weekly-arch  Sun, 3:00 AM  s3-archive     [✎] │
    │   └────────────────────────────────────────────────┘
    │   [+ Create backup schedule]
    │       → <ScheduleFormDialog>
    │         ├── Backup name: "daily-full"
    │         ├── Storage: [s3-prod ▼]  ← ALL namespace-level BackupStorages
    │         ├── Retention copies: 7
    │         └── Time: Every [day ▼] at [2] [00] [AM]
    │
    ├── Point-in-time Recovery:
    │   ├── PITR provides continuous backups of your database,
    │   │   enabling you to restore it to a specific point in time
    │   │   in case of accidental writes or deletes.
    │   ├── Single-PITR provider (`maxWithPITR: 1`):
    │   │   ├── [Enable PITR] toggle → ON
    │   │   └── Backups storage: [S3 compatible ▼]  ← storage select
    │   │       If provider constrains PITR to schedule storage → auto-set, locked
    │   │       If provider allows separate PITR storage → user picks
    │   │       If provider has pitr.config → <PITRConfigModal>
    │   └── Multi-PITR provider:
    │       ├── PITR entries:
    │       │   ┌─────────────────────────────────────────────────┐
    │       │   │ wal-main  storage: repo1  enabled        [✎]   │
    │       │   └─────────────────────────────────────────────────┘
    │       └── [+ Add PITR] → choose storage + provider config
    │
    ├── No storages in namespace?
    │   └── <BackupsActionableAlert>
    │       "No backup storages found. Create one in Settings → Backup Storages."
    │
    └── [Submit] → buildSpecBackup():
        - Collects unique storages from schedule[].storage + PITR entry storages
        - Builds storages[] entries (binding is auto-created)
        - First used storage → main: true (auto)
        - Groups schedules by storage
        - Attaches pitr config to each PITR-enabled storage entry
        Result:
          enabled: true
          classRef: { name: "<provider-backup-class>" }
          storages:
          - name: "s3-prod"
            storageRef: { name: "s3-prod" }
            main: true
            pitr: { enabled: true, config: { oplogSpanMin: 10 } }
            schedules:
            - { name: "daily-full", cron: "0 2 * * *",
                retentionCopies: 7, enabled: true }
          - name: "s3-archive"
            storageRef: { name: "s3-archive" }
            schedules:
            - { name: "weekly-arch", cron: "0 3 * * 0",
                retentionCopies: 4, enabled: true }
```

> **Key difference from storage-row model:** User never sees or manages storage rows
> in the wizard. Storages are selected per-schedule and per-PITR. The instance-level
> binding (`storages[]`) is assembled on submit automatically from the set of unique
> storages referenced by schedules and PITR. No "[+ Add Storage]" button —
> additional storages are auto-created when a schedule references a new one.

**Provider-specific PITR behavior:**

| Provider constraint                   | Wizard PITR UI                                              | Notes                               |
| ------------------------------------- | ----------------------------------------------------------- | ----------------------------------- |
| `maxWithPITR: 1`, no separate storage | One toggle, storage auto-set to first schedule's storage    | Locked/hidden storage choice        |
| `maxWithPITR: 1`, separate storage    | One toggle + one storage select                             | User selects PITR storage           |
| `maxWithPITR: N` or unlimited         | PITR list with [+ Add PITR], one entry per storage / stream | Same mental model as schedules list |
| `supportsPITR: false` or not set      | PITR section hidden                                         | —                                   |

### Flow 2: Create on-demand backup

```
Instance Details → Tab: Backups → [Create backup ▼] → "Now"
│
▼
<OnDemandBackupModal>
├── Backup Name (auto-generated, editable, unique check)
├── BackupClass select → fetches uiSchema.backup
├── Storage select (registered + unregistered namespace storages)
├── <UIGenerator section="backup" />  ← dynamic fields
│   ├── Backup Type: [Logical ▼]
│   ├── Compression: [snappy ▼]
│   └── Compression Level: [6]
└── [Create]
    ├── If unregistered storage → PATCH Instance to add storage
    └── POST Backup CR with spec.config from UIGenerator
```

### Flow 3: Create schedule — no storages configured yet

```
Instance Details → Tab: Backups
│
├── Storages tab: (empty) → "No storages configured"
│   (first iteration: storages are added implicitly when creating schedules/backups)
│
├── User clicks [Create backup ▼] → "Schedule"
│   → <ScheduledBackupModal>
│   ├── Storage select: all NS-level BackupStorages
│   ├── Name, Cron, Retention, Enabled
│   └── [Create] → PATCH Instance (auto-binds storage + adds schedule)
│
├── Storage row appears in Storages tab (main: true, auto)
│   ┌──────────────────────────────────────────────────────────────────┐
│   │ s3-prod  PITR: [○ OFF]  active schedules: 1   Default   [🗑] │
│   └──────────────────────────────────────────────────────────────────┘
│
└── Schedule appears in Schedules tab
```

### Flow 4: Create schedule — storage already exists

```
Instance Details → Tab: Backups
│
├── Storages tab:
│   ┌──────────────────────────────────────────────────────────────────┐
│   │ s3-prod  PITR: [○ OFF]  active schedules: 0   Default   [🗑] │
│   └──────────────────────────────────────────────────────────────────┘
│
├── [Create backup ▼] → "Schedule" → <ScheduledBackupModal>
│   ├── Storage: searchable select (s3-prod pre-selected or user picks any NS storage)
│   │   Selecting unbound storage → auto-binds on save (PATCH Instance)
│   ├── Schedule Name: "daily-backup"
│   ├── Cron: Every day at 2:00 AM
│   ├── Retention: 7 copies
│   ├── Enabled: ON
│   └── [Create] → PATCH Instance (conflict retry)
│
└── Schedules table:
    | Name         | Storage | Schedule        | Retention | Status  | Actions      |
    | daily-backup | s3-prod | Every day, 2 AM | 7 copies  | Enabled | [Edit] [Del] |
```

### Flow 5: Restore from backup

```
Instance Details → Tab: Backups
│
├── Backups list: backup-001 (Succeeded, 2.1GB)
│
├── [Restore] on backup-001 → <RestoreModal>
│   ├── Source info (read-only): backup name, storage, date, size
│   ├── PITR section (if storage had PITR enabled):
│   │   ├── ○ Restore to backup point (default)
│   │   └── ○ Restore to point in time → Date picker
│   ├── <UIGenerator section="restore" />  ← dynamic (empty for PSMDB initially)
│   ├── ⚠️ "Restoring will replace all data. Cannot be undone."
│   └── [Restore]
│       → POST Restore CR with dataSource.pitr + config
│
├── Instance: Ready → Restoring → Ready
│
└── Restores tab: restore-xyz | backup-001 | Succeeded
```

---

## Open Questions

### Q1: Should schedule carry its own `config` (backup type, compression)?

**Current CRD state:** `InstanceBackupSchedule` has NO `config` field — only `name`, `enabled`,
`cron`, `retentionCopies`. Meanwhile `Backup.spec.config` is a `*runtime.RawExtension` set
per-Backup (validated against `BackupClass.spec.config.openAPIV3Schema`).

**Key insight:** A schedule is just a deferred backup. The operator creates `Backup` CRs from
schedules, and each Backup needs a `config`. If on-demand backups have config (backupType,
compression), scheduled backups logically need the same fields — the same UIGenerator `backup`
section should render in both `<OnDemandBackupModal>` and `<ScheduledBackupModal>`.

**Recommendation:** **Option A** — add `config *runtime.RawExtension` to `InstanceBackupSchedule`.

- CRD change required. Provider passes `config` from schedule to each generated Backup CR.
- UI: `<ScheduledBackupModal>` renders UIGenerator `backup` section (same as on-demand modal).
- Enables "physical daily + logical weekly" pattern.
- Consistent UX: same fields in both backup flows.

If a schedule has no `config`, operator falls back to provider defaults (backward-compatible).

### Q2: `builtIn` mechanism — BE/FE and evolution path

Proposal adds `builtIn: "backups"` to Provider uiSchema. Two sub-questions:

**a) Where does `builtIn` live — BE or FE?**

| Option                                         | Description                                                                   |
| ---------------------------------------------- | ----------------------------------------------------------------------------- |
| A) Add to Go `ProviderSpec` (persisted in CRD) | Provider explicitly declares built-in steps. BE validates. Clean contract.    |
| B) FE-only convention                          | UI checks for the key, BE ignores unknown fields in uiSchema. Faster to ship. |

**b) How far do we go with schema-describing the wizard step?**

The `builtIn: "backups"` mechanism gives provider control over presence and ordering, but
the step's internal structure is hardcoded React. Evolution path:

1. **First iteration:** `builtIn` → opaque React component.
2. **Later:** `builtIn` + `components` → built-in component reads components from the section
   schema and renders some fields via UIGenerator alongside its hardcoded layout.
3. **Eventually:** Fully schema-described if the ui-generator vocabulary is rich enough
   (CEL conditions, `schedule-list` uiType, info labels, etc.).

**Key constraint:** Can we do this more flexibly later? Yes — `builtIn` is additive.
We can extend Section to carry both `builtIn` and `components` fields without breaking
existing providers.

**Considerations for future schema-described step:**

- PITR toggle as `uiType: "toggle"` + CEL condition (`visible: "schedules.length > 0"`)
- Schedule list as `uiType: "schedule-list"`
- Warnings/info labels via standard `description` or `info` fields
- The first wizard step (base info) is similarly hardcoded — the same mechanism could serve both

### Q3: Backups tab — storage-centric vs schedule-centric layout

Two design approaches for the Backups tab (instance details):

**Option A: Storage-centric (current proposal)**
Storages tab shows storage rows. Each row has PITR toggle, schedule count, Default chip.
Schedules tab shows flat list.

- **Pro:** Clear ownership — "this storage has these schedules and this PITR config."
- **Con:** Unclear how to show schedule/PITR details per storage without cascading/expanding rows.
  Adding more per-storage config would need modals or multi-level UI.

**Option B: Schedule/PITR-centric (v1-style)**
No storage rows. Show flat schedule list + flat PITR list (or tiles). Each schedule and PITR
has its own storage select column. Storage binding is implicit.

- **Pro:** Matches v1 UX. Simpler. Schedules are the primary concept, storages are attributes.
- **Con:** Harder to see "which storages are bound to this instance" at a glance.

**Option C: Cascading storage-row model (future)**
Storage rows that can expand to reveal their schedules and PITR config inline. Like a tree view.

- **Pro:** Complete per-storage view.
- **Con:** Complex UI. Needs careful design. Risk of deep nesting.

**Current decision:** Option A (storage-centric with Storages/Schedules tabs).
The tabs separate the two concerns cleanly. Option C is interesting for a future design iteration.

### Q4: Multiple PITRs per instance

The CRD allows PITR per-storage (`InstanceBackupStoragePITR`). Multiple storages can each
have `pitr.enabled: true`. However:

> _Engines that support only a single PITR stream (e.g. PSMDB, PXC) require at most one
> storage on the Instance to set `.pitr.enabled=true`; this is enforced by the provider,
> not by the core schema (PG legitimately archives WAL to every configured repo)._

**UI implication:** If a provider supports multiple PITRs, the wizard should show
multiple PITR tiles/items (same visual level as schedule items), each with its own storage
selection and optional provider config. For single-PITR providers, UI keeps the simpler
v1-style toggle + storage select.

**For v2.0:** Keep **different UI by context**:

- **Backups tab:** Storages tab with storage rows; PITR is configured on the storage row.
- **Wizard:** No storages list, to avoid overloading the user. Schedules are shown as a flat list,
  and PITR is shown either as one toggle block or as a flat PITR list depending on provider limits.

Provider validation still prevents invalid combinations when only one PITR stream is allowed.

### Q5: PITR–Backup schedule dependency

**v1 behavior (UI-enforced, no BE validation):**

| Scenario               | Mongo/PXC                                  | MySQL                            | PostgreSQL                                          |
| ---------------------- | ------------------------------------------ | -------------------------------- | --------------------------------------------------- |
| PITR without schedules | Disabled — toggle greyed out               | Disabled — toggle greyed out     | Disabled (no backups = no PITR)                     |
| PITR storage           | Auto-set to first schedule's storage       | User-selectable (separate field) | Auto per-schedule                                   |
| Deleting last schedule | PITR auto-disabled                         | PITR auto-disabled               | PITR auto-disabled                                  |
| PG-specific            | —                                          | —                                | Auto-enabled when backups ON, user can't toggle OFF |
| Alert text             | "PITR relies on an active backup schedule" | Same                             | "PITR is automatically enabled"                     |

**v2 CRD state:** No CRD-level validation tying PITR to schedules. `InstanceBackupStoragePITR`
can be `enabled: true` independently. Single-PITR-stream constraint is provider-enforced.

**v2 options:**

| Option                          | Description                                                                                                                                                                                                                                                                                                                                                                        |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A) Keep v1 behavior in UI       | PITR toggle disabled when `schedules.length === 0`. Simple, safe.                                                                                                                                                                                                                                                                                                                  |
| B) Allow independent PITR       | UI allows enabling PITR without schedules. Provider reports error via `ConditionBackupConfigured=False` if invalid.                                                                                                                                                                                                                                                                |
| C) Provider declares dependency | `BackupClass.spec.providerManaged.pitrRequiresSchedule: bool`. UI reads flag.                                                                                                                                                                                                                                                                                                      |
| D) CEL expression in ui-schema  | Provider describes the constraint as a CEL condition on the PITR toggle (e.g. `enabled: "schedules.length > 0"`). UI evaluates it and auto-enables/disables PITR. This is the most flexible long-term approach — same mechanism can express PG's "always ON" behavior. Requires CEL expression support in ui-generator (see [Q2](#q2-builtin-mechanism--befe-and-evolution-path)). |

**Recommendation:** Start with **Option A** (v1 behavior). UI shows alert:
"Point-in-time recovery requires at least one active backup schedule."
Provider can still reject invalid combinations via condition status.
Option D is the ideal long-term solution — when CEL expressions are implemented in the
ui-generator, the PITR-schedule dependency can be described declaratively per provider.

**PITR storage behavior:**

- Whether PITR storage is user-selectable or auto-assigned depends on provider constraints.
  v1 hardcoded this per DB type; v2 should read from `BackupClass.spec.providerManaged`
  (e.g. `pitrStorageMode: "auto" | "manual"` — future CRD extension).
- For first iteration: hardcode v1 behavior per provider type (same as today).

### Q6: [+ Schedule] on storage row — per-storage schedule creation

Should each storage row have a [+ Schedule] button to open `<ScheduledBackupModal>`
with that storage pre-filled?

- **Pro:** Fast flow — user sees a storage and adds a schedule to it in one click
- **Con:** Overloads the row visually (name + PITR toggle + schedule count + Default + gear + delete + schedule button)
- **first iteration decision:** Omit. Schedules are created via [Create backup ▼] → "Schedule" from the header, with storage select in the modal. Per-storage [+ Schedule] can be added later.

---

## Future Ideas / TODO

Items deferred from first iteration to keep scope manageable:

1. **[+ Add Storage] button** in Storages tab — explicit storage binding flow. Currently storages
   are bound implicitly when creating schedules/backups. Explicit binding allows pre-configuring
   PITR before any schedules exist.

2. **[+ Schedule] per storage row** — shortcut to create schedule with storage pre-filled (Q6).

3. **[+ Create New Storage] inside modals** — inline NS-level BackupStorage creation from
   schedule/backup modal storage selects. Currently user must create storages separately
   (e.g. Settings → Backup Storages) before using them.

4. **Default storage reassignment** — allow user to click a "Default" chip on any storage row
   to reassign `main: true`. Currently default is always the first bound storage.

5. **Cascading storage rows** (Q3 Option C) — rows that expand to show their schedules and PITR
   config inline, like a tree view.

6. **CEL-driven PITR auto-enable/disable** (Q5 Option D) — describe PITR-schedule dependency
   as a CEL condition in the ui-schema, replacing hardcoded logic.

7. **Schema-described wizard backup step** (Q2 evolution) — replace `builtIn: "backups"` with
   granular ui-schema vocabulary for PITR toggles, schedule lists, etc.

8. **Provider-driven PITR storage mode** — `pitrStorageMode: "auto" | "manual"` on
   `BackupClass.spec.providerManaged` to replace hardcoded per-DB-type behavior (Q5).
