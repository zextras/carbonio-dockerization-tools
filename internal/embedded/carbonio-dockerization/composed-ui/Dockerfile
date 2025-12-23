ARG CARBONIO_SHELL_UI_IMAGE
ARG CARBONIO_MAILS_UI_IMAGE
ARG CARBONIO_CONTACTS_UI_IMAGE
ARG CARBONIO_CALENDARS_UI_IMAGE
ARG CARBONIO_SEARCH_UI_IMAGE
ARG CARBONIO_FILES_UI_IMAGE
ARG CARBONIO_TASKS_UI_IMAGE
ARG CARBONIO_AUTH_UI_IMAGE
ARG CARBONIO_WSC_UI_IMAGE
ARG CARBONIO_LOGIN_UI_IMAGE
ARG CARBONIO_ADMIN_CONSOLE_UI_IMAGE
ARG CARBONIO_ADMIN_LOGIN_UI_IMAGE
ARG CARBONIO_PROXY_IMAGE
ARG CARBONIO_WEBUI_I18N_IMAGE

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

FROM ${CARBONIO_LOGIN_UI_IMAGE} AS login-ui

FROM ${CARBONIO_ADMIN_CONSOLE_UI_IMAGE} AS admin-console-ui

FROM ${CARBONIO_ADMIN_LOGIN_UI_IMAGE} AS admin-login-ui

FROM ${CARBONIO_WEBUI_I18N_IMAGE} AS webui-i18n

FROM ${CARBONIO_PROXY_IMAGE}

USER root

RUN apt update && apt install -y jq wget unzip && apt clean

# Install consul-template
RUN wget https://releases.hashicorp.com/consul-template/0.40.0/consul-template_0.40.0_linux_amd64.zip && \
    unzip consul-template_0.40.0_linux_amd64.zip -d /usr/local/bin && \
    rm consul-template_0.40.0_linux_amd64.zip

# webui
COPY --from=shell-ui /opt/zextras/web/iris/carbonio-shell-ui /opt/zextras/web/iris/carbonio-shell-ui
COPY --from=mails-ui /opt/zextras/web/iris/carbonio-mails-ui /opt/zextras/web/iris/carbonio-mails-ui
COPY --from=calendars-ui /opt/zextras/web/iris/carbonio-calendars-ui /opt/zextras/web/iris/carbonio-calendars-ui
COPY --from=contacts-ui /opt/zextras/web/iris/carbonio-contacts-ui /opt/zextras/web/iris/carbonio-contacts-ui
COPY --from=files-ui /opt/zextras/web/iris/carbonio-files-ui /opt/zextras/web/iris/carbonio-files-ui
COPY --from=tasks-ui /opt/zextras/web/iris/carbonio-tasks-ui /opt/zextras/web/iris/carbonio-tasks-ui
COPY --from=wsc-ui /opt/zextras/web/iris/carbonio-ws-collaboration-ui /opt/zextras/web/iris/carbonio-ws-collaboration-ui
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
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-tasks-ui/i18n /opt/zextras/web/iris/carbonio-tasks-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-mails-ui/i18n /opt/zextras/web/iris/carbonio-mails-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-search-ui/i18n /opt/zextras/web/iris/carbonio-search-ui/i18n 
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-shell-ui/i18n /opt/zextras/web/iris/carbonio-shell-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-tasks-ui/i18n /opt/zextras/web/iris/carbonio-tasks-ui/i18n
COPY --from=webui-i18n /opt/zextras/web/iris/carbonio-ws-collaboration-ui/i18n /opt/zextras/web/iris/carbonio-ws-collaboration-ui/i18n

# delete empty dirs
RUN find /opt/zextras/web/iris -type d -empty -delete 2>/dev/null || true && \
    find /opt/zextras/admin/iris -type d -empty -delete 2>/dev/null || true && \
    find /opt/zextras/web/login -type d -empty -delete 2>/dev/null || true && \
    find /opt/zextras/admin/login -type d -empty -delete 2>/dev/null || true

COPY entrypoint-proxy.sh ./entrypoint-proxy.sh

ENTRYPOINT ["./entrypoint-proxy.sh"]
