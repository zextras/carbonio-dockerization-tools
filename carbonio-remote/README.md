# Connecting Local Dev to a Remote VM

Run a Carbonio service locally and join it to the Consul mesh of a remote VM.

```
+---------------------------+              +-------------------------------+
|  Your machine (podman)    |    VPN       |  Remote VM                    |
|                           |              |                               |
|  carbonio-catalog (app)   |              |  consul-server                |
|  consul-agent  -----------|-- gossip ----|----> port 8301                |
|  envoy sidecar            |-- RPC+TLS ---|----> port 8300                |
+---------------------------+              +-------------------------------+
```

## Prerequisites

- 🔐 **VPN access** to the remote VM
- 🐳 **Podman** (ldapsearch runs in a container)
- 🔑 **Mesh GPG passphrase** (the service-discover secret)

## Setup

Run only once to fetch mesh credentials from LDAP, decrypt them, and generate 
TLS certs:

```bash
MESH_PASSWORD=<gpg-passphrase> REMOTE_HOST=<vm-hostname> ./setup.sh
```

This creates `remote-config/` with everything needed. The consul token is 
printed at the end along with the script command.

## Run

> **Note:** This is an example for the `carbonio-catalog`
> service (see [`templates/carbonio-catalog-remote.yaml`](templates/carbonio-catalog-remote.yaml) and
> [`run.sh`](run.sh) for the implementation). \
> The idea can be adapted to any other service.

Now:
- 🌐 **Find your VPN IP**: `ip route get <remote-vm-ip>` (look for `src`)
- 🚀 **Start**: `CONSUL_TOKEN=<secret-id> ADVERTISE_ADDR=<your-vpn-ip> REMOTE_HOST=<vm-hostname> ./run.sh up`
- 📊 **Status**: `./run.sh status`
- 📋 **Logs**: `./run.sh logs [container]`
- 🛑 **Stop**: `./run.sh down`

| Variable | Required | Default | Description |
|---|---|---|---|
| `MESH_PASSWORD` | setup | - | Service-discover GPG passphrase |
| `CONSUL_TOKEN` | run | - | Printed by `setup.sh` |
| `ADVERTISE_ADDR` | run | - | Your VPN IP (`ip route get <remote-vm-ip>`) |
| `REMOTE_HOST` | no | `kc-dev4-u22-ce.demo.zextras.io` | Remote VM hostname |
| `REMOTE_USER` | no | `root` | SSH user for the remote VM |

## 💡 Verify

1. 🔽 VM -> your PC:
From the remote VM, hit the catalog upstream through the proxy:

```bash
curl http://127.78.0.1:20001
```
You should see request logs on your local container with `./run.sh logs app`.

2. 🔼 PC -> VM:
`podman exec -it carbonio-catalog-test-sidecar bash`
Then:
`curl localhost:20000` -> you should hit the mailbox.
```
oot@carbonio-catalog:/# curl localhost:20000/service/soap
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">
<soap:Body><soap:Fault><soap:Code><soap:Value>soap:Sender</soap:Value></soap:Code><soap:Reason><soap:Text>parse error: empty request payload</soap:Text></soap:Reason><soap:Detail><Error xmlns="urn:zimbra"><Code>service.PARSE_ERROR</Code><Trace>qtp1730399463-468556:1774601302847:215983e0bca1725e</Trace></Error></soap:Detail></soap:Fault>
</soap:Body></soap:Envelope>
```

## Troubleshooting

| Error | Fix |
|---|---|
| `Remote state is encrypted` | Wrong gossip key -- re-run `setup.sh` |
| `tls: certificate required` | Client cert missing -- re-run `setup.sh` |
| `rpc error making call: EOF` | Wrong `ADVERTISE_ADDR` (must be your IP, not the remote) |
| `Node name X is reserved` | Stale node: `curl -X PUT http://localhost:8500/v1/catalog/deregister -H "X-Consul-Token: <token>" -d '{"Node":"carbonio-catalog"}'` |
