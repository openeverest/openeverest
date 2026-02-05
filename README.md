# OpenEverest

![!image](logo.png)

[OpenEverest](https://openeverest.io/) is an open source cloud-native database platform that helps developers deploy code faster, scale deployments rapidly, and reduce database administration overhead while regaining control over their data, database configuration, and DBaaS costs.

Why you should try OpenEverest:

- Launch database instance with just a few clicks
- Enable your team to develop faster and reduce time to market
- Scale seamlessly
- Simplify maintenance
- Monitor and optimize
- Automate backups
- Ensure data security

[Discover all the features and capabilities of OpenEverest](https://openeverest.io/) and see how it can transform your database management experience.

## Documentation

For comprehensive information about OpenEverest, see the [documentation](https://openeverest.io/documentation/current/).

## Install OpenEverest Using Helm (Recommended)

Helm is the recommended installation method for OpenEverest as it simplifies deployment and resource management in Kubernetes environments.

### Prerequisites

- Ensure you have a Kubernetes cluster set up (e.g., Amazon EKS, Google GKE).
- Install Helm on your local machine: [Helm Installation Guide](https://helm.sh/docs/intro/install/).

### Steps to Install

1. **Add the Percona Helm repository:**

```bash
helm repo add percona https://percona.github.io/percona-helm-charts/
helm repo update
```

2. **Install the OpenEverest Helm Chart:**

```bash
helm install everest-core percona/everest \
--namespace everest-system \
--create-namespace
```

3. **Retrieve Admin Credentials:**

```bash
kubectl get secret everest-accounts -n everest-system -o jsonpath='{.data.users\.yaml}' | base64 --decode | yq '.admin.passwordHash'
```

- Default username: **admin**
- You can set a different default admin password by using the server.initialAdminPassword parameter during installation.

4. **Access the OpenEverest UI:**

   By default, OpenEverest is not exposed via an external IP. Use one of the following options:

- Port Forwarding:

```bash
kubectl port-forward svc/everest 8080:8080 -n everest-system
```

Access the UI at http://127.0.0.1:8080.

For more information about our Helm charts, visit the official [OpenEverest Helm Charts repository](https://github.com/percona/percona-helm-charts/tree/main/charts/everest).

## Install OpenEverest using CLI


### Prerequisites

- Ensure you have a Kubernetes cluster set up (e.g., Amazon EKS, Google GKE).

- Verify access to your Kubernetes cluster:

  ```bash
  kubectl get nodes
  ```

- Ensure your `kubeconfig` file is located in the default path `~/.kube/config`. If not, set the path using the following command:

  ```bash
  export KUBECONFIG=~/.kube/config
  ```

## Steps to Install

Starting from version **1.4.0**, `everestctl` uses the Helm chart to install OpenEverest. You can configure chart parameters using:

- `--helm.set` for individual parameters.
- `--helm.values` to provide a values file.

1. **Download the OpenEverest CLI:**

   Linux and WSL

   ```sh
   curl -sSL -o everestctl-linux-amd64 https://github.com/openeverest/openeverest/releases/latest/download/everestctl-linux-amd64
   sudo install -m 555 everestctl-linux-amd64 /usr/local/bin/everestctl
   rm everestctl-linux-amd64
   ```

   macOS (Apple Silicon)

   ```sh
   curl -sSL -o everestctl-darwin-arm64 https://github.com/openeverest/openeverest/releases/latest/download/everestctl-darwin-arm64
   sudo install -m 555 everestctl-darwin-arm64 /usr/local/bin/everestctl
   rm everestctl-darwin-arm64

   ```

   macOS (Intel CPU)

   ```sh
   curl -sSL -o everestctl-darwin-amd64 https://github.com/openeverest/openeverest/releases/latest/download/everestctl-darwin-amd64
   sudo install -m 555 everestctl-darwin-amd64 /usr/local/bin/everestctl
   rm everestctl-darwin-amd64

   ```

2. **Install OpenEverest Using the Wizard:**

   Run the following command and specify the namespaces for OpenEverest to manage:

   ```sh
   everestctl install
   ```

   If you skip adding namespaces, you can add them later:

   ```bash
   everestctl namespaces add <NAMESPACE>
   ```

3. **Install OpenEverest in Headless Mode:**

   Run the following command to set namespaces and database operators during installation:

   ```bash
   everestctl install --namespaces <namespace-name1>,<namespace-name2> --operator.mongodb=true --operator.postgresql=true --operator.mysql=true --skip-wizard
   ```

4. **Access Admin Credentials:**

   Retrieve the generated admin password:

   ```bash
   everestctl accounts initial-admin-password
   ```

5. **Access the OpenEverest UI:**

   Use one of the following methods to access the UI:

- Port Forwarding:

  ```bash
  kubectl port-forward svc/everest 8080:8080 -n everest-system
  ```

  Open the UI at http://127.0.0.1:8080.

- LoadBalancer (Optional):

  ```bash
  kubectl patch svc/everest -n everest-system -p '{"spec": {"type": "LoadBalancer"}}'
  ```

# Need help?

| **Commercial Support** | **Community Support** |
| :---: | :---: |
| Get enterprise-grade support and services for OpenEverest from certified partners. | Connect with our engineers and fellow users for general questions, troubleshooting, and sharing feedback and ideas. |
| **[Get Commercial Support](https://openeverest.io/support/)** | **[Talk to us](https://github.com/openeverest#getting-in-touch)** |

# Contributing

We believe that community is the backbone of OpenEverest. That's why we always welcome and encourage you to actively contribute and help us enhance OpenEverest.

See the [Contribution Guide](CONTRIBUTING.md) for more information on how you can contribute.

## Communication

We value your thoughts and opinions and we would be thrilled to hear from you! [Get in touch with us](https://openeverest.io/#community) to ask questions, share your feedback, and spark creative ideas with our community.

# Submitting Bug Reports

If you find a bug in OpenEverest, [create a GitHub issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/creating-an-issue#creating-an-issue-from-a-repository) in this repository.

Learn more about submitting bugs, new features ideas and improvements in the [documentation](https://openeverest.io/documentation/current/contribute.html).
