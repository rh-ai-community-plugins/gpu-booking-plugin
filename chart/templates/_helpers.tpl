{{/*
Target namespace for all namespaced resources.
*/}}
{{- define "gpu-booking.namespace" -}}
{{- .Values.namespace | default .Release.Namespace }}
{{- end }}

{{/*
Expand the name of the chart.
*/}}
{{- define "gpu-booking.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "gpu-booking.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "gpu-booking.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "gpu-booking.labels" -}}
helm.sh/chart: {{ include "gpu-booking.chart" . }}
{{ include "gpu-booking.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "gpu-booking.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gpu-booking.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the frontend service account to use
*/}}
{{- define "gpu-booking.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "gpu-booking.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the backend (BFF) service account to use
*/}}
{{- define "gpu-booking.bffServiceAccountName" -}}
{{- if .Values.bff.serviceAccount.create }}
{{- default (printf "%s-bff" (include "gpu-booking.fullname" .)) .Values.bff.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.bff.serviceAccount.name }}
{{- end }}
{{- end }}
