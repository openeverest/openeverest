# Tech stack

Currently, our development team has fewer people than components/repositories. It is essential for us to use shared libraries and tools to make our lives easier. It's also OK to bring in new ones if there is a reason, but that reason should be more appealing than just "let's try this new cool package" or "that's overengineering". Also, if we decide to make a change in this list, it's better to change it in all components within a reasonable timeframe.

- Read more
  - [Best practices](./go_best_practices.md)
  - [Code style](./go_best_practices.md#code-style)

## Our technology stack

- [testify](https://github.com/stretchr/testify) or stdlib `testing` package should be used for writing tests. Testify should be used only for `assert` and `require` packages – suites here have some problems with logging and parallel tests. Common setups and teardowns should be implemented with `testing` [subtests](https://golang.org/pkg/testing/#hdr-Subtests_and_Sub_benchmarks).
- [golangci-lint](https://github.com/golangci/golangci-lint) is used for static code checks.
- [Docker Compose](https://docs.docker.com/compose/) is used for a local development environment and in CI.
- [go modules](https://go.dev/ref/mod#introduction) for vendoring.
- [operator-sdk](https://sdk.operatorframework.io/) for building operators.
- [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) as part of operator-sdk for building k8s APIs using CRDs.
- [oapi-codegen](https://github.com/deepmap/oapi-codegen) for generating code from openapi spec for echo framework
- [echo](https://echo.labstack.com/) as an HTTP framework to build REST APIs.

