# Carbonio on k3s — one entry point for the dev lifecycle.
# Run `make help` to see all targets.

NAMESPACE ?= carbonio
RELEASE   ?= carbonio
CHART_DIR ?= chart

.PHONY: help \
        install_k3s install_k3s_full stop_k3s start_k3s uninstall_k3s \
        carbonio_install carbonio_upgrade carbonio_uninstall \
        carbonio_provision

help:
	@echo "k3s lifecycle:"
	@echo "  make install_k3s         Bootstrap k3s + namespace; secret requires REGISTRY_USER + REGISTRY_PASS"
	@echo "  make install_k3s_full    Same + Portainer + ChartMuseum (requires Helm)"
	@echo "  make stop_k3s            Stop k3s and ALL pod containers (k3s-killall.sh)"
	@echo "  make start_k3s           Start k3s (systemctl start k3s)"
	@echo "  make uninstall_k3s       Wipe k3s entirely (k3s-uninstall.sh)"
	@echo ""
	@echo "Carbonio chart (k3s):"
	@echo "  make carbonio_install    helm install $(RELEASE) $(CHART_DIR) -n $(NAMESPACE) --timeout $(HELM_TIMEOUT)"
	@echo "  make carbonio_upgrade    helm upgrade $(RELEASE) $(CHART_DIR) -n $(NAMESPACE) --timeout $(HELM_TIMEOUT)"
	@echo "  make carbonio_uninstall  helm uninstall $(RELEASE) -n $(NAMESPACE)"
	@echo "  make carbonio_provision  Manual provisioning fallback (./provision.sh)"
	@echo ""
	@echo "Override defaults with NAMESPACE=, RELEASE=, CHART_DIR=, HELM_TIMEOUT=."

install_k3s:
	./install_k3s.sh

install_k3s_full:
	./install_k3s.sh --with-portainer --with-chartmuseum

stop_k3s:
	sudo /usr/local/bin/k3s-killall.sh

start_k3s:
	sudo systemctl start k3s

uninstall_k3s:
	sudo /usr/local/bin/k3s-uninstall.sh

# --timeout is generous: the post-install/upgrade provision hook blocks helm
# until the mailbox is up, zmprov has run, and (on first provisioning) the
# mailbox has been restarted so it registers correctly for the proxy. On a cold
# cluster that includes image pulls + two ordered mailbox boots. Raise
# HELM_TIMEOUT if yours is slower.
HELM_TIMEOUT ?= 45m

carbonio_install:
	helm install $(RELEASE) $(CHART_DIR) -n $(NAMESPACE) --timeout $(HELM_TIMEOUT)

carbonio_upgrade:
	helm upgrade $(RELEASE) $(CHART_DIR) -n $(NAMESPACE) --timeout $(HELM_TIMEOUT)

carbonio_uninstall:
	helm uninstall $(RELEASE) -n $(NAMESPACE)

carbonio_provision:
	./provision.sh
