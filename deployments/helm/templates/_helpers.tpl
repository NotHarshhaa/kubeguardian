{{/*
Expand the name of the chart.
*/}}
{{- define "kubeguardian.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubeguardian.fullname" -}}
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
{{- define "kubeguardian.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kubeguardian.labels" -}}
helm.sh/chart: {{ include "kubeguardian.chart" . }}
{{ include "kubeguardian.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kubeguardian.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubeguardian.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: controller
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kubeguardian.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kubeguardian.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "kubeguardian.image" -}}
{{- $registry := .Values.image.registry -}}
{{- $repository := .Values.image.repository -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- if .Values.image.digest }}
{{- printf "%s/%s@%s" $registry $repository .Values.image.digest -}}
{{- else -}}
{{- printf "%s/%s:%s" $registry $repository $tag -}}
{{- end -}}
{{- end }}

{{/*
Return the proper Docker Image Registry Secret Names
*/}}
{{- define "kubeguardian.imagePullSecrets" -}}
{{- include "common.images.pullSecrets" (dict "images" (list .Values.image) "context" $) -}}
{{- end }}

{{/*
Create the config map name
*/}}
{{- define "kubeguardian.configMapName" -}}
{{- printf "%s-config" (include "kubeguardian.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the rules config map name
*/}}
{{- define "kubeguardian.rulesConfigMapName" -}}
{{- printf "%s-rules" (include "kubeguardian.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the secret name
*/}}
{{- define "kubeguardian.secretName" -}}
{{- printf "%s-secrets" (include "kubeguardian.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Return the proper metrics service name
*/}}
{{- define "kubeguardian.metricsServiceName" -}}
{{- printf "%s-metrics" (include "kubeguardian.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Return the proper health service name
*/}}
{{- define "kubeguardian.healthServiceName" -}}
{{- printf "%s-health" (include "kubeguardian.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Validate values
*/}}
{{- define "kubeguardian.validateValues" -}}
{{- $messages := list }}
{{- $messages := append $messages (include "kubeguardian.validateValues.controller" .) }}
{{- $messages := append $messages (include "kubeguardian.validateValues.detection" .) }}
{{- $messages := append $messages (include "kubeguardian.validateValues.remediation" .) }}
{{- $messages := without $messages "" }}
{{- $message := join "\n" $messages -}}
{{- if $message }}
{{- printf "\nVALUES VALIDATION:\n%s" $message | fail -}}
{{- end }}
{{- end }}

{{/*
Validate controller values
*/}}
{{- define "kubeguardian.validateValues.controller" -}}
{{- if not .Values.controller.metricsAddr -}}
{{- "controller.metricsAddr is required" -}}
{{- end -}}
{{- end }}

{{/*
Validate detection values
*/}}
{{- define "kubeguardian.validateValues.detection" -}}
{{- if not .Values.detection.evaluationInterval -}}
{{- "detection.evaluationInterval is required" -}}
{{- end -}}
{{- end }}

{{/*
Validate remediation values
*/}}
{{- define "kubeguardian.validateValues.remediation" -}}
{{- if not (hasKey .Values.remediation "enabled") -}}
{{- "remediation.enabled is required" -}}
{{- end -}}
{{- end }}
