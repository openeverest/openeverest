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
	Validate(c *Context) error
	Sync(c *Context) error
	Status(c *Context) (Status, error)
	Cleanup(c *Context) error
}
```

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
func (p *MyProvider) Validate(c *controller.Context) error { ... }
func (p *MyProvider) Sync(c *controller.Context) error { ... }
func (p *MyProvider) Status(c *controller.Context) (controller.Status, error) { ... }
func (p *MyProvider) Cleanup(c *controller.Context) error { ... }
```

## Leader election

Providers that run more than one replica should enable leader election when
constructing their reconciler:

```go
r, err := reconciler.New(ctx, provider, reconciler.WithLeaderElection())
```

The runtime derives the Lease name from the provider name as
`<provider-name>.provider.openeverest.io`, so providers deployed in the same
namespace do not contend for leadership. Leadership is released when the
manager stops, allowing a replacement pod to take over promptly during a
rolling update.

Leader election is opt-in for compatibility. A provider that enables it must
grant its service account namespace-scoped `get`, `create`, and `update`
permissions for `coordination.k8s.io` Lease resources. Validation, health, and
readiness remain available on standby replicas.
