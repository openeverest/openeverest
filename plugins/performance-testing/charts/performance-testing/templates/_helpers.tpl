{{- define "performance-testing.fullname" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "performance-testing.labels" -}}
app.kubernetes.io/name: {{ include "performance-testing.fullname" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}
