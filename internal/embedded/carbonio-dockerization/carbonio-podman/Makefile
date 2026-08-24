# Carbonio on k3s — one entry point for the dev lifecycle; `make help` lists all.
# Shared targets live here (chart, namespace, pull secret). The k3s lifecycle is
# per-platform: Makefile.linux (native k3s + systemd) or Makefile.darwin (k3d on
# a docker daemon), included from `uname -s` at the bottom of this file.

NAMESPACE ?= carbonio
RELEASE   ?= carbonio
CHART_DIR ?= chart

REGISTRY_SERVER ?= registry.dev.zextras.com
REGISTRY_SECRET ?= zextras-registry

# Same per-dev override run.sh already honours; passed last so it wins.
VALUES_LOCAL := $(wildcard $(CHART_DIR)/values.local.yaml)
HELM_VALUES  := $(if $(VALUES_LOCAL),-f $(VALUES_LOCAL))

.PHONY: help \
        namespace registry_secret \
        install_k3s_full install_portainer install_chartmuseum \
        carbonio_install carbonio_upgrade carbonio_uninstall carbonio_purge \
        carbonio_provision

help: platform_help
	@echo ""
	@echo "Cluster prep (any platform):"
	@echo "  make namespace           Create namespace $(NAMESPACE)"
	@echo "  make registry_secret     Create $(REGISTRY_SECRET) from REGISTRY_USER + REGISTRY_PASS"
	@echo "  make install_k3s_full    Bootstrap k3s + Portainer + ChartMuseum (requires Helm)"
	@echo "  make install_portainer   Portainer web UI (NodePort 30777/30779)"
	@echo "  make install_chartmuseum ChartMuseum Helm repo (NodePort 30808)"
	@echo ""
	@echo "Carbonio chart:"
	@echo "  make carbonio_install    helm install $(RELEASE) $(CHART_DIR) -n $(NAMESPACE) --timeout $(HELM_TIMEOUT)"
	@echo "  make carbonio_upgrade    helm upgrade $(RELEASE) $(CHART_DIR) -n $(NAMESPACE) --timeout $(HELM_TIMEOUT)"
	@echo "  make carbonio_uninstall  helm uninstall $(RELEASE), then wait for every pod to go"
	@echo "  make carbonio_purge      Same + delete the PVCs, so the next install provisions fresh"
	@echo "  make carbonio_provision  Manual provisioning fallback (./provision.sh)"
	@echo ""
	@echo "Override defaults with NAMESPACE=, RELEASE=, CHART_DIR=, HELM_TIMEOUT=."

# apply-from-dry-run keeps both targets idempotent (create alone fails if exists).
namespace:
	kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -

# Credentials come from the environment, or from the keychain on macOS.
# install_k3s.sh runs the same script, so there is one copy of this logic.
registry_secret:
	@NAMESPACE='$(NAMESPACE)' REGISTRY_SERVER='$(REGISTRY_SERVER)' \
	 REGISTRY_SECRET='$(REGISTRY_SECRET)' \
	 REGISTRY_USER='$(REGISTRY_USER)' REGISTRY_PASS='$(REGISTRY_PASS)' \
	 ./registry_secret.sh

# The full dev cluster: k3s (per-platform) plus both optional web tools.
install_k3s_full: install_k3s install_portainer install_chartmuseum

# NodePort, not LoadBalancer: servicelb is disabled, so an LB stays <pending>.
install_portainer:
	helm repo add portainer https://portainer.github.io/k8s/ 2>/dev/null || true
	helm repo update portainer
	helm upgrade --install portainer portainer/portainer \
	  -n portainer --create-namespace \
	  --set service.type=NodePort

# ChartMuseum only earns its keep if you install the chart from Portainer's Helm
# tab: Portainer CE speaks classic HTTP repos, not OCI. See k3s.md.
install_chartmuseum:
	helm repo add chartmuseum https://chartmuseum.github.io/charts 2>/dev/null || true
	helm repo update chartmuseum
	helm upgrade --install chartmuseum chartmuseum/chartmuseum \
	  -n chartmuseum --create-namespace \
	  --set env.open.DISABLE_API=false \
	  --set env.open.ALLOW_OVERWRITE=true \
	  --set persistence.enabled=true \
	  --set service.type=NodePort \
	  --set service.nodePort=30808

# Generous by design: the provision hook blocks helm until the mailbox is up,
# zmprov has run and (first install only) the mailbox has restarted so it
# registers with the proxy — on a cold cluster, plus every image pull.
HELM_TIMEOUT ?= 45m

carbonio_install:
	helm install $(RELEASE) $(CHART_DIR) $(HELM_VALUES) -n $(NAMESPACE) --timeout $(HELM_TIMEOUT)

carbonio_upgrade:
	helm upgrade $(RELEASE) $(CHART_DIR) $(HELM_VALUES) -n $(NAMESPACE) --timeout $(HELM_TIMEOUT)

# The drain is not optional: composed-ui and postfix are bare Pods, so a racing
# reinstall makes helm adopt the dying objects and no controller recreates them.
# The provision Job is a hook helm leaves behind on uninstall — delete it first
# or its Completed pod makes the drain wait forever.
carbonio_uninstall:
	helm uninstall $(RELEASE) -n $(NAMESPACE) --ignore-not-found
	kubectl -n $(NAMESPACE) delete job carbonio-provision --ignore-not-found
	kubectl -n $(NAMESPACE) wait --for=delete pod --all --timeout=180s

# Uninstall + drop the PVCs, so the next install provisions from scratch.
# Keeping them makes the provision hook skip ("Domain already exists"), and a
# reused openldap volume can serve an LDAP whose indexes no longer answer.
carbonio_purge: carbonio_uninstall
	kubectl -n $(NAMESPACE) delete pvc --all

# Fallback for --set provision.auto=false; otherwise the chart hook does this.
carbonio_provision:
	./provision.sh

# Defines platform_help + the install/stop/start/uninstall_k3s targets.
include Makefile.$(shell uname -s | tr '[:upper:]' '[:lower:]')
