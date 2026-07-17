# Carbonio on Consul Service Mesh with Podman

## Requirements
- [podman](https://podman.io/) (4.9 minimum): https://podman.io/docs/installation
- allow bind port 80 and 443 to the host (WebUI): 
```bash
# Create a sysctl configuration file
echo "net.ipv4.ip_unprivileged_port_start=80" | sudo tee /etc/sysctl.d/99-podman-privileged-ports.conf
 
# Apply the configuration immediately
sudo sysctl --system 
```

## Startup

`run.sh` renders the chart for podman (`helm template --set platform=podman | podman play kube -`)
and brings the pods up — `chart/` is the single source of truth for both k3s
and podman. Helm is required.



Start everything: `./run.sh up`

Skip pulling images already present locally: `./run.sh up --missing`

Customize startup with specific groups: `VALUES_FILE=custom_values.yaml ./run.sh up`
Or create a `chart/values.local.yaml` and it will be picked up automatically. 

Example (enable files):
```yaml

# chart/values.local.yaml
groups:
 files:
  enabled: true
```

## Teardown
Stop: `./run.sh down`
Stop and remove volumes: `./run.sh down --force`

Run `./run.sh` to see the usage helper.

## k3s

See [k3s.md](k3s.md) for the k3s deployment.

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

### ACL & Token Management

Consul ACLs are enabled with `default_policy = "deny"`. 

**Default token is `00000000-0000-0000-0000-000000000000`.**
See [consul-acl.yaml](configmaps/consul-acl.yaml) 
for details.

Each sidecar registers its service, creates ACL policies and intentions, and generates a token. 
The token is shared with the app container via a `hostPath` volume mount.


## FAQ

1. **DNS resolution across pods**: `podman play kube` with `--network` allows pod-to-pod DNS by pod name.
   Verify that `consul-server` resolves correctly from client agents via `-retry-join`.
2. **Consul ACL bootstrap**: See [ACL & Token Management](#acl--token-management) above.
3. **Persistent volumes**: Consul server data should be persisted across restarts for dev stability.
4. **Health check adaptation**: The `carbonio-mailbox.hcl` health check uses `http://localhost:8080/...`
   which works in the pod model (shared namespace). No change needed.
5. Sidecar images are based on [Dockerfile-envoy](../images/Dockerfile-envoy)

## Available features
- Supports only Advanced
- Starts Advanced, Files, Docs Connector, Docs Editor, Message Broker, Message Dispatcher, and WS Collaboration by default
- Adds monitoring (tracing, logs, metrics) by default, available at 
  localhost:3000 (Grafana)

## TODO
- [ ] Add carbonio-preview
- [ ] Add carbonio-tasks
- [ ] Add carbonio-videoserver
- [ ] Think about adding memcached to the mesh (currently there is only a 
  local memcached in the composed-ui container)
