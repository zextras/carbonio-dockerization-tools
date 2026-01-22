#!/bin/bash
# Carbonio Minimal Admin Panel Startup Script
# Starts only essential services for admin console functionality
# Disable UI components
export CARBONIO_SHELL_UI_IMAGE=disabled
export CARBONIO_MAILS_UI_IMAGE=disabled
export CARBONIO_CONTACTS_UI_IMAGE=disabled
export CARBONIO_CALENDARS_UI_IMAGE=disabled
export CARBONIO_SEARCH_UI_IMAGE=disabled
export CARBONIO_FILES_UI_IMAGE=disabled
export CARBONIO_WSC_UI_IMAGE=disabled
export CARBONIO_LOGIN_UI_IMAGE=disabled
# Minimal admin panel services
ADMIN_SERVICES=(
	consul
	traefik
	memcached
	carbonio-openldap
	carbonio-postfix
	carbonio-mariadb
	carbonio-postgres
	carbonio-mailbox
	carbonio-user-management
	carbonio-catalog
	event-listener
	consul-register
	carbonio-composed-ui
)
# Provisioning service (runs once for initial setup)
PROVISIONING_SERVICE=(carbonio-provisioner)
# Default values
PULL_POLICY="always"
ADVANCED=false
NO_PROVISION=false
EXTRA_SERVICES=()
# Parse command line arguments
while [[ $# -gt 0 ]]; do
	case "$1" in
	--advanced)
		ADVANCED=true
		shift
		;;
	--no-provision)
		NO_PROVISION=true
		shift
		;;
	--missing)
		PULL_POLICY="missing"
		shift
		;;
	*)
		EXTRA_SERVICES+=("$1")
		shift
		;;
	esac
done

# Determine OS-specific override
if [[ "$(uname)" == "Darwin" ]]; then
	OS_OVERRIDE=":docker-compose.macos.override.yaml"
else
	OS_OVERRIDE=""
fi

if [[ "$ADVANCED" == true ]]; then
	export COMPOSE_FILE="docker-compose.yaml:docker-compose-advanced.yaml${OS_OVERRIDE}"
else
	export COMPOSE_FILE="docker-compose.yaml${OS_OVERRIDE}"
fi
if [[ "$NO_PROVISION" == true ]]; then
	echo "Skipping provisioning service."
else
	ADMIN_SERVICES+=("${PROVISIONING_SERVICE[@]}")
fi
# Add any additional services passed as arguments
ADMIN_SERVICES+=("${EXTRA_SERVICES[@]}")

echo "=========================================="
echo "Stopping services"
echo "=========================================="
docker compose down -v -t 0 --remove-orphans

echo "=========================================="
if [[ "$ADVANCED" == true ]]; then
	echo "Starting Carbonio Admin Panel ADVANCED"
else
	echo "Starting Carbonio Admin Panel CE"
fi
echo "=========================================="
exec docker compose up -d --build --pull "$PULL_POLICY" "${ADMIN_SERVICES[@]}"
