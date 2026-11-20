# Controller Package

This package contains the core controller abstractions for building Everest providers.

## Key Files

| File | Purpose |
|------|---------|
| `common.go` | The `Context` handle and resource operations |
| `interface.go` | Provider interface types (`ProviderInterface`, `BaseProvider`) |
| `metadata.go` | Provider spec helper functions and YAML generation |
| `generate.go` | CLI manifest generation utilities |

## Main Concepts

### The Context Handle (`common.go`)

The `Context` struct is the main interface for provider code:

```go
type Context struct {
    ctx          context.Context
    client       client.Client
    in           *v1alpha1.Instance
    providerName string
}

// Key methods:
c.Name()           // Instance name
c.Namespace()      // Instance namespace
c.Spec()           // Instance spec
c.Apply(obj)       // Create/update with owner reference
c.Get(obj, name)   // Read resource
c.Delete(obj)      // Delete resource
c.ProviderSpec()   // Fetch Provider spec from cache (always up-to-date)
```

## Provider Interface

Implement the `ProviderInterface` to create a provider:

```go
type ProviderInterface interface {
    Name() string
	Types() func(*runtime.Scheme) error
	Validate(in ValidationInput) ValidationResult
	Sync(c *Context) error
	Status(c *Context) (Status, error)
	Cleanup(c *Context) error
}
```

`Validate` is pure: it gets the Instance and its Provider, no client, and
returns every error with its field path. The runtime calls it before every
`Sync`, after checking the Instance against the topologies, components,
versions, parameter schemas and `supportedFields` the Provider declares. Rules
about another object's contents, such as the keys of the user Secret, go in the
optional `ReferenceValidator.ValidateReferences(c *Context) (field.ErrorList, error)`:
its errors name the Instance field that refers to the object and report the
Instance with reason `InvalidReference`, and the check runs again every minute,
since the object can change without an event on the Instance.

With `reconciler.WithAdmission()`, the same checks and `Validate` also run in
an admission webhook, so a write is rejected before it is persisted. The
webhook also rejects a backup class that exists but doesn't fit the Instance:
one that runs backups as Jobs, doesn't list the provider, or whose limits or
PITR schema the backup configuration breaks. On an update, `Validate` gets the
previous Instance as `Old`, and errors the stored Instance already had do not
block the write. Objects the Instance names that are missing, or whose
contents `ValidateReferences` rejects, come back as warnings and never block a
write: they may be created after the Instance.

Test `Validate` with `controller/validationtest`, which runs the same checks
against the `provider-spec.yaml` your chart publishes, with Instance fixtures
and no cluster.

### Reporting status

`Status` reports what the provider observes. The runtime turns it into the
Instance's phase, conditions and status word, the same way for every provider,
so a provider chooses no words and no severities:

```go
func (p *MyProvider) Status(c *controller.Context) (controller.Status, error) {
    db := &mydbv1.MyDatabase{}
    if err := c.Get(db, c.Name()); err != nil {
        return controller.Status{}, err
    }
    switch {
    case db.Status.Ready:
        return controller.ReadyWithConnectionDetails(connectionDetails(db)), nil
    case db.Status.Upgrading:
        return controller.Progressing("Rolling out the new version").
            WithAvailable(controller.Available).
            WithOperation(v1alpha1.ConditionUpgrading, "RollingUpdate", db.Status.Message), nil
    case db.Status.Error != "":
        return controller.Progressing("Waiting for the cluster").
            WithProblem("ReplicationBroken", db.Status.Error), nil
    }
    return controller.Progressing("Waiting for the cluster"), nil
}
```

| The provider reports | The Instance shows |
|---|---|
| `Ready()`, `ReadyWithConnectionDetails(...)` | `Ready=True` once every check passes. The engine runs what the Instance asks for and the provider has nothing left to do. |
| `Progressing(message)` | `Ready=False`, `Reconciling=True`, with the message. The status word is `Creating` until the database has first been available and `ApplyingChanges` afterwards. A wait with no named operation is reported as `Stalled` after 30 minutes. |
| `WithOperation(type, reason, message)` | The operation's condition (`Restoring`, `Upgrading`, `StorageResizing`) is True while reported, and its type is the status word. A named operation has no deadline, so name everything that can take long. |
| `WithAvailable(...)`, `WithUnavailable(reason, message)` | The `Available` condition. A provider that does not say is assumed available once converged, not available until the database has first been available, and unknown otherwise. |
| `WithProblem(reason, message)` | `EngineHealthy=False` and `NeedsAttention=True` at once, with the reason as the status word, and `Stalled=True` if the problem is still there after 10 minutes. |
| `WithTerminalProblem(reason, message)` | `Stalled=True` at once. Use it only when nothing but a person can resolve the problem, typically because the engine cannot do what the spec asks. |

What is merely in progress, such as pods that are starting, keeps the Instance
from being Ready and does not change the status word. A problem does: a pod
in a back-off, a missing referent, or one the provider reports.

`Sync` and `Cleanup` return `controller.WaitFor(reason)` to be called again
later; the Instance reads as in progress with the reason as its message. Any
other error from `Sync` or `Status` is retried with backoff. At first it does
not change the status word, and on an Instance that had converged it changes
nothing at all. Once the calls have failed for a minute without a success, it
is reported as a problem with reason `ReconcileError`.

The two deadlines can be changed with `reconciler.WithProblemDeadline` and
`reconciler.WithProgressDeadline`. Their clocks are kept in memory and start
again when the provider restarts, and they assume that one replica of the
provider is running.

Use `BaseProvider` to inherit default implementations:

```go
type MyProvider struct {
    controller.BaseProvider
}

func NewMyProvider() *MyProvider {
    return &MyProvider{
        BaseProvider: controller.BaseProvider{
            ProviderName: "mydb",
            SchemeFuncs:  []func(*runtime.Scheme) error{mydbv1.AddToScheme},
        },
    }
}

// Implement required methods
func (p *MyProvider) Validate(in controller.ValidationInput) controller.ValidationResult { ... }
func (p *MyProvider) Sync(c *controller.Context) error { ... }
func (p *MyProvider) Status(c *controller.Context) (controller.Status, error) { ... }
func (p *MyProvider) Cleanup(c *controller.Context) error { ... }
```
