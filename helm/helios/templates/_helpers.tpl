{{- define "helios.name" -}}
{{- default .Chart.Name .Values.nameOverride -}}
{{- end -}}

{{- define "helios.fullname" -}}
{{- if .Values.fullnameOverride }}
  {{- .Values.fullnameOverride -}}
{{- else }}
  {{- printf "%s-%s" .Release.Name (include "helios.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end }}
{{- end -}}

{{- define "helios.labels" -}}
app.kubernetes.io/name: {{ include "helios.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | default .Chart.Version }}
app.kubernetes.io/managed-by: Helm
{{- end -}}
