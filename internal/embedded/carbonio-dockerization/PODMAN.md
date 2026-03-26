# Carbonio on Consul Service Mesh with Podman

## Architecture

Each app pod contains a **Consul client agent** that joins the central Consul server,
loads `.hcl` service definitions locally, and provides a gRPC endpoint for Envoy sidecars
to bootstrap their configuration. This mirrors the VM deployment model 1:1.

```
                    ┌──────────────────────────┐
                    │   Pod: consul-server      │
                    │   consul (server mode)    │
                    │   :8500 (UI/HTTP)         │
                    │   :8502 (gRPC)            │
                    └────────────┬─────────────┘
                                 │
              -retry-join=consul-server (gossip)
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
         ▼                       ▼                       ▼
┌─── Pod: mailbox ───────┐  ┌─ Pod: files ──────┐  ┌─ Pod: preview ─────┐
│ mailbox-app            │  │ files-app         │  │ preview-app        │
│ consul-agent (client)  │  │ consul-agent (cl.)│  │ consul-agent (cl.) │
│ envoy-mailbox          │  │ envoy-files       │  │ envoy-preview      │
│ envoy-mailbox-admin    │  │                   │  │                    │
│ envoy-nslookup         │  └───────────────────┘  └────────────────────┘
│ envoy-internal-api     │
└────────────────────────┘
```

Each pod has:
- The application container
- A **Consul agent in client mode** (~30-50 MB overhead) that joins the central server
- One **Envoy sidecar** per Consul service (bootstrapped via `consul connect envoy`)

All containers in a pod share the same network namespace (localhost).
Cross-pod traffic is routed through Envoy proxies using Consul Connect mTLS.

### Startup
./run.sh up

### Teardown

./run.sh down

## FAQ

1. **DNS resolution across pods**: `podman play kube` with `--network` allows pod-to-pod DNS by pod name.
   Verify that `consul-server` resolves correctly from client agents via `-retry-join`.
2. **Consul ACL bootstrap**: For production-like testing, ACL tokens need to be bootstrapped and
   distributed to each pod. For local dev, ACLs can be disabled (`-acl_default_policy=allow`).
3. **Persistent volumes**: Consul server data should be persisted across restarts for dev stability.
4. **Health check adaptation**: The `carbonio-mailbox.hcl` health check uses `http://localhost:8080/...`
   which works in the pod model (shared namespace). No change needed.
5. Sidecar images are based on [Dockerfile-envoy](./images/Dockerfile-envoy)

## Available features
- Supports only Advanced
- Starts Advanced and Files by default (customization is possible manually)
- Adds monitoring (tracing, logs, metrics) by default, available at 
  localhost:3000 (Grafana)

## TODO
- [ ] Add carbonio-docs-connector
- [ ] Add carbonio-docs-editor
- [ ] Add carbonio-message-broker
- [ ] Add carbonio-message-dispatcher
- [ ] Add carbonio-preview
- [ ] Add carbonio-tasks
- [ ] Add carbonio-videoserver
- [ ] Add carbonio-ws-collaboration
- [ ] Think about adding memcached to the mesh (currently there is only a 
  local memcached in the composed-ui container)
- [ ] Think about extracting consul-agent common config or define an image 
  which supports BOOTSTRAP_TOKEN as env variable

