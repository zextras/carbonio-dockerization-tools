{{/*
Returns "true" if any service group is enabled. Used to gate the shared core
infra (consul-server, openldap, postgres, monitoring) that every group needs.
*/}}
{{- define "carbonio.coreEnabled" -}}
{{- or .Values.groups.mails.enabled .Values.groups.files.enabled .Values.groups.wsc.enabled .Values.groups.ui.enabled .Values.groups.todo.enabled -}}
{{- end -}}

