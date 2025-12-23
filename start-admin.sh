#!/bin/bash

# Carbonio Minimal Admin Panel Startup Script
# Starts only essential services for admin console functionality

export COMPOSE_FILE="docker-compose.yaml"

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

# Parse command line arguments
if [[ "$1" == "--advanced" ]]; then
	export COMPOSE_FILE="docker-compose.yaml:docker-compose-advanced.yaml"
	echo "Starting Admin Panel ADVANCED"
	shift
else
	echo "Starting Admin Panel CE"
fi

# Add provisioning service unless explicitly disabled
if [[ "$1" == "--no-provision" ]]; then
	echo "Skipping provisioning service."
	shift
else
	ADMIN_SERVICES+=("${PROVISIONING_SERVICE[@]}")
fi

# Add any additional services passed as arguments
ADMIN_SERVICES+=("$@")

echo "=========================================="
echo "Starting Carbonio Admin Panel"
echo "=========================================="

# Start the minimal admin services
exec docker compose up -d --build --pull missing "${ADMIN_SERVICES[@]}"
