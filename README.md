# dev-connect

`dev-connect` enables VS Code Desktop Remote SSH development against private-cloud Linux development servers through the Kubernetes API.

Current status:

- Phase 1 Architecture: approved.
- Phase 2 Functional Specification: approved.
- Phase 3 Non-Functional Requirements: approved.
- Phase 4 Kubernetes Design: approved.
- Phase 5 Client Design: approved.
- Phase 6 Security Specification: approved.
- Phase 7 Testing Strategy: approved.
- Phase 8 Repository Structure: approved.
- Phase 9 CI/CD Design: approved.
- Phase 10 Documentation: approved.

Implementation status:

- Go CLI implementation is in progress and buildable.
- Client builds are generated for Windows amd64, Linux amd64/arm64, and macOS amd64/arm64.
- Kubernetes gateway manifests, Helm chart, and Kustomize resources are present.
- Unit, integration, security, E2E, and local performance tests are wired through `make`.
- Production load testing is intentionally not part of the current implementation slice.

Primary local validation:

```text
make test
make lint
make build-all VERSION=dev
make helm-template
make kustomize-build
```
