# Carbonio on Consul Service Mesh with Podman

## Requirements

- [Podman](https://podman.io/) 4.9 or newer
- [Helm](https://helm.sh/docs/intro/install/)
- Access to the images configured in `chart/values.yaml`; authenticate to the
  default private registry with `podman login registry.dev.zextras.com`
- Permission for rootless Podman to bind the Web UI to host port 443:

```bash
echo "net.ipv4.ip_unprivileged_port_start=80" | sudo tee /etc/sysctl.d/99-podman-privileged-ports.conf
sudo sysctl --system
```

> This is a development environment. The chart contains development passwords,
> ACL tokens, test accounts, and TLS material; do not expose it as production.

## Startup

`run.sh` renders `chart/` with `platform=podman` and passes the result to
`podman play kube`; the chart is the source of truth for both Podman and k3s.

Start the groups enabled in `chart/values.yaml`: `./run.sh up`

Skip pulling images already present locally: `./run.sh up --missing`

Override values with `VALUES_FILE=custom_values.yaml ./run.sh up`, or create
`chart/values.local.yaml`; `run.sh` picks it up automatically.

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

Meshed application pods contain a **Consul client agent** that joins the central
Consul server, loads `.hcl` service definitions locally, and provides a gRPC
endpoint for service sidecars to bootstrap their configuration. Some
infrastructure traffic, including LDAP and memcached, remains outside the mesh.

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
│ sidecar-mailbox        │  │ sidecar           │  │ sidecar            │
│ sidecar-admin          │  │                   │  │                    │
│ sidecar-nslookup       │  └───────────────────┘  └────────────────────┘
│ sidecar-internal-api   │
└────────────────────────┘
```

Each meshed application pod has:
- The application container
- A **Consul agent in client mode** that joins the central server
- One service sidecar per registered Consul service

All containers in a pod share the same network namespace (`localhost`). Meshed
cross-pod traffic is routed through Consul Connect sidecars using mTLS.

### ACL & Token Management

Consul ACLs are enabled with `default_policy = "deny"`. 

**The development bootstrap token is
`00000000-0000-0000-0000-000000000000`.** See
[consul-acl-configmap.yaml](chart/templates/consul-acl-configmap.yaml).

Each sidecar registers its service, creates ACL policies and intentions, and
writes its token to an `emptyDir` volume shared with the application container.


## FAQ

1. **DNS resolution across pods**: `podman play kube` with `--network` allows pod-to-pod DNS by pod name.
   Verify that `consul-server` resolves correctly from client agents via `-retry-join`.
2. **Consul ACL bootstrap**: See [ACL & Token Management](#acl--token-management) above.
3. **Persistent volumes**: Consul server data should be persisted across restarts for dev stability.
4. **Health check adaptation**: The `carbonio-mailbox.hcl` health check uses `http://localhost:8080/...`
   which works in the pod model (shared namespace). No change needed.
5. Sidecar images are based on [Dockerfile-envoy](../images/Dockerfile-envoy)

## Features and defaults

The default deployment uses the Advanced edition and enables only the `mails`
and `ui` groups. Shared infrastructure such as Consul, OpenLDAP, PostgreSQL,
storages, catalog, user management, message broker, and memcached is rendered
whenever any service group is enabled. `carbonio-license-service` joins them on
the Advanced edition only; it reads the `subscription` database through the
`carbonio-mailbox-db` upstream and the license from Consul KV.

Available values include:

```yaml
edition: advanced       # advanced or ce
image:
  pullPolicy: Always    # use IfNotPresent for local-only images
monitoring:
  enabled: false        # enables OTEL LGTM and Grafana on port 3000
docsEditor:
  enabled: false        # requires groups.files and an image override/import
provision:
  auto: true            # k3s only; requires mails + ui
groups:
  mails:
    enabled: true
    replicas: 2         # k3s only
  files:
    enabled: false
  wsc:
    enabled: false
  ui:
    enabled: true
  todo:
    enabled: false      # preview, tasks, and videoserver
```

`./run.sh up` provisions `carbonio.localhost` and its development accounts when
needed, then restarts composed-ui so its proxy configuration sees the mailbox.
Run the smoke tests afterward with `./tests/run.sh`.
