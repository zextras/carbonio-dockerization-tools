{{/*
Returns "true" if any service group is enabled. Used to gate the shared core
infra (consul-server, openldap, postgres, monitoring) that every group needs.
*/}}
{{- define "carbonio.coreEnabled" -}}
{{- or .Values.groups.mails.enabled .Values.groups.files.enabled .Values.groups.wsc.enabled .Values.groups.ui.enabled .Values.groups.todo.enabled -}}
{{- end -}}

{{/*
Mailbox pod containers (shared between the k3s StatefulSet and the podman Pod).
The caller is responsible for the surrounding `containers:` key and indentation
(use `include "carbonio.mailbox.containers" . | indent N`).

Images come from .Values.images and CE/non-CE sidecars are gated on
.Values.edition, matching the inline pattern used by the other templates.
*/}}
{{- define "carbonio.mailbox.containers" -}}
- name: app
{{- if eq .Values.edition "ce" }}
  image: {{ required "images.carbonio-mailbox is required" (index .Values.images "carbonio-mailbox") }}
{{- else }}
  image: {{ required "images.carbonio-advanced is required" (index .Values.images "carbonio-advanced") }}
{{- end }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  ports:
    - containerPort: 8080
{{- if ne .Values.platform "k3s" }}
      hostPort: 8080
{{- end }}
    - containerPort: 7071
{{- if ne .Values.platform "k3s" }}
      hostPort: 7071
{{- end }}
  env:
    # LDAP is outside the mesh; mariadb is in-pod (localhost)
    - name: MAILBOXD_JAVA_OPTS
      # ttl=0 so mailboxd re-resolves k8s names (incl. carbonio-composed-ui for
      # getTrustedIPs) instead of caching the startup failure permanently.
      value: "-Xss256k -Xms128m -Xmx1024m -Dsun.net.inetaddr.ttl=0 -Dsun.net.inetaddr.negative.ttl=0"
    - name: LDAP_URL
      value: "ldap://carbonio-openldap:1389"
    - name: LDAP_ROOT_PASSWORD
      value: "qh6hWZvc"
    - name: LDAP_ADMIN_PASSWORD
      value: "password"
    - name: MARIADB_ROOT_PASSWORD
      value: "password"
    - name: MARIADB_URL
      value: "localhost"
    - name: MARIADB_PORT
      value: "3306"
    - name: CARBONIO_FULL_QUOTA_CHECK_ENABLED
      value: "true"
    - name: JAVA_TOOL_OPTIONS
      value: "-javaagent:/opt/zextras/opentelemetry-javaagent.jar"
    - name: OTEL_LOGS_EXPORTER
      value: "otlp"
    - name: OTEL_EXPORTER_OTLP_ENDPOINT
      value: "http://carbonio-monitoring:4318"
    - name: OTEL_METRICS_EXPORTER
      value: "otlp"
    - name: OTEL_TRACES_EXPORTER
      value: "otlp"
{{- if not .Values.monitoring.enabled }}
    - name: OTEL_SDK_DISABLED
      value: "true"
{{- end }}
  volumeMounts:
    - name: mailbox-data
      mountPath: /opt/zextras/data/mailbox
    - name: mailbox-index
      mountPath: /opt/zextras/index
    - name: mailbox-store
      mountPath: /opt/zextras/store
    - name: token-mailbox
      mountPath: /etc/carbonio/mailbox/service-discover
    - name: token-mailbox-admin
      mountPath: /etc/carbonio/mailbox-admin/service-discover
    - name: token-mailbox-nslookup
      mountPath: /etc/carbonio/mailbox-nslookup/service-discover
    - name: token-mailbox-internal-api
      mountPath: /etc/carbonio/mailbox-internal-api/service-discover
{{- if ne .Values.edition "ce" }}
    - name: token-advanced
      mountPath: /etc/carbonio/advanced/service-discover
{{- end }}
{{- if ne .Values.edition "ce" }}
    - name: token-address-book
      mountPath: /etc/carbonio/address-book/service-discover
{{- end }}
{{- if ne .Values.edition "ce" }}
    - name: token-auth
      mountPath: /etc/carbonio/auth/service-discover
{{- end }}

- name: mariadb
  image: {{ required "images.carbonio-mariadb is required" (index .Values.images "carbonio-mariadb") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  volumeMounts:
    - name: mariadb-data
      mountPath: /var/lib/mysql

- name: consul-agent
  image: {{ required "images.consul is required" (index .Values.images "consul") }}
  args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
  volumeMounts:
    - name: consul-agent-acl
      mountPath: /consul/acl/consul-agent-acl.hcl
      subPath: consul-agent-acl.hcl
      readOnly: true

- name: sidecar-mailbox
  image: {{ required "images.carbonio-mailbox-sidecar is required" (index .Values.images "carbonio-mailbox-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: token-mailbox
      mountPath: /shared-tokens

- name: sidecar-admin
  image: {{ required "images.carbonio-mailbox-admin-sidecar is required" (index .Values.images "carbonio-mailbox-admin-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: token-mailbox-admin
      mountPath: /shared-tokens

- name: sidecar-nslookup
  image: {{ required "images.carbonio-mailbox-nslookup-sidecar is required" (index .Values.images "carbonio-mailbox-nslookup-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: token-mailbox-nslookup
      mountPath: /shared-tokens

- name: sidecar-internal-api
  image: {{ required "images.carbonio-mailbox-internal-api-sidecar is required" (index .Values.images "carbonio-mailbox-internal-api-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: token-mailbox-internal-api
      mountPath: /shared-tokens

{{- if ne .Values.edition "ce" }}
- name: sidecar-advanced
  image: {{ required "images.carbonio-advanced-sidecar is required" (index .Values.images "carbonio-advanced-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: token-advanced
      mountPath: /shared-tokens
{{- end }}

{{- if ne .Values.edition "ce" }}
- name: sidecar-address-book
  image: {{ required "images.carbonio-address-book-sidecar is required" (index .Values.images "carbonio-address-book-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: token-address-book
      mountPath: /shared-tokens
{{- end }}

{{- if ne .Values.edition "ce" }}
- name: sidecar-auth
  image: {{ required "images.carbonio-auth-sidecar is required" (index .Values.images "carbonio-auth-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: token-auth
      mountPath: /shared-tokens
{{- end }}
{{- end -}}

{{/*
Mailbox in-pod (non-PVC) volumes — shared between StatefulSet and Pod branches.
The caller renders `volumes:` and includes this with `indent N`. The PVC-backed
volumes are supplied separately (volumeClaimTemplates on k3s, claimName volumes
on podman).
*/}}
{{- define "carbonio.mailbox.sharedVolumes" -}}
# Each token dir is shared between the app and its sidecar within this pod
# (sidecar writes the consul token, app reads it) — emptyDir replaces the
# ./tokens hostPath used under podman.
- name: token-mailbox
  emptyDir: {}
- name: token-mailbox-admin
  emptyDir: {}
- name: token-mailbox-nslookup
  emptyDir: {}
- name: token-mailbox-internal-api
  emptyDir: {}
{{- if ne .Values.edition "ce" }}
- name: token-advanced
  emptyDir: {}
{{- end }}
{{- if ne .Values.edition "ce" }}
- name: token-address-book
  emptyDir: {}
{{- end }}
{{- if ne .Values.edition "ce" }}
- name: token-auth
  emptyDir: {}
{{- end }}
- name: consul-agent-acl
  configMap:
    name: consul-acl
    items:
      - key: consul-agent-acl.hcl
        path: consul-agent-acl.hcl
{{- end -}}