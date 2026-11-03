{{- define "spring-app.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "spring-app.labels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Resource presets, used when .Values.resources is empty. */}}
{{- define "spring-app.resources.preset" -}}
{{- $presets := dict
  "nano" (dict "requests" (dict "cpu" "100m" "memory" "128Mi") "limits" (dict "cpu" "150m" "memory" "192Mi"))
  "micro" (dict "requests" (dict "cpu" "250m" "memory" "256Mi") "limits" (dict "cpu" "375m" "memory" "384Mi"))
  "small" (dict "requests" (dict "cpu" "500m" "memory" "512Mi") "limits" (dict "cpu" "750m" "memory" "768Mi"))
  "medium" (dict "requests" (dict "cpu" "500m" "memory" "1024Mi") "limits" (dict "cpu" "750m" "memory" "1536Mi"))
  "large" (dict "requests" (dict "cpu" "1" "memory" "2048Mi") "limits" (dict "cpu" "1.5" "memory" "3072Mi"))
-}}
{{- if hasKey $presets .type -}}
{{- index $presets .type | toYaml -}}
{{- else -}}
{{- printf "unknown resourcesPreset %q" .type | fail -}}
{{- end -}}
{{- end -}}
