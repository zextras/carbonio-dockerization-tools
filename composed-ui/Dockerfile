# Defaults target the devel tag so CI (which passes no build args) composes the
# devel UI bundle. run.sh still overrides these via --build-arg for local builds.
ARG CARBONIO_SHELL_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-shell-ui:devel
ARG CARBONIO_MAILS_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-mails-ui:devel
ARG CARBONIO_CONTACTS_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-contacts-ui:devel
ARG CARBONIO_CALENDARS_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-calendars-ui:devel
ARG CARBONIO_SEARCH_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-search-ui:devel
ARG CARBONIO_FILES_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-files-ui:devel
ARG CARBONIO_TASKS_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-tasks-ui:devel
ARG CARBONIO_AUTH_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-auth-ui:devel
ARG CARBONIO_WSC_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-ws-collaboration-ui:devel
ARG CARBONIO_STORAGES_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-storages-ui:devel
ARG CARBONIO_LOGIN_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-login-ui:devel
ARG CARBONIO_ADMIN_CONSOLE_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-admin-console-ui:devel
ARG CARBONIO_ADMIN_LOGIN_UI_IMAGE=registry.dev.zextras.com/dev/carbonio-admin-login-ui:devel
ARG CARBONIO_PROXY_IMAGE=registry.dev.zextras.com/dev/carbonio-proxy:devel
ARG CARBONIO_WEBUI_I18N_IMAGE=registry.dev.zextras.com/dev/carbonio-webui-i18n:devel

# not really proud of this solution but it's very easy: if we receive UI envs set to "disabled" this empty image will
# be used, so the copy does not crash (but copies empty directories) and then we delete them.
FROM busybox:latest AS disabled
RUN mkdir -p /opt/zextras/web/iris/carbonio-shell-ui \
             /opt/zextras/web/iris/carbonio-mails-ui \
             /opt/zextras/web/iris/carbonio-contacts-ui \
             /opt/zextras/web/iris/carbonio-calendars-ui \
             /opt/zextras/web/iris/carbonio-search-ui \
             /opt/zextras/web/iris/carbonio-files-ui \
             /opt/zextras/web/iris/carbonio-tasks-ui \
             /opt/zextras/web/iris/carbonio-auth-ui \
             /opt/zextras/web/iris/carbonio-ws-collaboration-ui \
             /opt/zextras/web/iris/carbonio-storages-ui \
             /opt/zextras/web/login \
             /opt/zextras/admin/iris/carbonio-admin-console-ui \
             /opt/zextras/admin/login

FROM ${CARBONIO_SHELL_UI_IMAGE} AS shell-ui

FROM ${CARBONIO_MAILS_UI_IMAGE} AS mails-ui

FROM ${CARBONIO_CONTACTS_UI_IMAGE} AS contacts-ui

FROM ${CARBONIO_CALENDARS_UI_IMAGE} AS calendars-ui

FROM ${CARBONIO_AUTH_UI_IMAGE} AS auth-ui

FROM ${CARBONIO_SEARCH_UI_IMAGE} AS search-ui

FROM ${CARBONIO_FILES_UI_IMAGE} AS files-ui

FROM ${CARBONIO_TASKS_UI_IMAGE} AS tasks-ui

FROM ${CARBONIO_WSC_UI_IMAGE} AS wsc-ui

FROM ${CARBONIO_STORAGES_UI_IMAGE} AS storages-ui

FROM ${CARBONIO_LOGIN_UI_IMAGE} AS login-ui

FROM ${CARBONIO_ADMIN_CONSOLE_UI_IMAGE} AS admin-console-ui

FROM ${CARBONIO_ADMIN_LOGIN_UI_IMAGE} AS admin-login-ui

FROM ${CARBONIO_WEBUI_I18N_IMAGE} AS webui-i18n

FROM ${CARBONIO_PROXY_IMAGE}

RUN apt update && apt install jq -y

USER root

# webui
COPY --from=shell-ui /opt/zextras/web/iris/carbonio-shell-ui /opt/zextras/web/iris/carbonio-shell-ui
COPY --from=mails-ui /opt/zextras/web/iris/carbonio-mails-ui /opt/zextras/web/iris/carbonio-mails-ui
COPY --from=calendars-ui /opt/zextras/web/iris/carbonio-calendars-ui /opt/zextras/web/iris/carbonio-calendars-ui
COPY --from=contacts-ui /opt/zextras/web/iris/carbonio-contacts-ui /opt/zextras/web/iris/carbonio-contacts-ui
COPY --from=files-ui /opt/zextras/web/iris/carbonio-files-ui /opt/zextras/web/iris/carbonio-files-ui
COPY --from=tasks-ui /opt/zextras/web/iris/carbonio-tasks-ui /opt/zextras/web/iris/carbonio-tasks-ui
COPY --from=wsc-ui /opt/zextras/web/iris/carbonio-ws-collaboration-ui /opt/zextras/web/iris/carbonio-ws-collaboration-ui
COPY --from=storages-ui /opt/zextras/web/iris/carbonio-storages-ui /opt/zextras/web/iris/carbonio-storages-ui
COPY --from=search-ui /opt/zextras/web/iris/carbonio-search-ui /opt/zextras/web/iris/carbonio-search-ui
COPY --from=auth-ui /opt/zextras/web/iris/carbonio-auth-ui /opt/zextras/web/iris/carbonio-auth-ui
# web login
COPY --from=login-ui /opt/zextras/web/login /opt/zextras/web/login

# admin
COPY --from=admin-console-ui /opt/zextras/admin/iris/ /opt/zextras/admin/iris/

# admin login
COPY --from=admin-login-ui /opt/zextras/admin/login /opt/zextras/admin/login

# translations
COPY --from=webui-i18n /opt/zextras/admin/login-i18n /opt/zextras/admin/login-i18n
COPY --from=webui-i18n /opt/zextras/admin/iris/i18n /opt/zextras/admin/iris/i18n
COPY --from=webui-i18n /opt/zextras/web/login-i18n /opt/zextras/web/login-i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-auth-ui/i18n /opt/zextras/web/iris/carbonio-auth-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-calendars-ui/i18n /opt/zextras/web/iris/carbonio-calendars-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-contacts-ui/i18n /opt/zextras/web/iris/carbonio-contacts-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-files-ui/i18n /opt/zextras/web/iris/carbonio-files-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-mails-ui/i18n /opt/zextras/web/iris/carbonio-mails-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-search-ui/i18n /opt/zextras/web/iris/carbonio-search-ui/i18n 
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-shell-ui/i18n /opt/zextras/web/iris/carbonio-shell-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-storages-ui/i18n /opt/zextras/web/iris/carbonio-storages-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-tasks-ui/i18n /opt/zextras/web/iris/carbonio-tasks-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-ws-collaboration-ui/i18n /opt/zextras/web/iris/carbonio-ws-collaboration-ui/i18n

# Each UI module serves assets from a path that includes its commit hash, e.g.:
#   /opt/zextras/web/iris/carbonio-files-ui/<COMMIT_ID>/i18n
# The COPY above places i18n at the module root, so we relocate it
# into the correct <COMMIT_ID> subdirectory, reading the hash from component.json.
RUN for module_dir in /opt/zextras/web/iris/carbonio-*/; do \
      component_json=$(find "$module_dir" -maxdepth 2 -name component.json | head -1); \
      if [ -z "$component_json" ] || [ ! -d "$module_dir/i18n" ]; then \
        continue; \
      fi; \
      commit_id=$(jq -r '.commit' "$component_json"); \
      ln -s ../i18n "$module_dir/$commit_id/i18n"; \
    done

RUN ln -sf /opt/zextras/admin/iris/i18n /opt/zextras/admin/iris/carbonio-admin-ui/i18n
RUN ln -sf /opt/zextras/.version /opt/zextras/admin/iris/carbonio-admin-ui/.version

# delete empty dirs
RUN find /opt/zextras/web/iris -type d -empty -delete 2>/dev/null || true && \
    find /opt/zextras/admin/iris -type d -empty -delete 2>/dev/null || true && \
    find /opt/zextras/web/login -type d -empty -delete 2>/dev/null || true && \
    find /opt/zextras/admin/login -type d -empty -delete 2>/dev/null || true

COPY entrypoint-proxy.sh ./entrypoint-proxy.sh

ENTRYPOINT ["./entrypoint-proxy.sh"]
