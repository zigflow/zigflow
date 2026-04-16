{*
 Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*}

{{/*
Expand the name of the chart.
*/}}
{{- define "zigflow.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "zigflow.fullname" -}}
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
{{- define "zigflow.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "zigflow.labels" -}}
helm.sh/chart: {{ include "zigflow.chart" . }}
{{ include "zigflow.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "zigflow.selectorLabels" -}}
app.kubernetes.io/name: {{ include "zigflow.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "zigflow.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "zigflow.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Render the shared Pod template used by both Deployment and TemporalWorkerDeployment.
*/}}
{{- define "zigflow.podTemplate" -}}
template:
  metadata:
    annotations:
      # Prefill the Prometheus annotations. To activate, you will need to set:
      # prometheus.io/scrape: "true"
      prometheus.io/path: "/metrics"
      prometheus.io/port: {{ .Values.service.metrics.port | quote }}
      {{- if .Values.workflow.enabled }}
      {{- if .Values.workflow.useInline }}
      checksum/workflow-inline: {{ .Values.workflow.inline | toYaml | b64enc | sha256sum | quote }}
      {{- end }}
      {{- if .Values.workflow.rawFile }}
      checksum/workflow-raw: {{ .Values.workflow.rawFile | sha256sum | quote }}
      {{- end }}
      checksum/workflow-secret: {{ .Values.workflow.secret | sha256sum | quote }}
      {{- end }}
    {{- with .Values.podAnnotations }}
      {{- toYaml . | nindent 8 }}
    {{- end }}
    labels:
      {{- include "zigflow.labels" . | nindent 8 }}
      {{- with .Values.podLabels }}
      {{- toYaml . | nindent 8 }}
      {{- end }}
  spec:
    {{- with .Values.imagePullSecrets }}
    imagePullSecrets:
      {{- toYaml . | nindent 8 }}
    {{- end }}
    serviceAccountName: {{ include "zigflow.serviceAccountName" . }}
    {{- with .Values.podSecurityContext }}
    securityContext:
      {{- toYaml . | nindent 8 }}
    {{- end }}
    containers:
      - name: {{ .Chart.Name }}
        {{- with .Values.securityContext }}
        securityContext:
          {{- toYaml . | nindent 12 }}
        {{- end }}
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.Version }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        args:
          - run
        {{- if .Values.workflow.enabled }}
          - --file={{ .Values.workflow.file }}
        {{- end}}
        {{- range $key, $value := .Values.config }}
          - {{ printf "--%s=%v" $key $value | quote }}
        {{- end }}
        env:
          - name: ENABLE_VERSIONING
            value: {{ .Values.controller.enabled | quote }}
          - name: HEALTH_LISTEN_ADDRESS
            value: {{ printf "0.0.0.0:%.0f" .Values.service.health.port | quote }}
          - name: METRICS_LISTEN_ADDRESS
            value: {{ printf "0.0.0.0:%.0f" .Values.service.metrics.port | quote }}
        {{- with .Values.envvars }}
          {{- toYaml . | nindent 12 }}
        {{- end }}
        ports:
          - name: health
            containerPort: {{ .Values.service.health.port }}
            protocol: TCP
          - name: metrics
            containerPort: {{ .Values.service.metrics.port }}
            protocol: TCP
        {{- with .Values.livenessProbe }}
        livenessProbe:
          {{- toYaml . | nindent 12 }}
        {{- end }}
        {{- with .Values.readinessProbe }}
        readinessProbe:
          {{- toYaml . | nindent 12 }}
        {{- end }}
        {{- with .Values.resources }}
        resources:
          {{- toYaml . | nindent 12 }}
        {{- end }}
        volumeMounts:
          - mountPath: /tmp
            name: tmp
        {{- if or .Values.workflow.enabled .Values.volumes }}
        {{- if .Values.workflow.enabled }}
          - mountPath: {{ .Values.workflow.file | quote }}
            subPath: workflow.yaml
            name: workflow
            readOnly: true
        {{- end }}
        {{- with .Values.volumeMounts }}
          {{- toYaml . | nindent 12 }}
        {{- end }}
        {{- end }}
    volumes:
      # Zigflow can create small temporary files
      - name: tmp
        emptyDir:
          {{- with (.Values.tmpVolume | default dict) }}
          {{- if .medium | default "" }}
          medium: {{ .medium }}
          {{- end }}
          sizeLimit: {{ .sizeLimit | default "32Mi" }}
          {{- end }}
    {{- if .Values.workflow.enabled }}
      - name: workflow
        secret:
          secretName: {{ .Values.workflow.secret }}
    {{- end }}
    {{- with .Values.volumes }}
      {{- toYaml . | nindent 8 }}
    {{- end }}
    {{- with .Values.nodeSelector }}
    nodeSelector:
      {{- toYaml . | nindent 8 }}
    {{- end }}
    {{- with .Values.affinity }}
    affinity:
      {{- toYaml . | nindent 8 }}
    {{- end }}
    {{- with .Values.tolerations }}
    tolerations:
      {{- toYaml . | nindent 8 }}
    {{- end }}
{{- end }}

{{/*
Render autoscaling metrics for HPA and WorkerResourceTemplate.
*/}}
{{- define "zigflow.autoscalingMetrics" -}}
{{- if .Values.autoscaling.targetCPUUtilizationPercentage }}
- type: Resource
  resource:
    name: cpu
    target:
      type: Utilization
      averageUtilization: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
{{- end }}
{{- if .Values.autoscaling.targetMemoryUtilizationPercentage }}
- type: Resource
  resource:
    name: memory
    target:
      type: Utilization
      averageUtilization: {{ .Values.autoscaling.targetMemoryUtilizationPercentage }}
{{- end }}
{{- end }}
