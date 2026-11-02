package loader

import (
	"path/filepath"
	"testing"
)

const multiDoc = `
apiVersion: v1
kind: Service
metadata:
  name: api
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: shop
spec:
  template:
    spec:
      containers:
        - name: app
          image: eclipse-temurin:17-jre
          env:
            - name: JAVA_TOOL_OPTIONS
              value: -Xmx512m
            - name: SECRET
              valueFrom:
                secretKeyRef: {name: s, key: k}
          resources:
            limits: {memory: 1Gi, cpu: 500m}
          livenessProbe:
            initialDelaySeconds: 10
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: report
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: job
              image: amazoncorretto:21
---
apiVersion: v1
kind: List
items:
  - apiVersion: v1
    kind: Pod
    metadata:
      name: debug
    spec:
      containers:
        - name: c
          image: busybox
`

func TestParse(t *testing.T) {
	ws, err := Parse([]byte(multiDoc), "test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 3 {
		t.Fatalf("got %d workloads, want 3", len(ws))
	}
	d := ws[0]
	if d.ID() != "shop/Deployment/api" {
		t.Errorf("ID = %s", d.ID())
	}
	c := d.Spec.Containers[0]
	if c.Resources.Limits["memory"] != "1Gi" || c.Resources.Limits["cpu"] != "500m" {
		t.Errorf("limits = %v", c.Resources.Limits)
	}
	if c.LivenessProbe == nil || c.LivenessProbe.InitialDelaySeconds != 10 {
		t.Errorf("liveness = %+v", c.LivenessProbe)
	}
	if c.Env[1].ValueFrom == nil {
		t.Errorf("valueFrom not decoded")
	}
	if c.Line != 16 {
		t.Errorf("container line = %d, want 16", c.Line)
	}
	if ws[1].Kind != "CronJob" || ws[1].Spec.Containers[0].Image != "amazoncorretto:21" {
		t.Errorf("cronjob = %+v", ws[1])
	}
	if ws[2].Kind != "Pod" || ws[2].Name != "debug" {
		t.Errorf("list item = %+v", ws[2])
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse([]byte("{{- if .Values.enabled }}\nkind: Pod\n"), "t.yaml"); err == nil {
		t.Error("want error for template syntax")
	}
}

func TestLoadPaths(t *testing.T) {
	ws, err := LoadPaths([]string{filepath.Join("..", "..", "testdata")})
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) == 0 {
		t.Fatal("no workloads in testdata")
	}
}
