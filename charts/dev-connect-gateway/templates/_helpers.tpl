{{- define "dev-connect-gateway.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "dev-connect-gateway.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" (include "dev-connect-gateway.name" .) .Values.target.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

