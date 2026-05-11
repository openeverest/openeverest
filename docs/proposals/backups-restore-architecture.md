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
  - [`main` field on InstanceBackupStorage](#main-field-on-instancebackupstorage)
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
- Toggle lives in `<PITRConfigModal />` from storage row's [Configure PITR]
- Disabled when `maxWithPITR` limit reached
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
    │   ├── [Create backup ▼] ──→ <MenuButton> dropdown
    │   │   ├── "Now"      → <OnDemandBackupModal>
    │   │   └── "Schedule" → <ScheduledBackupModal> (with storage select)
    │   │
    │   └── [Restore] ──→ <RestoreModal>
    │                      ├── Source info (read-only)                   STATIC
    │                      ├── PITR section (if supportsPITR)            STATIC
    │                      ├── <UIGenerator section="restore" />         DYNAMIC
    │                      └── ⚠️ Warning + [Restore] button
    │
    ├── MUI Tabs: [Storages] [Schedules]
    │
    │   Tab: Storages
    │   │  List of instance-bound storages.
    │   │  Each row is a horizontal bar (StorageRow) — see "StorageRow visual design".
    │   ├── <StorageRow> per instance.spec.backup.storages[]
    │   │   ├── Display: name, [Default] badge, PITR: ON/OFF toggle, N schedules
    │   │   ├── PITR toggle (inline) → PATCH Instance
    │   │   │   └── If provider has pitr config: turning ON opens <PITRConfigModal>
    │   │   │       If modal dismissed without save → toggle stays OFF
    │   │   ├── [⚙] → context menu: [Set as Default], [Configure PITR], [Remove]
    │   │   └── [+ Schedule] → <ScheduledBackupModal> (storage pre-filled)
    │   └── [+ Add Storage] → storage picker
    │       (disabled if >= storageLimits.max)
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

> **[Create backup ▼]** is a `<MenuButton>` dropdown (as in v1), not two separate buttons.
> "Now" opens on-demand backup modal. "Schedule" opens schedule modal with
> storage select (searchable). The [+ Schedule] on each storage row opens
> the same modal with storage pre-filled.

### StorageRow visual design (Backups tab only)

`<StorageRow>` is used **only in the Backups tab** (instance details page), not in the wizard.
It's a **horizontal bar** on full container width, ~48–56px tall. Similar to v1 `EditableItem`
for schedules. **Not** an MUI Card, not a modal, not an expandable section.

```
┌──────────────────────────────────────────────────────────────────────┐
│  s3-prod        [Default]   PITR: [● ON]   2 schedules    [+ Schedule] [⚙] │
└──────────────────────────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────────────────────────┐
│  s3-archive                PITR: [○ OFF]  1 schedule     [+ Schedule] [⚙] │
└──────────────────────────────────────────────────────────────────────┘
[+ Add Storage]
```

**Layout:** `display: flex; align-items: center; gap: 16px;`

- **Left:** storage name (bold)
- **Center:** badges/toggles — `[Default]` chip, PITR toggle (MUI Switch), schedule count
- **Right:** `[+ Schedule]` text button, `[⚙]` icon button (context menu)

**Boundary:** The row itself is **read-only display + inline PITR toggle**.
All editing (PITR config, schedule creation, set-as-default, remove) happens in **modals or menus**.
No fields expand inline — the row never grows taller.

**PITR toggle behavior:**

- Toggle OFF → ON: if provider has `uiSchema.pitr.components` (non-empty), opens `<PITRConfigModal>` with UIGenerator fields. User must save → toggle stays ON. If modal dismissed → toggle reverts to OFF.
- Toggle ON → OFF: simple confirmation → PATCH Instance.
- If provider has no pitr config (just on/off): toggle directly, no modal.

**[⚙] context menu actions:**

- **[Configure PITR]** → opens `<PITRConfigModal>` (same as toggle ON, but for editing existing config)
- **[Set as Default]** → PATCH Instance (set `main: true`, unset on previous default)
- **[Remove]** → `<StorageRemoveConfirmDialog>` (see Storage removal section)

### Storage Edit Modal (Backups tab)

Triggered from [⚙] → [Configure PITR] on a storage row, or from PITR toggle ON → provider has config.

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

> **⚠️ Open question (Q4):** The `builtIn` mechanism below is a starting point. We may
> evolve toward describing sub-elements (PITR, schedules, warnings) via standard ui-schema
> vocabulary. See [Q4](#q4-wizard-backup-step--hardcoded-vs-schema-described).

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
- **Storage select:** ALL namespace-level BackupStorages (instance doesn't exist yet). [+ Create New Storage] if none.
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

### `main` field on InstanceBackupStorage

`main: true` marks the engine's **default storage**. At most one per instance (not enforced by CEL).

- **Auto-assigned:** First storage used by a schedule (wizard) or first bound (tab) gets `main: true`
- **User never picks primary explicitly.** Re-assign via [⚙] → [Set as Default]. No restrictions.
- **Optional:** Zero storages can be main. Removing the main storage leaves no default.
- **UI:** Shown as "Default" badge on StorageRow
- **v2 CRD addition** — not present in v1

### Storage selection flow (Backups tab)

The two-level storage model is **hidden from the user:**

1. [+ Add Storage] → picker shows NS-level BackupStorages not yet bound to instance
2. User selects one (or creates new → S3 form)
3. Storage row appears — instance-level binding created automatically
4. No separate "register storage" step

### Storage removal

Removing a storage from an instance = removing entry from `instance.spec.backup.storages[]`.

**Blocking conditions** (Remove button disabled with tooltip):

- Active backup in progress (Running/Pending) to this storage
- This is the `main` storage and there are other storages (must reassign main first)

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

| Context          | Static fields (React)          | Dynamic fields (UIGenerator)                                         |
| ---------------- | ------------------------------ | -------------------------------------------------------------------- |
| On-demand backup | name, backupClass, storage     | `backup` section: e.g. backupType, compressionType, compressionLevel |
| PITR config      | pitr.enabled toggle            | `pitr` section: e.g. oplogSpanMin, compressionType                   |
| Restore          | source info, PITR date picker  | `restore` section: (empty for PSMDB initially)                       |
| Schedule         | name, cron, retention, enabled | _(none — fully static, maxSchedulesPerStorage as gating limit)_      |
| Storage row      | name, Default badge            | _(none — display only + action buttons)_                             |

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
    │         ├── Storage: [s3-prod ▼]  ← ALL namespace BackupStorages
    │         │   └── [+ Create New Storage] → inline S3 form
    │         ├── Retention copies: 7
    │         └── Time: Every [day ▼] at [2] [00] [AM]
    │
    ├── Point-in-time Recovery:
    │   ├── PITR provides continuous backups of your database,
    │   │   enabling you to restore it to a specific point in time
    │   │   in case of accidental writes or deletes.
    │   ├── Single-PITR provider:
    │   │   ├── [Enable PITR] toggle → ON
    │   │   └── Backups storage: [S3 compatible ▼]  ← storage select
    │   │       (PSMDB: auto-set to first schedule's storage, locked)
    │   │       (MySQL: separate storage picker)
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
    │       "No backup storages found. Create one to enable backups."
    │       [+ Create Backup Storage] → inline S3 form → NLS created
    │
    └── [Submit] → buildSpecBackup():
        - Collects unique storages from schedule[].storage + PITR entry storages
        - Builds storages[] entries (binding is auto-created)
        - First used storage → main: true (auto)
        - Groups schedules by storage
        - Attaches pitr config to each PITR-enabled storage entry
        Result:
          enabled: true
          classRef: { name: "psmdb-managed" }
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

| Engine / capability  | Wizard PITR UI                                                | Notes                               |
| -------------------- | ------------------------------------------------------------- | ----------------------------------- |
| Single-PITR provider | One toggle + one storage select                               | Matches v1-style flow               |
| PSMDB                | One PITR, storage auto-set to first schedule's storage        | Locked/hidden storage choice        |
| MySQL                | One PITR, separate storage picker                             | User selects PITR storage           |
| Multi-PITR provider  | PITR list with `[+ Add PITR]`, one entry per storage / stream | Same mental model as schedules list |

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
│   └── [+ Add Storage] → storage picker
│       ├── Select from available NS-level BackupStorages
│       ├── [+ Create New Storage] → S3 form → creates BackupStorage CR
│       └── [Select] → PATCH Instance (add to storages[], auto-enable backups)
│
├── Storage row appears (main: true, auto)
│   ┌──────────────────────────────────────────────────────────────────┐
│   │ s3-prod  [Default]  PITR: [○ OFF]  0 schedules  [+ Schedule] [⚙] │
│   └──────────────────────────────────────────────────────────────────┘
│
├── User clicks [+ Schedule] on row → <ScheduledBackupModal>
│   (or [Create backup ▼] → Schedule from header)
│   ├── Storage: s3-prod (pre-selected)
│   ├── Name, Cron, Retention, Enabled
│   └── [Create] → PATCH Instance
│
└── Schedule appears in Schedules tab
```

### Flow 4: Create schedule — storage already exists

```
Instance Details → Tab: Backups
│
├── Storages tab:
│   ┌──────────────────────────────────────────────────────────────────┐
│   │ s3-prod  [Default]  PITR: [○ OFF]  0 schedules  [+ Schedule] [⚙] │
│   └──────────────────────────────────────────────────────────────────┘
│
├── [+ Schedule] on row → <ScheduledBackupModal>
│   ├── Storage: "s3-prod" (pre-filled, can change to any NS storage)
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

## Mock Data

### Mock strategy

Until BE populates `BackupClass.spec.providerManaged.uiSchema`, UI uses a mock hook that
injects the uiSchema from a local file. Structure mirrors the [BackupClass](#backupclass) YAML
above, converted to TypeScript `Record<string, Section>`.

| File                                                            | Purpose                                                                      |
| --------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `ui/.../mocks/backup-class-mock.ts`                             | `psmdbBackupClassUISchema` — three sections: `backup`, `pitr`, `restore`     |
| `ui/.../hooks/api/backup-classes/useBackupClassWithUISchema.ts` | Wraps `useGetBackupClass`, injects mock `uiSchema`. Remove when BE is ready. |

```typescript
// useBackupClassWithUISchema.ts — temporary mock wrapper
export const useBackupClassWithUISchema = (className?: string) => {
  const query = useGetBackupClass(className);
  return {
    ...query,
    data: query.data
      ? {
          ...query.data,
          spec: {
            ...query.data.spec,
            providerManaged: {
              ...query.data.spec?.providerManaged,
              uiSchema: psmdbBackupClassUISchema, // from mock file
            },
          },
        }
      : undefined,
  };
};
```

---

## Implementation Phases

### Phase 0: Backend — CRD extension + provider support

| #   | Task                                                                        | File / Repo                                                              | Depends on |
| --- | --------------------------------------------------------------------------- | ------------------------------------------------------------------------ | ---------- |
| 0a  | Extend `ProviderManagedSpec` with `UISchema`, `StorageLimits`, `PITRConfig` | `api/backup/v1alpha1/backupclass_types.go`                               | —          |
| 0b  | Regenerate CRD OpenAPI + client                                             | `make generate && make generate-client`                                  | 0a         |
| 0c  | Create BackupClass manifest for PSMDB                                       | `provider-percona-server-mongodb/charts/.../templates/backup-class.yaml` | 0a         |
| 0d  | Read `Backup.spec.config` in `SyncBackup()` → set backup type + compression | `provider-percona-server-mongodb/internal/provider/backup.go`            | 0a         |
| 0e  | Read `pitr.config` in `buildBackupSpec()` → set oplogSpanMin, compression   | same file                                                                | 0a         |

> **Note:** UI work (Phases 1-7) can proceed in parallel using mock data. Phases 0 and 1+ are independent tracks.

### Phase 1: Foundation — mocks + UISchema support

| #   | Task                                           | File                                                            | Depends on |
| --- | ---------------------------------------------- | --------------------------------------------------------------- | ---------- |
| 1   | Create mock BackupClass with uiSchema sections | `ui/.../mocks/backup-class-mock.ts`                             | —          |
| 2   | Create mock hook `useBackupClassWithUISchema`  | `ui/.../hooks/api/backup-classes/useBackupClassWithUISchema.ts` | 1          |
| 3   | Add `getBackupClassUISection()` utility        | `ui/.../utils/backup-class-utils.ts`                            | —          |
| 4   | Test UIGenerator integration with mock section | test or story                                                   | 1, 2, 3    |

### Phase 2: On-demand backup modal enhancement

| #   | Task                                                            | File                                                                | Depends on |
| --- | --------------------------------------------------------------- | ------------------------------------------------------------------- | ---------- |
| 5   | Add UIGenerator to `OnDemandBackupFieldsWrapper`                | `ui/.../on-demand-backup-modal/on-demand-backup-fields-wrapper.tsx` | 3          |
| 6   | Update submit handler to pack dynamic fields into `spec.config` | `ui/.../on-demand-backup-modal/on-demand-backup-modal.tsx`          | 5          |
| 7   | Update Zod schema for dynamic config                            | `ui/.../on-demand-backup-modal/on-demand-backup-modal.types.ts`     | 6          |

### Phase 3: Storage management (Backups tab)

| #   | Task                                                                  | File                                                               | Depends on |
| --- | --------------------------------------------------------------------- | ------------------------------------------------------------------ | ---------- |
| 8   | Migrate BackupStorage hooks to v2 API                                 | `ui/.../hooks/api/backup-storages/`, `ui/.../api/backupStorage.ts` | —          |
| 9   | Create `<StorageRow />` (presentational — Backups tab only)           | `ui/.../backups/storage-row/storage-row.tsx`                       | 8          |
| 10  | Create `<PITRConfigModal />` with toggle + UIGenerator `pitr` section | `ui/.../backups/pitr-config-modal/pitr-config-modal.tsx`           | 3, 9       |
| 11  | Create `<StorageRemoveConfirmDialog />`                               | `ui/.../backups/storage-remove-confirm-dialog.tsx`                 | —          |
| 12  | Add Storages/Schedules tabs + storage picker to Backups tab           | `ui/.../backups/backups.tsx`                                       | 9, 10, 11  |

### Phase 4: Scheduled backups

| #   | Task                                                              | File                                                                     | Depends on |
| --- | ----------------------------------------------------------------- | ------------------------------------------------------------------------ | ---------- |
| 13  | Create `<ScheduleFormFields />`                                   | `ui/.../backups/scheduled-backup-modal/schedule-form-fields.tsx`         | —          |
| 14  | Implement `<ScheduledBackupModal />`                              | `ui/.../backups/scheduled-backup-modal/scheduled-backup-modal.tsx`       | 13         |
| 15  | Implement `<ScheduledBackupsList />` (flat table)                 | `ui/.../backups/backups-list/table-header/scheduled-backups-list.tsx`    | 14         |
| 16a | Wire [+ Schedule] on storage row                                  | `ui/.../backups/storage-row/storage-row.tsx`                             | 14, 12     |
| 16b | Wire [Create backup ▼] → Schedule in header (with storage select) | `ui/.../backups/backups-list/table-header/backups-list-table-header.tsx` | 14         |

### Phase 5: Restore

| #   | Task                                     | File                                               | Depends on |
| --- | ---------------------------------------- | -------------------------------------------------- | ---------- |
| 17  | Add `createRestoreFn`                    | `ui/.../api/restores.ts`                           | —          |
| 18  | Create `useCreateRestore` hook           | `ui/.../hooks/api/restores/useDbClusterRestore.ts` | 17         |
| 19  | Create `<RestoreModal />`                | `ui/.../backups/restore-modal/restore-modal.tsx`   | 3, 18      |
| 20  | Add restore button to backup row actions | `ui/.../backups/backups-list/backups-list.tsx`     | 19         |

### Phase 6: Instance creation wizard — backups step

Wizard step has **separate orchestration** from the Backups tab: it reads/writes React Hook Form state, not API.
Uses v1-style flat layout: schedules list + PITR section, each with storage select from ALL NS-level BackupStorages.
Shared presentational components: `ScheduleFormFields`, `PITRConfigFields` (UIGenerator wrapper).
Includes `buildSpecBackup()` utility that auto-assembles `spec.backup.storages[]` from schedule/PITR storage selections.

| #   | Task                                                | File                                               | Depends on |
| --- | --------------------------------------------------- | -------------------------------------------------- | ---------- |
| 21  | Create `<WizardBackupsStep />` with v1-style layout | `ui/.../database-form/steps/backups-step/`         | 13         |
| 22  | Create `buildSpecBackup()` utility                  | `ui/.../database-form/steps/backups-step/utils.ts` | —          |
| 23  | Implement restore mode (`WizardMode.Restore`)       | `ui/.../database-form/steps/backups-step/`         | 21         |
| 24  | Register `backupClasses` api-provider               | `ui/.../ui-generator/api-providers/providers.ts`   | —          |
| 25  | Register `backupStorages` api-provider              | `ui/.../ui-generator/api-providers/providers.ts`   | —          |

### Phase 7: Cleanup

| #   | Task                                  | File                 | Depends on |
| --- | ------------------------------------- | -------------------- | ---------- |
| 26  | Remove legacy v1 backup types/imports | various              | all above  |
| 27  | Remove commented-out PITR hook code   | `useBackups.ts`      | all above  |
| 28  | Remove old v1 backup form step        | `steps-old/backups/` | 21         |
| 29  | Enable RBAC checks in hooks           | various hooks        | all above  |

---

## File Inventory

### Files to create (UI)

| File                                                                         | Purpose                                                     |
| ---------------------------------------------------------------------------- | ----------------------------------------------------------- |
| `ui/apps/everest/src/mocks/backup-class-mock.ts`                             | Mock BackupClass with providerManaged.uiSchema              |
| `ui/apps/everest/src/hooks/api/backup-classes/useBackupClassWithUISchema.ts` | Mock hook (temporary)                                       |
| `ui/apps/everest/src/utils/backup-class-utils.ts`                            | `getBackupClassUISection()` utility                         |
| `ui/.../backups/storage-row/storage-row.tsx`                                 | Presentational storage row (Backups tab only)               |
| `ui/.../backups/pitr-config-modal/pitr-config-modal.tsx`                     | PITR toggle + UIGenerator `pitr` section                    |
| `ui/.../backups/storage-remove-confirm-dialog.tsx`                           | Confirmation dialog with pre-check warnings                 |
| `ui/.../backups/scheduled-backup-modal/schedule-form-fields.tsx`             | Presentational schedule form fields (shared: tab + wizard)  |
| `ui/.../backups/restore-modal/restore-modal.tsx`                             | Restore creation modal                                      |
| `ui/.../database-form/steps/backups-step/backups-step.tsx`                   | Wizard backups step (form-state orchestration)              |
| `ui/.../database-form/steps/backups-step/utils.ts`                           | `buildSpecBackup()` — auto-assembles spec.backup.storages[] |

### Files to modify (UI)

| File                                                                     | Change                                                         |
| ------------------------------------------------------------------------ | -------------------------------------------------------------- |
| `ui/.../on-demand-backup-modal/on-demand-backup-fields-wrapper.tsx`      | Add UIGenerator `backup` section                               |
| `ui/.../on-demand-backup-modal/on-demand-backup-modal.tsx`               | Pack dynamic fields into spec.config                           |
| `ui/.../on-demand-backup-modal/on-demand-backup-modal.types.ts`          | Extend Zod schema                                              |
| `ui/.../backups/backups.tsx`                                             | Add Storages/Schedules MUI Tabs + storage picker               |
| `ui/.../backups/backups-list/table-header/backups-list-table-header.tsx` | Restore [Create backup ▼] MenuButton with Now/Schedule options |
| `ui/.../backups/backups-list/backups-list.tsx`                           | Add restore row action                                         |
| `ui/.../backups/scheduled-backup-modal/scheduled-backup-modal.tsx`       | Un-stub, implement                                             |
| `ui/.../backups/backups-list/table-header/scheduled-backups-list.tsx`    | Un-stub, implement                                             |
| `ui/.../hooks/api/backup-storages/useBackupStorages.ts`                  | Migrate to v2 API                                              |
| `ui/.../api/backupStorage.ts`                                            | Migrate endpoints to v2                                        |
| `ui/.../api/restores.ts`                                                 | Add createRestoreFn                                            |
| `ui/.../hooks/api/restores/useDbClusterRestore.ts`                       | Add useCreateRestore                                           |
| `ui/.../ui-generator/api-providers/providers.ts`                         | Register backupClasses + backupStorages                        |

### Files to modify (BE)

| File / Repo                                                              | Change                       |
| ------------------------------------------------------------------------ | ---------------------------- |
| `api/backup/v1alpha1/backupclass_types.go`                               | Extend `ProviderManagedSpec` |
| `api/openapi/crds.gen.yaml`                                              | Regenerated                  |
| `client/everest-client.gen.go`                                           | Regenerated                  |
| `provider-percona-server-mongodb/charts/.../templates/backup-class.yaml` | New: BackupClass manifest    |
| `provider-percona-server-mongodb/internal/provider/backup.go`            | Read config/pitr.config      |

---

## Verification Criteria

1. On-demand modal renders dynamic fields from BackupClass → config sent in `Backup.spec.config`
2. Storages tab shows storage rows with PITR toggle; PITRConfigModal shows dynamic config
3. Scheduled backups CRUD per storage via Instance PATCH
4. Restore modal shows source info, PITR date picker, dynamic restore config
5. Instance wizard backups step renders v1-style layout (schedules + PITR) via builtIn step mechanism
6. All forms degrade gracefully when BackupClass has no `uiSchema`
7. `pnpm lint` and `pnpm typecheck` pass
8. Existing on-demand backup + backups list + restores list unchanged
9. BE: `ProviderManagedSpec` extended, CRD/client regenerated
10. BE: PSMDB provider creates BackupClass at install, reads `Backup.spec.config` in `SyncBackup()`
11. [Create backup ▼] MenuButton works with Now/Schedule options
12. Storage removal confirmation dialog shows pre-check warnings

---

## Open Questions

### Q1: Should schedule carry its own `config` (backup type, compression)?

**Current CRD state:** `InstanceBackupSchedule` has NO `config` field — only `name`, `enabled`,
`cron`, `retentionCopies`. Meanwhile `Backup.spec.config` is a `*runtime.RawExtension` set
per-Backup (validated against `BackupClass.spec.config.openAPIV3Schema`).

**How scheduled backups work:** The operator creates `Backup` CRs from schedules. The operator
decides what `config` to put on each Backup. Currently the operator would use provider defaults
(or none).

**Analysis:**

| Option                                                      | Description                                                  | Pros                                                             | Cons                                                                                                                     |
| ----------------------------------------------------------- | ------------------------------------------------------------ | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| A) Add `config` to `InstanceBackupSchedule`                 | Each schedule defines its own backup type, compression, etc. | Full user control per schedule. Physical daily + logical weekly. | CRD change. UI complexity. Each schedule form gets UIGenerator `backup` section. Provider must pass config to Backup CR. |
| B) Keep schedule simple, inherit from BackupClass defaults  | All scheduled backups use provider-defined defaults.         | Simple. No CRD change. Schedule form stays static.               | Less flexibility. Can't mix backup types across schedules.                                                               |
| C) Config at storage level (`InstanceBackupStorage.config`) | All schedules on a given storage share config.               | Middle ground. Groups config by destination.                     | Doesn't map well to "I want physical daily and logical weekly to same storage."                                          |

**Recommendation:** Start with **Option B** (simple). The v1 schedule had no config at all.
On-demand backups already support per-backup config via the modal. If users need per-schedule
config later, Option A is the cleanest CRD extension (add `config *runtime.RawExtension` to
`InstanceBackupSchedule`).

### Q2: `builtIn` key on Section type — BE or FE only?

Proposal adds `builtIn: "backups"` to Provider uiSchema. This is a new field on the Section type.

| Option                                         | Description                                                                   |
| ---------------------------------------------- | ----------------------------------------------------------------------------- |
| A) Add to Go `ProviderSpec` (persisted in CRD) | Provider explicitly declares built-in steps. BE validates. Clean contract.    |
| B) FE-only convention                          | UI checks for the key, BE ignores unknown fields in uiSchema. Faster to ship. |

### Q3: PITR config — modal vs inline in wizard

In Backups tab, PITR config opens a modal (because the storage row is compact). In wizard step,
PITR config could either: A) Always modal (consistent UX). B) Expand inline below the toggle
(more space in wizard). **v1 reference:** PITR was just a toggle + storage select, no config
modal. If we keep it simple (toggle + storage), no modal needed in wizard at all. Modal only
needed when provider has advanced PITR config fields.

### Q4: Wizard backup step — hardcoded vs schema-described

The `builtIn: "backups"` mechanism gives provider control over presence and ordering, but
the step's internal structure is hardcoded React. Should we go further?

**Considerations:**

- Could PITR toggle be described as a ui-generator field with `uiType: "toggle"` + cell expression
  conditions (e.g. `visible: "schedules.length > 0"`)?
- Could schedule list be a ui-generator component with `uiType: "schedule-list"`?
- Could warnings / info / tech-preview labels be added via standard `description` or `info` fields?
- The first wizard step (base info) is similarly hardcoded and has a separate task for making it
  extensible — the same mechanism could serve both.

**Verdict for v2.0:** Start with `builtIn` (simple). The design allows evolution:

1. `builtIn` today → opaque React component.
2. Later: `builtIn` + `components` → the built-in component reads components from the section
   schema and renders some fields via UIGenerator alongside its hardcoded layout.
3. Eventually: fully schema-described if the ui-generator vocabulary is rich enough.

The key constraint is: **can we do this more flexibly later?** Yes — `builtIn` is additive.
We can extend Section to carry both `builtIn` and `components` fields without breaking existing
providers. The built-in component would merge hardcoded layout with provider-defined extras.

### Q5: Backups tab — storage-centric vs schedule-centric layout

Two design approaches for the Backups tab (instance details):

**Option A: Storage-centric (current proposal)**
Storages tab shows storage rows. Each row has PITR toggle, schedule count, [+ Schedule].
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

**Current decision:** Option A (storage-centric with Storages/Schedules tabs) for v2.0.
The tabs separate the two concerns cleanly. Option C is interesting for a future design iteration.

### Q6: StorageLimits — not in current CRD

`BackupClassInstanceConstraints` currently only has `requiredFields []string`. There is no
`StorageLimits` (min/max) type in the CRD. This proposal references `storageLimits.max`
in several places.

**Options:**

- A) Add `StorageLimits` to `BackupClassInstanceConstraints` as proposed (CRD change needed).
- B) Use the `InstanceBackupSpec.Storages` max array size (currently 10) as the only limit.
  Provider-specific limits enforced via webhook validation, not schema.
- C) Defer — let providers bind as many storages as the array allows; optimize later.

### Q7: Multiple PITRs per instance

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
