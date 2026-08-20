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
    - containerPort: 5005
      hostPort: 5005
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
{{- if .Values.monitoring.enabled }}
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
{{- end }}
{{- if ne .Values.edition "ce" }}
  # The Consul checks the advanced image ships (/zx/auth/health/live, TCP 8742)
  # pass as soon as Jetty binds, so they never catch a module that failed to
  # start (ModuleStarter disables it and leaves the JVM up). /health/ready does
  # a SELECT 1 on the auth DB, so it does. Covers the auth module only.
  # On podman this only marks the container unhealthy: play kube maps
  # livenessProbe to a healthcheck with health-on-failure=none.
  livenessProbe:
    httpGet:
      path: /zx/auth/health/ready
      port: 8742
    initialDelaySeconds: 300
    periodSeconds: 15
    timeoutSeconds: 5
    failureThreshold: 4
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
    - name: token-license
      mountPath: /etc/carbonio/license/service-discover
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
{{- if eq .Values.platform "podman" }}
    - name: mailbox-consul-data
      mountPath: /consul/data
{{- end }}

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
- name: sidecar-license
  image: {{ required "images.carbonio-license-sidecar is required" (index .Values.images "carbonio-license-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: token-license
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
- name: token-license
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
{{- if eq .Values.platform "podman" }}
- name: mailbox-consul-data
  persistentVolumeClaim:
    claimName: mailbox-consul-data
{{- end }}
{{- end -}}

{{/*
--- Stateless service pod bodies (k3s Deployment / podman Pod) --------------
Each helper renders a pod-spec body (containers, volumes, and any pod-level
fields) at column 0. Callers wrap it: on k3s inside a Deployment's
template.spec (indent 6), on podman inside a bare Pod's spec (indent 2). The
k3s-only enableServiceLinks/imagePullSecrets live in the Deployment wrapper
(see "carbonio.deployment.k3sPodDefaults"), so they are absent here.
*/}}

{{- define "carbonio.catalog.podbody" -}}
containers:
  - name: app
    image: {{ required "images.carbonio-catalog is required" (index .Values.images "carbonio-catalog") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    volumeMounts:
      - name: catalog-token
        mountPath: /etc/carbonio/catalog/service-discover
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: catalog-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-catalog-sidecar is required" (index .Values.images "carbonio-catalog-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: catalog-token
        mountPath: /shared-tokens
volumes:
  - name: catalog-token
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: catalog-consul-data
    persistentVolumeClaim:
      claimName: catalog-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.user-management.podbody" -}}
containers:
  - name: app
    image: {{ required "images.carbonio-user-management is required" (index .Values.images "carbonio-user-management") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: QUARKUS_OTEL_SDK_DISABLED
        value: "{{ not .Values.monitoring.enabled }}"
      - name: QUARKUS_OTEL_EXPORTER_OTLP_ENDPOINT
        value: "http://carbonio-monitoring:4317"
    volumeMounts:
      - name: um-token
        mountPath: /etc/carbonio/user-management/service-discover
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: user-management-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-user-management-sidecar is required" (index .Values.images "carbonio-user-management-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: um-token
        mountPath: /shared-tokens
volumes:
  - name: um-token
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: user-management-consul-data
    persistentVolumeClaim:
      claimName: user-management-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.license-service.podbody" -}}
containers:
  - name: app
    image: {{ required "images.carbonio-license-service is required" (index .Values.images "carbonio-license-service") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      # The datasource credentials for the `subscription` database are provisioned
      # by the mailbox-db bootstrap under its own KV prefix, not the service's.
      - name: SUBSCRIPTION_DB_CONSUL_SERVICE
        value: "carbonio-subscription"
    volumeMounts:
      - name: license-service-token
        mountPath: /etc/carbonio/license-service/service-discover
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: license-service-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-license-service-sidecar is required" (index .Values.images "carbonio-license-service-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: license-service-token
        mountPath: /shared-tokens
volumes:
  - name: license-service-token
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: license-service-consul-data
    persistentVolumeClaim:
      claimName: license-service-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.monitoring.podbody" -}}
containers:
  - name: lgtm
    image: {{ required "images.otel-lgtm is required" (index .Values.images "otel-lgtm") }}
    ports:
      - containerPort: 3000
        hostPort: 3000
      - containerPort: 4317
        hostPort: 4317
      - containerPort: 4318
        hostPort: 4318
{{- end -}}

{{- define "carbonio.message-broker.podbody" -}}
# RabbitMQ runs as the 'rabbitmq' user (uid 100/gid 101) and must own its
# Erlang cookie under /var/lib/rabbitmq. Without a dedicated writable volume
# the cookie can become unreadable across restarts (eacces), failing Erlang
# distribution startup. fsGroup makes the mounted dir group-owned by rabbitmq.
# Mirrors docker-compose's carbonio-message-broker_data volume.
securityContext:
  fsGroup: 101
containers:
  - name: app
    image: {{ required "images.carbonio-message-broker is required" (index .Values.images "carbonio-message-broker") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    ports:
      - containerPort: 10000
      - containerPort: 10001
    env:
      - name: RABBITMQ_NODENAME
        value: "carbonio-message-broker-clustered"
      - name: CONSUL_SETUP
        value: "true"
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: message-broker-token
        mountPath: /etc/carbonio/message-broker/service-discover
      - name: message-broker-data
        mountPath: /var/lib/rabbitmq
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: message-broker-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-message-broker-sidecar is required" (index .Values.images "carbonio-message-broker-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: message-broker-token
        mountPath: /shared-tokens
volumes:
  - name: message-broker-token
    emptyDir: {}
  - name: message-broker-data
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: message-broker-consul-data
    persistentVolumeClaim:
      claimName: message-broker-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.files.podbody" -}}
containers:
  - name: app
    image: {{ required "images.carbonio-files is required" (index .Values.images "carbonio-files") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: JAVA_TOOL_OPTIONS
{{- if .Values.monitoring.enabled }}
        value: "-javaagent:/opt/zextras/opentelemetry-javaagent.jar -Dsun.net.inetaddr.ttl=0"
{{- else }}
        value: "-Dsun.net.inetaddr.ttl=0"
{{- end }}
      - name: _JAVA_OPTIONS
        value: "-Xms128m -Xmx512m"
{{- if .Values.monitoring.enabled }}
      - name: OTEL_LOGS_EXPORTER
        value: "otlp"
      - name: OTEL_EXPORTER_OTLP_ENDPOINT
        value: "http://carbonio-monitoring:4318"
      - name: OTEL_METRICS_EXPORTER
        value: "otlp"
      - name: OTEL_TRACES_EXPORTER
        value: "otlp"
{{- end }}
    command: ["/bin/sh", "-c"]
    args:
      - |
        while [ ! -f /etc/carbonio/files/service-discover/token ]; do sleep 1; done
        export CONSUL_HTTP_TOKEN=$(cat /etc/carbonio/files/service-discover/token)
        exec /app/entrypoint.sh
    volumeMounts:
      - name: files-token
        mountPath: /etc/carbonio/files/service-discover
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: files-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-files-sidecar is required" (index .Values.images "carbonio-files-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: files-token
        mountPath: /shared-tokens
volumes:
  - name: files-token
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: files-consul-data
    persistentVolumeClaim:
      claimName: files-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.docs-connector.podbody" -}}
containers:
  - name: app
    image: {{ required "images.carbonio-docs-connector is required" (index .Values.images "carbonio-docs-connector") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    command: ["/bin/sh", "-c"]
    args:
      - |
        while [ ! -f /etc/carbonio/docs-connector/service-discover/token ]; do sleep 1; done
        export CONSUL_HTTP_TOKEN=$(cat /etc/carbonio/docs-connector/service-discover/token)
        exec /app/entrypoint.sh
    volumeMounts:
      - name: docs-connector-token
        mountPath: /etc/carbonio/docs-connector/service-discover
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: docs-connector-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-docs-connector-sidecar is required" (index .Values.images "carbonio-docs-connector-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: docs-connector-token
        mountPath: /shared-tokens
volumes:
  - name: docs-connector-token
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: docs-connector-consul-data
    persistentVolumeClaim:
      claimName: docs-connector-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.docs-editor.podbody" -}}
containers:
  - name: app
    image: {{ required "images.carbonio-docs-editor is required" (index .Values.images "carbonio-docs-editor") }}
    securityContext:
      capabilities:
        add: [MKNOD, SYS_ADMIN, FOWNER, CHOWN, SYS_CHROOT]
    volumeMounts:
      - name: docs-editor-token
        mountPath: /etc/carbonio/docs-editor/service-discover
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: docs-editor-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-docs-editor-sidecar is required" (index .Values.images "carbonio-docs-editor-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: docs-editor-token
        mountPath: /shared-tokens
volumes:
  - name: docs-editor-token
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: docs-editor-consul-data
    persistentVolumeClaim:
      claimName: docs-editor-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.ws-collaboration.podbody" -}}
restartPolicy: Always
containers:
  - name: app
    image: {{ required "images.carbonio-ws-collaboration is required" (index .Values.images "carbonio-ws-collaboration") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: JAVA_TOOL_OPTIONS
{{- if .Values.monitoring.enabled }}
        value: "-javaagent:/opt/zextras/opentelemetry-javaagent.jar -Dsun.net.inetaddr.ttl=0"
{{- else }}
        value: "-Dsun.net.inetaddr.ttl=0"
{{- end }}
      - name: _JAVA_OPTIONS
        value: "-Xms128m -Xmx512m"
{{- if .Values.monitoring.enabled }}
      - name: OTEL_LOGS_EXPORTER
        value: "otlp"
      - name: OTEL_EXPORTER_OTLP_ENDPOINT
        value: "http://carbonio-monitoring:4318"
      - name: OTEL_METRICS_EXPORTER
        value: "otlp"
      - name: OTEL_TRACES_EXPORTER
        value: "otlp"
{{- end }}
    command: [ "/busybox/sh", "-c" ]
    args:
      - |
        while [ ! -f /etc/carbonio/ws-collaboration/service-discover/token ]; do sleep 1; done
        export CONSUL_HTTP_TOKEN=$(cat /etc/carbonio/ws-collaboration/service-discover/token)
        exec java -jar /app/app.jar
    volumeMounts:
      - name: ws-collaboration-token
        mountPath: /etc/carbonio/ws-collaboration/service-discover
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: ws-collaboration-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-ws-collaboration-sidecar is required" (index .Values.images "carbonio-ws-collaboration-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: ws-collaboration-token
        mountPath: /shared-tokens
volumes:
  - name: ws-collaboration-token
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: ws-collaboration-consul-data
    persistentVolumeClaim:
      claimName: ws-collaboration-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.message-dispatcher.podbody" -}}
restartPolicy: Always
containers:
  - name: auth
    image: {{ required "images.carbonio-message-dispatcher-auth is required" (index .Values.images "carbonio-message-dispatcher-auth") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
{{- if .Values.monitoring.enabled }}
      - name: JAVA_TOOL_OPTIONS
        value: "-javaagent:/opt/zextras/opentelemetry-javaagent.jar"
{{- end }}
      - name: _JAVA_OPTIONS
        value: "-Xms128m -Xmx512m"
{{- if .Values.monitoring.enabled }}
      - name: OTEL_LOGS_EXPORTER
        value: "otlp"
      - name: OTEL_EXPORTER_OTLP_ENDPOINT
        value: "http://carbonio-monitoring:4318"
      - name: OTEL_METRICS_EXPORTER
        value: "otlp"
      - name: OTEL_TRACES_EXPORTER
        value: "otlp"
{{- end }}
    command: ["/busybox/sh", "-c"]
    args:
      - |
        while [ ! -f /etc/carbonio/message-dispatcher/service-discover/token ]; do sleep 1; done
        export CONSUL_HTTP_TOKEN=$(cat /etc/carbonio/message-dispatcher/service-discover/token)
        exec java -Djava.net.preferIPv4Stack=true \
          -jar /app/carbonio-message-dispatcher-auth.jar
    volumeMounts:
      - name: token-message-dispatcher
        mountPath: /etc/carbonio/message-dispatcher/service-discover
  - name: mongoose
    image: {{ required "images.carbonio-message-dispatcher-mongoose is required" (index .Values.images "carbonio-message-dispatcher-mongoose") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: token-message-dispatcher
        mountPath: /etc/carbonio/message-dispatcher/service-discover
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: message-dispatcher-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar-auth
    image: {{ required "images.carbonio-message-dispatcher-auth-sidecar is required" (index .Values.images "carbonio-message-dispatcher-auth-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: token-message-dispatcher
        mountPath: /shared-tokens
  - name: sidecar-http
    image: {{ required "images.carbonio-message-dispatcher-http-sidecar is required" (index .Values.images "carbonio-message-dispatcher-http-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: token-message-dispatcher
        mountPath: /shared-tokens
  - name: sidecar-xmpp
    image: {{ required "images.carbonio-message-dispatcher-xmpp-sidecar is required" (index .Values.images "carbonio-message-dispatcher-xmpp-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    env:
      - name: CONSUL_HTTP_TOKEN
        valueFrom:
          configMapKeyRef:
            name: consul-acl
            key: bootstrap-token
    volumeMounts:
      - name: token-message-dispatcher
        mountPath: /shared-tokens
volumes:
  - name: token-message-dispatcher
    emptyDir: {}
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: message-dispatcher-consul-data
    persistentVolumeClaim:
      claimName: message-dispatcher-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.tasks.podbody" -}}
containers:
  - name: app
    image: {{ required "images.carbonio-tasks-ce is required" (index .Values.images "carbonio-tasks-ce") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: tasks-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-tasks-sidecar is required" (index .Values.images "carbonio-tasks-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
volumes:
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: tasks-consul-data
    persistentVolumeClaim:
      claimName: tasks-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.preview.podbody" -}}
containers:
  - name: app
    image: {{ required "images.carbonio-preview-ce is required" (index .Values.images "carbonio-preview-ce") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
  - name: consul-agent
    image: {{ required "images.consul is required" (index .Values.images "consul") }}
    args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
    volumeMounts:
      - name: consul-agent-acl
        mountPath: /consul/acl/consul-agent-acl.hcl
        subPath: consul-agent-acl.hcl
        readOnly: true
{{- if eq .Values.platform "podman" }}
      - name: preview-consul-data
        mountPath: /consul/data
{{- end }}

  - name: sidecar
    image: {{ required "images.carbonio-preview-sidecar is required" (index .Values.images "carbonio-preview-sidecar") }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
volumes:
  - name: consul-agent-acl
    configMap:
      name: consul-acl
      items:
        - key: consul-agent-acl.hcl
          path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
  - name: preview-consul-data
    persistentVolumeClaim:
      claimName: preview-consul-data
{{- end }}
{{- end -}}

{{/*
Shared k3s Deployment wrapper preamble. Renders the Deployment head + pod
template metadata for a stateless service. Caller passes a dict:
  {{ include "carbonio.deployment.head" (dict "name" "carbonio-catalog") }}
and follows it with the pod body at indent 6. Kept as a helper so the
Deployment boilerplate lives in one place.
*/}}
{{- define "carbonio.deployment.k3sPodDefaults" -}}
enableServiceLinks: false
imagePullSecrets:
  - name: {{ .Values.image.pullSecret.name }}
{{- end -}}

{{/*
--- Stateful singletons (k3s StatefulSet / podman Pod) ----------------------
Like the mailbox helpers: containers and non-PVC ("shared") volumes live in
helpers, while the PVC-backed volumes are supplied per branch — as
volumeClaimTemplates on k3s, as claimName volume refs on podman. Callers write
the `containers:` / `volumes:` keys and include with `indent N`.
*/}}

{{- define "carbonio.postgres.containers" -}}
- name: postgres
  image: {{ required "images.postgres is required" (index .Values.images "postgres") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: POSTGRES_DB
      value: "postgres"
    - name: POSTGRES_USER
      value: "admin"
    - name: POSTGRES_PASSWORD
      value: "admin"
  ports:
    - containerPort: 5432
  volumeMounts:
    - name: postgres-data
      mountPath: /var/lib/postgresql/data
    - name: init-scripts
      mountPath: /docker-entrypoint-initdb.d
- name: consul-agent
  image: {{ required "images.consul is required" (index .Values.images "consul") }}
  args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
  volumeMounts:
    - name: consul-agent-acl
      mountPath: /consul/acl/consul-agent-acl.hcl
      subPath: consul-agent-acl.hcl
      readOnly: true
{{- if eq .Values.platform "podman" }}
    - name: postgres-consul-data
      mountPath: /consul/data
{{- end }}
{{- if and .Values.groups.mails.enabled (ne .Values.edition "ce") }}
- name: mailbox-db-sidecar
  image: {{ required "images.carbonio-mailbox-db-sidecar is required" (index .Values.images "carbonio-mailbox-db-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
    - name: PGPASSWORD
      value: "password"
    - name: POSTGRES_USER
      value: "carbonio-mailbox-adm"
{{- end }}
{{- if .Values.groups.files.enabled }}
- name: files-db-sidecar
  image: {{ required "images.carbonio-files-db-sidecar is required" (index .Values.images "carbonio-files-db-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
    - name: PGPASSWORD
      value: "password"
    - name: POSTGRES_USER
      value: "carbonio-files-adm"
- name: docs-connector-db-sidecar
  image: {{ required "images.carbonio-docs-connector-db-sidecar is required" (index .Values.images "carbonio-docs-connector-db-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
    - name: PGPASSWORD
      value: "password"
    - name: POSTGRES_USER
      value: "carbonio-docs-connector-adm"
{{- end }}
{{- if .Values.groups.wsc.enabled }}
- name: message-dispatcher-db-sidecar
  image: {{ required "images.carbonio-message-dispatcher-db-sidecar is required" (index .Values.images "carbonio-message-dispatcher-db-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
    - name: PGPASSWORD
      value: "password"
    - name: POSTGRES_USER
      value: "carbonio_adm"
- name: ws-collaboration-db-sidecar
  image: {{ required "images.carbonio-ws-collaboration-db-sidecar is required" (index .Values.images "carbonio-ws-collaboration-db-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
    - name: PGPASSWORD
      value: "password"
    - name: POSTGRES_USER
      value: "carbonio_adm"
{{- end }}
{{- end -}}

{{- define "carbonio.postgres.sharedVolumes" -}}
- name: init-scripts
  configMap:
    name: postgres-init-scripts
- name: consul-agent-acl
  configMap:
    name: consul-acl
    items:
      - key: consul-agent-acl.hcl
        path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
- name: postgres-consul-data
  persistentVolumeClaim:
    claimName: postgres-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.openldap.containers" -}}
- name: openldap
  image: {{ required "images.carbonio-openldap is required" (index .Values.images "carbonio-openldap") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  ports:
    - containerPort: 1389
  volumeMounts:
    - name: ldap-data
      mountPath: /opt/zextras/data/ldap
    - name: ldap-run
      mountPath: /run/carbonio
    - name: ldap-conf
      mountPath: /opt/zextras/conf
{{- end -}}

{{- define "carbonio.storages.containers" -}}
- name: app
  image: {{ required "images.carbonio-storages-service is required" (index .Values.images "carbonio-storages-service") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: JDK_JAVA_OPTIONS
      value: "-Xms128m -Xmx512m"
    - name: QUARKUS_OTEL_SDK_DISABLED
      value: "{{ not .Values.monitoring.enabled }}"
    - name: QUARKUS_OTEL_EXPORTER_OTLP_ENDPOINT
      value: "http://carbonio-monitoring:4317"
  volumeMounts:
    - name: storages-data
      mountPath: /storage
    - name: storages-token
      mountPath: /etc/carbonio/storages/service-discover
- name: consul-agent
  image: {{ required "images.consul is required" (index .Values.images "consul") }}
  args: [agent, -client=0.0.0.0, -bind=0.0.0.0, -retry-join=consul-server, -grpc-port=8502, -config-dir=/consul/acl]
  volumeMounts:
    - name: consul-agent-acl
      mountPath: /consul/acl/consul-agent-acl.hcl
      subPath: consul-agent-acl.hcl
      readOnly: true
{{- if eq .Values.platform "podman" }}
    - name: storages-consul-data
      mountPath: /consul/data
{{- end }}

- name: sidecar
  image: {{ required "images.carbonio-storages-sidecar is required" (index .Values.images "carbonio-storages-sidecar") }}
  imagePullPolicy: {{ .Values.image.pullPolicy }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  volumeMounts:
    - name: storages-token
      mountPath: /shared-tokens
{{- end -}}

{{- define "carbonio.storages.sharedVolumes" -}}
- name: storages-token
  emptyDir: {}
- name: consul-agent-acl
  configMap:
    name: consul-acl
    items:
      - key: consul-agent-acl.hcl
        path: consul-agent-acl.hcl
{{- if eq .Values.platform "podman" }}
- name: storages-consul-data
  persistentVolumeClaim:
    claimName: storages-consul-data
{{- end }}
{{- end -}}

{{- define "carbonio.consul-server.containers" -}}
- name: consul
  image: {{ required "images.consul is required" (index .Values.images "consul") }}
  args:
    - agent
    - -server
    - -bootstrap-expect=1
    - -ui
    - -client=0.0.0.0
    - -grpc-port=8502
    - -bind=0.0.0.0
    - -data-dir=/consul/data
    - -disable-host-node-id
    - -node=consul-server
    - -config-dir=/consul/acl
  ports:
    - containerPort: 8500
      hostPort: 8500
    - containerPort: 8502
      hostPort: 8502
  volumeMounts:
    - name: consul-data
      mountPath: /consul/data
    - name: consul-server-acl
      mountPath: /consul/acl/consul-server-acl.hcl
      subPath: consul-server-acl.hcl
      readOnly: true
- name: kv-init
  image: {{ required "images.curl is required" (index .Values.images "curl") }}
  env:
    - name: CONSUL_HTTP_TOKEN
      valueFrom:
        configMapKeyRef:
          name: consul-acl
          key: bootstrap-token
  command: ["/bin/sh", "-c"]
  args:
    - |
      echo "[kv-init] Waiting for consul server..."
      until curl -sf http://localhost:8500/v1/status/leader | grep -q ':'; do sleep 1; done
      echo "[kv-init] Consul ready. Running KV init scripts..."
      for f in /consul-kv/*.sh; do
        echo "[kv-init] Running $(basename "$f")..."
        sh "$f"
      done
      echo "[kv-init] Done. Sleeping."
      exec sleep infinity
  volumeMounts:
    - name: consul-kv-scripts
      mountPath: /consul-kv
      readOnly: true
{{- end -}}

{{- define "carbonio.consul-server.sharedVolumes" -}}
- name: consul-kv-scripts
  configMap:
    name: consul-kv-scripts
- name: consul-server-acl
  configMap:
    name: consul-acl
    items:
      - key: consul-server-acl.hcl
        path: consul-server-acl.hcl
{{- end -}}
