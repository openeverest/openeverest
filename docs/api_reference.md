# OpenEverest API Reference - TLS Support

## TLSSpec

The `TLSSpec` defines TLS configuration for database clusters in OpenEverest.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | No | TLS verification mode. Allowed values: `require`, `verify-ca`, `verify-full` |
| `caSecretRef` | `SecretKeySelector` | Conditional | References the secret containing the CA certificate. Required for `verify-ca` and `verify-full` modes |
| `serverCertSecretRef` | `SecretKeySelector` | Conditional | References the secret containing the server certificate and key. Required for `verify-full` mode |
| `clientCertSecretRef` | `SecretKeySelector` | No | References the secret containing the client certificate and key. Optional, used for mTLS client authentication |

### Validation Rules

1. **Mode validation**: Must be one of `require`, `verify-ca`, or `verify-full`
2. **CA secret requirement**: Required for `verify-ca` and `verify-full` modes
3. **Server cert requirement**: Required for `verify-full` mode
4. **Secret reference validation**: Secret references must be valid Kubernetes object names

### Examples

#### Minimal TLS (require mode)
```yaml
apiVersion: everest.percona.com/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-cluster
spec:
  engine:
    type: postgresql
  tls:
    mode: require
```

#### Full TLS verification (verify-full mode)
```yaml
apiVersion: everest.percona.com/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-cluster
spec:
  engine:
    type: postgresql
  tls:
    mode: verify-full
    caSecretRef:
      name: ca-secret
      key: ca.crt
    serverCertSecretRef:
      name: server-secret
      key: tls.crt
```

#### mTLS with client certificates
```yaml
apiVersion: everest.percona.com/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-cluster
spec:
  engine:
    type: postgresql
  tls:
    mode: verify-full
    caSecretRef:
      name: ca-secret
      key: ca.crt
    serverCertSecretRef:
      name: server-secret
      key: tls.crt
    clientCertSecretRef:
      name: client-secret
      key: tls.crt
```

### Provider Support Matrix

| Provider | Mode | CA Secret | Server Cert | Client Cert |
|----------|------|-----------|-------------|-------------|
| PostgreSQL | Yes | Yes | Yes | Yes |
| MongoDB | Yes | Yes | Partial* | Yes |
| PXC | Yes | Yes | Yes | Partial* |

**Notes:**
- **Yes**: Fully supported
- **Partial***: Supported with additional configuration
- **No**: Not supported

### Migration Guide

#### From previous versions (without TLS)

Existing DatabaseCluster resources without TLS configuration will continue to work as before. The `tls` field is optional and defaults to `nil`.

#### Adding TLS to existing clusters

To add TLS to an existing cluster, update the DatabaseCluster resource:

```bash
kubectl patch databasecluster my-cluster \
  -n everest \
  --type=merge \
  -p='{"spec":{"tls":{"mode":"verify-full","caSecretRef":{"name":"ca-secret","key":"ca.crt"},"serverCertSecretRef":{"name":"server-secret","key":"tls.crt"}}}}'
```

### CLI Usage

```bash
# Create a DatabaseCluster with TLS using the CLI
everestctl create databasecluster my-cluster \
  --engine postgresql \
  --tls-mode verify-full \
  --tls-ca-secret ca-secret \
  --tls-server-cert-secret server-secret \
  --tls-client-cert-secret client-secret

# Update TLS configuration
everestctl update databasecluster my-cluster \
  --tls-mode verify-ca \
  --tls-ca-secret ca-secret
```

### Troubleshooting

#### Common Issues

1. **Invalid mode**: Ensure the mode is one of `require`, `verify-ca`, or `verify-full`
2. **Missing CA secret**: For `verify-ca` and `verify-full` modes, the CA secret is required
3. **Missing server cert**: For `verify-full` mode, the server cert is required
4. **Secret not found**: Ensure the referenced secrets exist in the same namespace
5. **Provider not supported**: Some TLS configurations may not be supported by all providers

#### Error Messages

- `"must be one of: require, verify-ca, verify-full"`: Invalid TLS mode
- `"required for verify-ca and verify-full modes"`: Missing CA secret
- `"required for verify-full mode"`: Missing server cert
- `"unsupported provider for TLS configuration"`: Provider doesn't support TLS
