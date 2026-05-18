/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package chart contains regression tests that render the Zigflow Helm chart
// via `helm template` and assert on the rendered output. The goal is to pin
// security-sensitive wiring (separate identities for worker and workload, RBAC
// bindings) so a refactor of the chart cannot silently broaden permissions.
package chart_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

const (
	releaseName = "zigflow"
	namespace   = "zigflow"
)

// k8sObject is the minimal shape we need to scan rendered output. We only
// look at kind, metadata.name and the loose body bag, which is enough for the
// assertions we care about and avoids pulling in the full Kubernetes API
// types in a test that doesn't need them.
type k8sObject struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   k8sObjectMeta          `json:"metadata"`
	Spec       map[string]interface{} `json:"spec,omitempty"`
	Subjects   []map[string]any       `json:"subjects,omitempty"`
	RoleRef    map[string]any         `json:"roleRef,omitempty"`
}

type k8sObjectMeta struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// renderChart runs `helm template` for the Zigflow chart with default values.
// helm and the chart directory are part of the repo, so this test runs on any
// developer machine without external state.
func renderChart(t *testing.T) []k8sObject {
	t.Helper()

	chartDir, err := filepath.Abs(filepath.Join("..", "..", "charts", "zigflow"))
	require.NoError(t, err)

	cmd := exec.CommandContext(
		t.Context(), "helm", "template", releaseName, chartDir,
		"--namespace", namespace,
	)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("helm template failed: %v\nstderr:\n%s", err, ee.Stderr)
		}
		t.Fatalf("helm template failed: %v", err)
	}

	docs := splitYAMLDocuments(string(out))
	objs := make([]k8sObject, 0, len(docs))
	for _, doc := range docs {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var o k8sObject
		if err := yaml.Unmarshal([]byte(doc), &o); err != nil {
			t.Fatalf("could not parse rendered doc: %v\n%s", err, doc)
		}
		if o.Kind == "" {
			continue
		}
		objs = append(objs, o)
	}
	return objs
}

// splitYAMLDocuments splits a multi-document YAML stream on the document
// separator. helm template emits one document per resource separated by "---"
// lines, plus a leading separator before the first resource.
func splitYAMLDocuments(in string) []string {
	parts := strings.Split(in, "\n---\n")
	// Trim a trailing newline-only fragment helm sometimes leaves behind.
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

func findOne(t *testing.T, objs []k8sObject, kind, name string) *k8sObject {
	t.Helper()
	for i := range objs {
		if objs[i].Kind == kind && objs[i].Metadata.Name == name {
			return &objs[i]
		}
	}
	t.Fatalf("no %s/%s in rendered output", kind, name)
	return nil
}

func TestChart_WorkerAndWorkloadServiceAccountsAreSeparate(t *testing.T) {
	objs := renderChart(t)

	// The fullname template prepends the release name when it does not
	// contain the chart name. With release "zigflow" and chart "zigflow"
	// the worker SA name is just the release name.
	worker := findOne(t, objs, "ServiceAccount", releaseName)
	workload := findOne(t, objs, "ServiceAccount", releaseName+"-workload")

	assert.NotEqual(t, worker.Metadata.Name, workload.Metadata.Name,
		"worker and workload service accounts must not share a name")
}

func TestChart_WorkloadServiceAccountDisablesAutomount(t *testing.T) {
	objs := renderChart(t)
	workload := findOne(t, objs, "ServiceAccount", releaseName+"-workload")

	// The Kind/Metadata-only k8sObject loses spec-level fields that are
	// outside .spec. Re-render the chart and parse the workload SA doc
	// fully into a generic map so automountServiceAccountToken is visible.
	chartDir, err := filepath.Abs(filepath.Join("..", "..", "charts", "zigflow"))
	require.NoError(t, err)
	cmd := exec.CommandContext(
		t.Context(), "helm", "template", releaseName, chartDir,
		"--namespace", namespace,
		"--show-only", "templates/workload-serviceaccount.yaml",
	)
	out, err := cmd.Output()
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, yaml.Unmarshal(out, &raw))
	require.Equal(t, "ServiceAccount", raw["kind"])
	require.Equal(t, workload.Metadata.Name, raw["metadata"].(map[string]any)["name"])

	v, ok := raw["automountServiceAccountToken"]
	require.True(t, ok, "automountServiceAccountToken must be set on workload SA")
	assert.Equal(t, false, v, "workload SA must default to automount=false")
}

func TestChart_WorkerDeploymentUsesWorkerServiceAccount(t *testing.T) {
	objs := renderChart(t)
	dep := findOne(t, objs, "Deployment", releaseName)

	tmpl := dep.Spec["template"].(map[string]any)
	spec := tmpl["spec"].(map[string]any)
	assert.Equal(t, releaseName, spec["serviceAccountName"],
		"worker deployment must run under the worker service account")
}

func TestChart_RoleBindingBindsOnlyTheWorkerServiceAccount(t *testing.T) {
	objs := renderChart(t)
	rb := findOne(t, objs, "RoleBinding", releaseName)

	require.Len(t, rb.Subjects, 1, "RoleBinding must have exactly one subject")
	sub := rb.Subjects[0]
	assert.Equal(t, "ServiceAccount", sub["kind"])
	assert.Equal(t, releaseName, sub["name"],
		"RoleBinding subject must be the worker SA, never the workload SA")
	assert.NotEqual(t, releaseName+"-workload", sub["name"],
		"workload SA must not appear in the worker RoleBinding")
}

func TestChart_ContainerRuntimeServiceAccountEnvPointsToWorkloadSA(t *testing.T) {
	objs := renderChart(t)
	dep := findOne(t, objs, "Deployment", releaseName)

	tmpl := dep.Spec["template"].(map[string]any)
	spec := tmpl["spec"].(map[string]any)
	containers := spec["containers"].([]any)
	require.Len(t, containers, 1)

	envs := containers[0].(map[string]any)["env"].([]any)
	var got map[string]any
	for _, e := range envs {
		m := e.(map[string]any)
		if m["name"] == "CONTAINER_RUNTIME_SERVICE_ACCOUNT" {
			got = m
			break
		}
	}
	require.NotNil(t, got, "CONTAINER_RUNTIME_SERVICE_ACCOUNT env var must be present")

	// The hardening: the env value is the workload SA literal, not a
	// downward-API reference to the worker's own service account.
	assert.Equal(t, releaseName+"-workload", got["value"],
		"CONTAINER_RUNTIME_SERVICE_ACCOUNT must resolve to the workload SA")
	_, hasValueFrom := got["valueFrom"]
	assert.False(t, hasValueFrom,
		"CONTAINER_RUNTIME_SERVICE_ACCOUNT must not be sourced from the downward API")
}

func TestChart_WorkloadServiceAccountCreateCanBeDisabled(t *testing.T) {
	chartDir, err := filepath.Abs(filepath.Join("..", "..", "charts", "zigflow"))
	require.NoError(t, err)

	// When workloadServiceAccount.create=false the chart still resolves a
	// name (defaulting to "default") but no SA resource is rendered. This
	// proves operators can BYO the workload identity without breaking the
	// chart.
	cmd := exec.CommandContext(
		t.Context(), "helm", "template", releaseName, chartDir,
		"--namespace", namespace,
		"--set", "workloadServiceAccount.create=false",
		"--set", "workloadServiceAccount.name=my-workload",
	)
	out, err := cmd.Output()
	require.NoError(t, err)

	rendered := string(out)
	assert.NotContains(t, rendered, "name: "+releaseName+"-workload",
		"workload SA must not be rendered when create=false")
	assert.Contains(t, rendered, `value: "my-workload"`,
		"CONTAINER_RUNTIME_SERVICE_ACCOUNT must use the explicit workload name")
}
