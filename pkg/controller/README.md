# SDK Controller Package

This package contains the core SDK abstractions for building Everest providers.

## Key Files

| File | Purpose |
|------|---------|
| `common.go` | The `Context` handle and resource operations |
| `interface.go` | Provider interface types (`ProviderInterface`, `BaseProvider`) |
| `metadata.go` | Provider metadata types and conversions |
| `generate.go` | CLI manifest generation utilities |

## Main Concepts

### The Context Handle (`common.go`)

The `Context` struct is the main interface for provider code:

```go
type Context struct {
    ctx      context.Context
    client   client.Client
    db       *v2alpha1.Workload
    metadata *ProviderMetadata
}

// Key methods:
c.Name()           // Workload name
c.Namespace()      // Workload namespace
c.Spec()           // Workload spec
c.Apply(obj)       // Create/update with owner reference
c.Get(obj, name)   // Read resource
c.Delete(obj)      // Delete resource
c.Metadata()       // Provider metadata
```

## Provider Interface

Implement the `ProviderInterface` to create a provider:

```go
type ProviderInterface interface {
    Name() string
    Types() func(*runtime.Scheme) error
    OwnedTypes() []client.Object
    Validate(c *Context) error
    Sync(c *Context) error
    Status(c *Context) (Status, error)
    Cleanup(c *Context) error
}
```

Use `BaseProvider` to inherit default implementations:

```go
type MyProvider struct {
    sdk.BaseProvider
}

func NewMyProvider() *MyProvider {
    return &MyProvider{
        BaseProvider: sdk.BaseProvider{
            ProviderName: "mydb",
            SchemeFuncs:  []func(*runtime.Scheme) error{mydbv1.AddToScheme},
            Owned:        []client.Object{&mydbv1.MyDB{}},
        },
    }
}

// Implement required methods
func (p *MyProvider) Validate(c *sdk.Context) error { ... }
func (p *MyProvider) Sync(c *sdk.Context) error { ... }
func (p *MyProvider) Status(c *sdk.Context) (sdk.Status, error) { ... }
func (p *MyProvider) Cleanup(c *sdk.Context) error { ... }
```

