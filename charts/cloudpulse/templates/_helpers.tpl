{{/*
Expand the name of the chart.
*/}}
{{- define "cloudpulse.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "cloudpulse.fullname" -}}
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
Chart label
*/}}
{{- define "cloudpulse.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cloudpulse.labels" -}}
helm.sh/chart: {{ include "cloudpulse.chart" . }}
{{ include "cloudpulse.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cloudpulse.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cloudpulse.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Backend full name
*/}}
{{- define "cloudpulse.backend.fullname" -}}
{{- printf "%s-%s" (include "cloudpulse.fullname" .) .Values.backend.name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Frontend full name
*/}}
{{- define "cloudpulse.frontend.fullname" -}}
{{- printf "%s-%s" (include "cloudpulse.fullname" .) .Values.frontend.name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Driver frontend full name
*/}}
{{- define "cloudpulse.driverFrontend.fullname" -}}
{{- printf "%s-%s" (include "cloudpulse.fullname" .) .Values.driverFrontend.name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Service account name
*/}}
{{- define "cloudpulse.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "cloudpulse.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "cloudpulse.namespace" -}}
{{- if .Values.namespace.name }}
{{- .Values.namespace.name }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}
