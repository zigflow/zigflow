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

package activities

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/testsuite"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

const (
	testKubeNamespace      = "workflows"
	testKubeServiceAccount = "workflows-sa"
	testKubeContainerName  = "test-container"
	testKubeImage          = "alpine:3.20"

	// Verb/resource constants used for asserting actions captured by the
	// fake client. They duplicate the values used by the Kubernetes
	// client-go testing helpers and exist here only to satisfy goconst.
	testVerbCreate         = "create"
	testVerbDelete         = "delete"
	testResourceJobs       = "jobs"
	testLifetimeEventually = "eventually"

	// API kinds used in synthetic OwnerReference fixtures.
	testJobAPIVersion = "batch/v1"
	testJobKind       = "Job"

	// Shared sentinel env entry used across container fixtures. Centralised
	// to keep goconst happy and to avoid drift between tests that all want a
	// "single arbitrary env var".
	testEnvName  = "FOO"
	testEnvValue = "bar"
)

// makeContainerTask returns a minimal RunTask suitable for the Kubernetes
// runtime tests. Tests mutate the returned value to customise specific
// scenarios; the helper keeps fixtures small and consistent.
func makeContainerTask() *model.RunTask {
	return &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Name:      testKubeContainerName,
				Image:     testKubeImage,
				Command:   "/bin/echo",
				Arguments: []string{"hello", "world"},
				Environment: map[string]string{
					testEnvName: testEnvValue,
				},
			},
		},
	}
}

// jobReactorOpts tunes the Create-jobs reactor installed by reactJobCreate.
// Reactor-driven pod seeding mirrors what the real Job controller does: a
// Pod is only created once the Job exists, so its OwnerReferences can point
// to the Job UID assigned at Create time.
type jobReactorOpts struct {
	// status is the JobStatus the reactor stamps onto the new Job before
	// it is added to the tracker. Required.
	status *batchv1.JobStatus
	// seedPod, when true, creates a Pod owned by the new Job. The Pod's
	// labels mirror Spec.Template.Labels and its controllerRef points to
	// the Job's UID. Defaults to true; set false to exercise the "no pods
	// found" branch.
	seedPod bool
	// extraPods, when set, are seeded into the tracker before the Job
	// reactor's own pod (if any). Use this to introduce stale pods,
	// orphans, or pods that would shadow the Job controller's pod
	// without an exact owner-ref match.
	extraPods []corev1.Pod
}

// reactJobCreate installs a Create reactor on Jobs that:
//  1. Assigns a deterministic Name and UID (the real apiserver does this
//     from GenerateName + the storage layer; the fake client does not).
//  2. Stamps the supplied JobStatus.
//  3. Adds the Job to the tracker so subsequent Get calls see the status.
//  4. Optionally seeds a controller-owned Pod into the tracker, matching the
//     behaviour of the real Job controller. The Pod's OwnerReferences point
//     to the Job, and its labels mirror Spec.Template.Labels.
//
// Returning (true, ...) suppresses the fake's default create-reactor, which
// would otherwise overwrite the tracker with an unmutated copy of the Job.
func reactJobCreate(t *testing.T, c *fake.Clientset, opts jobReactorOpts) {
	t.Helper()
	c.PrependReactor(testVerbCreate, testResourceJobs, func(action clienttesting.Action) (bool, runtime.Object, error) {
		create := action.(clienttesting.CreateAction)
		job := create.GetObject().(*batchv1.Job).DeepCopy()
		if job.Name == "" && job.GenerateName != "" {
			job.Name = job.GenerateName + "abc12"
		}
		// A stable UID is what couples the seeded Pod to the Job via
		// OwnerReferences. The Job controller-on-cluster generates a
		// fresh UID per Create; this fixed value is fine because the
		// fake clientset is reset between tests.
		if job.UID == "" {
			job.UID = types.UID(job.Name + "-uid")
		}
		if opts.status != nil {
			job.Status = *opts.status
		}
		if err := c.Tracker().Add(job); err != nil {
			return true, nil, err
		}

		for i := range opts.extraPods {
			p := opts.extraPods[i].DeepCopy()
			if p.Namespace == "" {
				p.Namespace = job.Namespace
			}
			if err := c.Tracker().Add(p); err != nil {
				return true, nil, err
			}
		}

		if opts.seedPod {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      job.Name + "-pod",
					Namespace: job.Namespace,
					Labels:    job.Spec.Template.Labels,
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: testJobAPIVersion,
						Kind:       testJobKind,
						Name:       job.Name,
						UID:        job.UID,
						Controller: utils.Ptr(true),
					}},
				},
			}
			if err := c.Tracker().Add(pod); err != nil {
				return true, nil, err
			}
		}

		return true, job, nil
	})
}

func completedJobReactor(t *testing.T, c *fake.Clientset) {
	t.Helper()
	reactJobCreate(t, c, jobReactorOpts{
		status: &batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
		seedPod: true,
	})
}

func failedJobReactor(t *testing.T, c *fake.Clientset, message string) {
	t.Helper()
	// No pod is seeded: production-side getJobLogs is never reached on the
	// failed-job path.
	reactJobCreate(t, c, jobReactorOpts{
		status: &batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:    batchv1.JobFailed,
				Status:  corev1.ConditionTrue,
				Message: message,
			}},
		},
	})
}

// runK8sActivity executes runKubernetesJobWithOptions inside a Temporal test
// activity environment so the production code reads real Workflow info (run
// ID, activity ID) instead of zero values. The fake clientset is wired in
// via the clientFactory seam. Pod seeding is driven entirely by the Create-
// jobs reactor so the seeded Pod's OwnerReferences can point at the Job UID
// assigned at Create time.
//
// namespace is fixed to testKubeNamespace because every Kubernetes test runs
// against the same target namespace; tests that need to exercise alternative
// values do so via buildJobSpec directly.
func runK8sActivity(
	t *testing.T,
	fakeClient *fake.Clientset,
	task *model.RunTask,
	serviceAccount string,
) (any, error) {
	t.Helper()

	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	testActivity := func(ctx context.Context) (any, error) {
		return run.runKubernetesJobWithOptions(
			ctx, task, utils.NewState(), testKubeNamespace, serviceAccount,
			kubernetesRuntimeOptions{
				clientFactory: func() (kubernetes.Interface, error) { return fakeClient, nil },
				// Use a very short interval so the poll loop is responsive
				// while tests stay fast. The reactor returns a terminal
				// status synchronously, so the loop normally exits on the
				// first tick.
				pollInterval: time.Millisecond,
			},
		)
	}
	env.RegisterActivity(testActivity)

	val, err := env.ExecuteActivity(testActivity)
	if err != nil {
		return nil, err
	}
	var out any
	require.NoError(t, val.Get(&out))
	return out, nil
}

func TestBuildJobSpec_PopulatesContainerFieldsAndLabels(t *testing.T) {
	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	task := makeContainerTask()

	var got *batchv1.Job
	testActivity := func(ctx context.Context) error {
		j, err := run.buildJobSpec(ctx, task, testKubeNamespace, testKubeServiceAccount, utils.NewState())
		if err != nil {
			return err
		}
		got = j
		return nil
	}
	env.RegisterActivity(testActivity)

	_, err := env.ExecuteActivity(testActivity)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, testKubeNamespace, got.Namespace)
	assert.Equal(t, jobNamePrefix, got.GenerateName)
	assert.Equal(t, labelValueAppName, got.Labels[labelKeyAppName])
	assert.Equal(t, labelValueAppComponent, got.Labels[labelKeyAppComponent])
	assert.Equal(t, testKubeContainerName, got.Labels[labelKeyContainerName])
	assert.NotEmpty(t, got.Labels[labelKeyRunID])
	assert.NotEmpty(t, got.Labels[labelKeyActivityID])

	require.Len(t, got.Spec.Template.Spec.Containers, 1)
	c := got.Spec.Template.Spec.Containers[0]
	assert.Equal(t, testKubeContainerName, c.Name)
	assert.Equal(t, testKubeImage, c.Image)
	assert.Equal(t, []string{"/bin/echo"}, c.Command)
	assert.Equal(t, []string{"hello", "world"}, c.Args)
	require.Len(t, c.Env, 1)
	assert.Equal(t, corev1.EnvVar{Name: testEnvName, Value: testEnvValue}, c.Env[0])

	assert.Equal(t, testKubeServiceAccount, got.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, corev1.RestartPolicyNever, got.Spec.Template.Spec.RestartPolicy)
	require.NotNil(t, got.Spec.BackoffLimit)
	assert.Equal(t, int32(0), *got.Spec.BackoffLimit)
}

// TestBuildJobSpec_UsesProvidedWorkloadServiceAccount pins the contract that
// the runtime passes the configured workload service account through to the
// generated Pod spec without modification, even when the value differs from
// any default. The chart relies on this to keep workload identity separate
// from the worker identity.
func TestBuildJobSpec_UsesProvidedWorkloadServiceAccount(t *testing.T) {
	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	const workloadSA = "zigflow-workload"

	var got *batchv1.Job
	testActivity := func(ctx context.Context) error {
		j, err := run.buildJobSpec(ctx, makeContainerTask(), testKubeNamespace, workloadSA, utils.NewState())
		got = j
		return err
	}
	env.RegisterActivity(testActivity)

	_, err := env.ExecuteActivity(testActivity)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, workloadSA, got.Spec.Template.Spec.ServiceAccountName)
}

// TestBuildJobSpec_AutomountServiceAccountTokenDefaultsToFalse guarantees
// that workflow-defined containers never receive a Kubernetes API token by
// default, even if the supplied workload service account would otherwise
// mount one. Tests that intentionally enable automount would have to flip
// this themselves, which is why the production code pins it explicitly.
func TestBuildJobSpec_AutomountServiceAccountTokenDefaultsToFalse(t *testing.T) {
	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	var got *batchv1.Job
	testActivity := func(ctx context.Context) error {
		j, err := run.buildJobSpec(ctx, makeContainerTask(), testKubeNamespace, "", utils.NewState())
		got = j
		return err
	}
	env.RegisterActivity(testActivity)

	_, err := env.ExecuteActivity(testActivity)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.NotNil(t, got.Spec.Template.Spec.AutomountServiceAccountToken,
		"AutomountServiceAccountToken must be explicitly set, not left nil")
	assert.False(t, *got.Spec.Template.Spec.AutomountServiceAccountToken)
}

// TestSanitiseLabelValue covers the small helper directly: simple valid
// values pass through unchanged, awkward values hash deterministically, and
// the empty string is handled (rather than producing an invalid empty
// label).
func TestSanitiseLabelValue(t *testing.T) {
	t.Parallel()

	run := &Run{}

	// Pre-compute the hash for an arbitrary awkward value so we can assert
	// the helper is deterministic and not just "some hash".
	const awkward = "Spaces and Caps!"
	awkwardHash := run.sanitiseLabelValue(awkward)

	tests := []struct {
		name           string
		in             string
		wantUnchanged  bool
		wantHashPrefix bool
	}{
		{
			name:          "simple alphanumeric value is left alone",
			in:            "simple-value-1",
			wantUnchanged: true,
		},
		{
			name:          "uuid-shaped value is left alone",
			in:            "01234567-89ab-cdef-0123-456789abcdef",
			wantUnchanged: true,
		},
		{
			name:          "underscored value is left alone",
			in:            "task_name_42",
			wantUnchanged: true,
		},
		{
			name:           "value with spaces is hashed",
			in:             "has spaces",
			wantHashPrefix: true,
		},
		{
			name:           "value starting with hyphen is hashed",
			in:             "-leading-hyphen",
			wantHashPrefix: true,
		},
		{
			name:           "value with slash is hashed",
			in:             "ns/name",
			wantHashPrefix: true,
		},
		{
			name:           "empty string is hashed",
			in:             "",
			wantHashPrefix: true,
		},
		{
			name:           "very long value is hashed and capped",
			in:             strings.Repeat("a", 64),
			wantHashPrefix: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := run.sanitiseLabelValue(tc.in)

			require.LessOrEqual(t, len(got), 63,
				"sanitised label must fit Kubernetes' 63-char limit")
			require.Empty(t, validation.IsValidLabelValue(got),
				"sanitised label %q must be a valid Kubernetes label value", got)

			if tc.wantUnchanged {
				assert.Equal(t, tc.in, got)
			}
			if tc.wantHashPrefix {
				assert.True(t, strings.HasPrefix(got, hashedLabelPrefix),
					"hashed value should start with %q, got %q", hashedLabelPrefix, got)
			}
		})
	}

	// Determinism: the same input always produces the same output.
	assert.Equal(t, awkwardHash, run.sanitiseLabelValue(awkward))

	// Collision resistance: distinct awkward inputs hash to distinct
	// outputs. We are not exhaustively testing SHA-256, just guarding
	// against a regression that swaps in something like a fixed string.
	assert.NotEqual(t, run.sanitiseLabelValue("foo bar"), run.sanitiseLabelValue("foo  bar"))
}

// TestSanitiseContainerName covers the DNS-1123 derivation directly: valid
// names pass through unchanged, invalid ones (uppercase, separators
// Kubernetes does not allow, empty, too long) are replaced with a hashed
// value that still satisfies the DNS-1123 label rules.
func TestSanitiseContainerName(t *testing.T) {
	t.Parallel()

	run := &Run{}

	// Pre-compute the hash for an awkward value so we can assert the
	// helper is deterministic, not just "some hash".
	const awkward = "Name With Spaces/and_slashes"
	awkwardHash := run.sanitiseContainerName(awkward)

	tests := []struct {
		name             string
		in               string
		wantUnchanged    bool
		wantSanitisedFmt bool
	}{
		{
			name:          "simple lowercase DNS label passes through",
			in:            "my-container",
			wantUnchanged: true,
		},
		{
			name:          "lowercase with digits passes through",
			in:            "worker-123",
			wantUnchanged: true,
		},
		{
			name:             "uppercase characters are not DNS-1123 and must be hashed",
			in:               "MyContainer",
			wantSanitisedFmt: true,
		},
		{
			name:             "underscores are not DNS-1123 and must be hashed",
			in:               "with_underscore",
			wantSanitisedFmt: true,
		},
		{
			name:             "slashes are not DNS-1123 and must be hashed",
			in:               "ns/container",
			wantSanitisedFmt: true,
		},
		{
			name:             "spaces are not DNS-1123 and must be hashed",
			in:               "has spaces",
			wantSanitisedFmt: true,
		},
		{
			name:             "leading hyphen is not DNS-1123 and must be hashed",
			in:               "-leading",
			wantSanitisedFmt: true,
		},
		{
			name:             "empty string is hashed deterministically",
			in:               "",
			wantSanitisedFmt: true,
		},
		{
			name:             "value longer than 63 chars is hashed and capped",
			in:               strings.Repeat("a", 64),
			wantSanitisedFmt: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := run.sanitiseContainerName(tc.in)

			require.LessOrEqual(t, len(got), 63,
				"sanitised container name must fit Kubernetes' DNS-1123 63-char limit")
			require.Empty(t, validation.IsDNS1123Label(got),
				"sanitised container name %q must be a valid DNS-1123 label", got)

			if tc.wantUnchanged {
				assert.Equal(t, tc.in, got)
			}
			if tc.wantSanitisedFmt {
				assert.True(t, strings.HasPrefix(got, sanitisedContainerNamePrefix),
					"hashed value should start with %q, got %q",
					sanitisedContainerNamePrefix, got)
			}
		})
	}

	// Determinism: same input always produces the same output.
	assert.Equal(t, awkwardHash, run.sanitiseContainerName(awkward))

	// Collision resistance: distinct awkward inputs hash to distinct
	// outputs so two workflows with similar-but-different bad names do not
	// collide on Container.Name.
	assert.NotEqual(t,
		run.sanitiseContainerName("Foo Bar"),
		run.sanitiseContainerName("Foo  Bar"))
}

// TestBuildJobSpec_AwkwardValuesProduceValidLabels exercises the regression
// path with deliberately hostile inputs: a container name longer than 63
// chars that also contains characters Kubernetes does not accept in label
// values. The Temporal test environment supplies fixed workflow/run/activity
// IDs that are already label-safe, so the workflow-ID and activity-ID
// hashing branches are covered by TestSanitiseLabelValue directly.
//
// The rendered Job must still be label-valid, and the raw values must be
// recoverable from annotations.
func TestBuildJobSpec_AwkwardValuesProduceValidLabels(t *testing.T) {
	const badContainer = "Name With Spaces/and slashes/" +
		// Pad past Kubernetes' 63-char label-value limit so both the
		// "invalid characters" and "too long" branches of sanitisation
		// fire together on this single value.
		"plus-some-padding-to-go-well-over-the-sixty-three-character-label-limit"

	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	task := makeContainerTask()
	task.Run.Container.Name = badContainer

	var got *batchv1.Job
	testActivity := func(ctx context.Context) error {
		j, err := run.buildJobSpec(ctx, task, testKubeNamespace, "", utils.NewState())
		got = j
		return err
	}
	env.RegisterActivity(testActivity)

	_, err := env.ExecuteActivity(testActivity)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Job labels and template labels must be identical: a selector built
	// from one must match the other.
	assert.Equal(t, got.Labels, got.Spec.Template.Labels)

	// Every label value must be Kubernetes-valid; this is the core
	// regression guard.
	for k, v := range got.Labels {
		require.LessOrEqual(t, len(v), 63,
			"label %q value %q exceeds 63 chars", k, v)
		require.Empty(t, validation.IsValidLabelValue(v),
			"label %q has invalid value %q", k, v)
	}

	// The container name was hostile: the label must have been replaced
	// with a hashed value (not just truncated, which would lose
	// uniqueness), and the raw value must survive in the matching
	// annotation.
	assert.True(t, strings.HasPrefix(got.Labels[labelKeyContainerName], hashedLabelPrefix),
		"container name with invalid chars should be hashed, got %q",
		got.Labels[labelKeyContainerName])
	assert.Equal(t, badContainer, got.Annotations[annotationKeyContainerName])

	// The Pod container name must be a valid DNS-1123 label even when the
	// raw workflow name is not. Without this, Job creation against a real
	// apiserver would fail with a validation error even though label
	// sanitisation would otherwise let the workflow run.
	require.Len(t, got.Spec.Template.Spec.Containers, 1)
	podContainerName := got.Spec.Template.Spec.Containers[0].Name
	require.LessOrEqual(t, len(podContainerName), 63,
		"pod container name %q exceeds 63 chars", podContainerName)
	require.Empty(t, validation.IsDNS1123Label(podContainerName),
		"pod container name %q must be a valid DNS-1123 label", podContainerName)
	assert.True(t, strings.HasPrefix(podContainerName, sanitisedContainerNamePrefix),
		"hostile container name should be replaced with %q-prefixed value, got %q",
		sanitisedContainerNamePrefix, podContainerName)
	assert.NotEqual(t, badContainer, podContainerName,
		"hostile raw container name must not be used verbatim as Container.Name")

	// Annotations must mirror across Job and Pod template, just like
	// labels, so the two stay in lockstep when copied.
	assert.Equal(t, got.Annotations, got.Spec.Template.Annotations)

	// The test env supplies fixed workflow/run/activity IDs; assert they
	// surface as annotations rather than asserting their exact values
	// (which are an SDK implementation detail).
	assert.NotEmpty(t, got.Annotations[annotationKeyWorkflowID])
	assert.NotEmpty(t, got.Annotations[annotationKeyRunID])
	assert.NotEmpty(t, got.Annotations[annotationKeyActivityID])
}

// TestGetJobLogsSelectorMatchesSanitisedLabels is the end-to-end guard for
// the round-trip between buildJobSpec and getJobLogs: the labels the runtime
// writes onto a pod must be selectable by the same runtime calling List.
// The fixture seeds a pod with sanitised labels, then drives runK8sActivity
// (which exercises getJobLogs) with awkward IDs. If the two sides ever
// diverge, log retrieval would fail with "no pods found for job".
func TestGetJobLogsSelectorMatchesSanitisedLabels(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	completedJobReactor(t, fakeClient)

	// runK8sActivity already sanitises the seeded pod's labels via
	// sanitiseLabelValue, mirroring production. A successful log fetch
	// proves the selector built inside getJobLogs matched.
	out, err := runK8sActivity(t, fakeClient, makeContainerTask(), testKubeServiceAccount)
	require.NoError(t, err)
	assert.Equal(t, "fake logs", out)
}

// TestBuildJobSpec_AnnotationsCarryRawValuesUnderNormalInputs verifies that
// even when sanitisation is a no-op, the raw values still land in
// annotations. This stops a refactor from accidentally dropping the
// annotation pass for simple inputs.
func TestBuildJobSpec_AnnotationsCarryRawValuesUnderNormalInputs(t *testing.T) {
	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	task := makeContainerTask()

	var got *batchv1.Job
	testActivity := func(ctx context.Context) error {
		j, err := run.buildJobSpec(ctx, task, testKubeNamespace, "", utils.NewState())
		got = j
		return err
	}
	env.RegisterActivity(testActivity)

	_, err := env.ExecuteActivity(testActivity)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Container name and run/activity IDs all came from defaults; assert
	// the annotations are populated rather than the exact values, because
	// the test env supplies the IDs.
	assert.NotEmpty(t, got.Annotations[annotationKeyRunID])
	assert.NotEmpty(t, got.Annotations[annotationKeyActivityID])
	assert.Equal(t, testKubeContainerName, got.Annotations[annotationKeyContainerName])
}

func TestBuildJobSpec_EventuallyLifetimeSetsTTL(t *testing.T) {
	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	task := makeContainerTask()
	task.Run.Container.Lifetime = &model.ContainerLifetime{
		Cleanup: testLifetimeEventually,
		After:   &model.Duration{Value: model.DurationInline{Seconds: 120}},
	}

	var got *batchv1.Job
	testActivity := func(ctx context.Context) error {
		j, err := run.buildJobSpec(ctx, task, testKubeNamespace, "", utils.NewState())
		got = j
		return err
	}
	env.RegisterActivity(testActivity)

	_, err := env.ExecuteActivity(testActivity)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Spec.TTLSecondsAfterFinished)
	assert.Equal(t, int32(120), *got.Spec.TTLSecondsAfterFinished)
}

func TestBuildJobSpec_EmptyNamespacePassesThroughUnchanged(t *testing.T) {
	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	task := makeContainerTask()

	var got *batchv1.Job
	testActivity := func(ctx context.Context) error {
		j, err := run.buildJobSpec(ctx, task, "", "", utils.NewState())
		got = j
		return err
	}
	env.RegisterActivity(testActivity)

	_, err := env.ExecuteActivity(testActivity)
	require.NoError(t, err)
	// The production code does not invent a default namespace; that decision
	// is left to the cluster (which will reject create) or the caller. Pin
	// that contract so a future "auto-default" change is a conscious one.
	assert.Equal(t, "", got.Namespace)
	assert.Equal(t, "", got.Spec.Template.Spec.ServiceAccountName)
}

// TestBuildJobSpec_NilArgumentsAndEnvironment is the regression guard for the
// review finding that optional run.container.arguments and
// run.container.environment could panic the Kubernetes runtime when omitted.
// All combinations of omitted/provided must produce a valid Job without
// panicking. The Args/Env shape pinned here is what corev1 honours as "no
// values configured" so the resulting pod spec matches a workflow that left
// the fields out of YAML.
func TestBuildJobSpec_NilArgumentsAndEnvironment(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{name: "both omitted", args: nil, env: nil},
		{name: "arguments omitted, environment provided", args: nil, env: map[string]string{testEnvName: testEnvValue}},
		{name: "environment omitted, arguments provided", args: []string{"a", "b"}, env: nil},
		{name: "explicit empty values are accepted", args: []string{}, env: map[string]string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run := &Run{}
			var s testsuite.WorkflowTestSuite
			env := s.NewTestActivityEnvironment()

			task := &model.RunTask{
				Run: model.RunTaskConfiguration{
					Container: &model.Container{
						Name:        testKubeContainerName,
						Image:       testKubeImage,
						Arguments:   tc.args,
						Environment: tc.env,
					},
				},
			}

			var got *batchv1.Job
			testActivity := func(ctx context.Context) error {
				j, err := run.buildJobSpec(ctx, task, testKubeNamespace, testKubeServiceAccount, utils.NewState())
				got = j
				return err
			}
			env.RegisterActivity(testActivity)

			_, err := env.ExecuteActivity(testActivity)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Len(t, got.Spec.Template.Spec.Containers, 1)
			c := got.Spec.Template.Spec.Containers[0]

			if len(tc.args) == 0 {
				assert.Empty(t, c.Args, "omitted/empty arguments must produce no Args entries")
			} else {
				assert.Equal(t, tc.args, c.Args)
			}

			if len(tc.env) == 0 {
				assert.Empty(t, c.Env, "omitted/empty environment must produce no Env entries")
			} else {
				require.Len(t, c.Env, len(tc.env))
				for _, ev := range c.Env {
					assert.Equal(t, tc.env[ev.Name], ev.Value)
				}
			}
		})
	}
}

// TestBuildJobSpec_RejectsVolumes is the regression guard for the second
// review finding: run.container.volumes is documented for the Docker runtime
// but is not implemented for Kubernetes. Silently dropping the field would
// produce a Job that does not match the workflow definition, so the Kubernetes
// runtime must reject the configuration with a clear error before any Job is
// created.
func TestBuildJobSpec_RejectsVolumes(t *testing.T) {
	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	task := makeContainerTask()
	task.Run.Container.Volumes = map[string]any{
		"/host/path": "/container/path",
	}

	testActivity := func(ctx context.Context) error {
		_, err := run.buildJobSpec(ctx, task, testKubeNamespace, testKubeServiceAccount, utils.NewState())
		return err
	}
	env.RegisterActivity(testActivity)

	_, err := env.ExecuteActivity(testActivity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run.container.volumes")
	assert.Contains(t, err.Error(), "Kubernetes")
}

// TestRunKubernetesJob_RejectsVolumesBeforeJobCreation pins the contract that
// the volume rejection short-circuits before deployKubernetesJob calls the
// fake clientset. If a future change moved the check below the Create call,
// the fake's recorded action list would show a stray Job create even though
// the activity returned an error.
func TestRunKubernetesJob_RejectsVolumesBeforeJobCreation(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	task := makeContainerTask()
	task.Run.Container.Volumes = map[string]any{
		"/host/path": "/container/path",
	}

	_, err := runK8sActivity(t, fakeClient, task, testKubeServiceAccount)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run.container.volumes")

	for _, action := range fakeClient.Actions() {
		if action.GetVerb() == testVerbCreate && action.GetResource().Resource == testResourceJobs {
			t.Fatalf("volume rejection must happen before any Job is created: %#v", action)
		}
	}
}

func TestRunKubernetesJob_SuccessReturnsPodLogs(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	completedJobReactor(t, fakeClient)

	task := makeContainerTask()
	out, err := runK8sActivity(t, fakeClient, task, testKubeServiceAccount)
	require.NoError(t, err)

	// The fake client's GetLogs subresource returns the string "fake logs"
	// by default; assert that the activity surfaces that as its result.
	assert.Equal(t, "fake logs", out)
}

func TestRunKubernetesJob_CreatesJobInRequestedNamespaceAndServiceAccount(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	completedJobReactor(t, fakeClient)

	task := makeContainerTask()
	_, err := runK8sActivity(t, fakeClient, task, testKubeServiceAccount)
	require.NoError(t, err)

	// Inspect the Create action the fake recorded.
	var created *batchv1.Job
	for _, action := range fakeClient.Actions() {
		if action.GetVerb() != testVerbCreate || action.GetResource().Resource != testResourceJobs {
			continue
		}
		created = action.(clienttesting.CreateAction).GetObject().(*batchv1.Job)
		break
	}
	require.NotNil(t, created, "expected a Job create call")

	assert.Equal(t, testKubeNamespace, created.Namespace)
	assert.Equal(t, testKubeServiceAccount, created.Spec.Template.Spec.ServiceAccountName)
	require.Len(t, created.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, testKubeImage, created.Spec.Template.Spec.Containers[0].Image)
}

func TestRunKubernetesJob_FailedJobIsSurfacedAsError(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	failedJobReactor(t, fakeClient, "OOMKilled")

	task := makeContainerTask()
	_, err := runK8sActivity(t, fakeClient, task, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OOMKilled")
}

func TestRunKubernetesJob_DeletesJobOnSuccess(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	completedJobReactor(t, fakeClient)

	_, err := runK8sActivity(t, fakeClient, makeContainerTask(), "")
	require.NoError(t, err)

	var deletes int
	for _, action := range fakeClient.Actions() {
		if action.GetVerb() == testVerbDelete && action.GetResource().Resource == testResourceJobs {
			deletes++
		}
	}
	assert.Equal(t, 1, deletes, "expected exactly one job delete on success")
}

func TestRunKubernetesJob_DeletesJobOnFailure(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	failedJobReactor(t, fakeClient, "boom")

	_, err := runK8sActivity(t, fakeClient, makeContainerTask(), "")
	require.Error(t, err)

	var deletes int
	for _, action := range fakeClient.Actions() {
		if action.GetVerb() == testVerbDelete && action.GetResource().Resource == testResourceJobs {
			deletes++
		}
	}
	assert.Equal(t, 1, deletes, "delete must run on failure to avoid leaking jobs")
}

func TestRunKubernetesJob_DoesNotDeleteWhenLifetimeNever(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	completedJobReactor(t, fakeClient)

	task := makeContainerTask()
	task.Run.Container.Lifetime = &model.ContainerLifetime{Cleanup: "never"}

	_, err := runK8sActivity(t, fakeClient, task, "")
	require.NoError(t, err)

	for _, action := range fakeClient.Actions() {
		if action.GetVerb() == testVerbDelete && action.GetResource().Resource == testResourceJobs {
			t.Fatalf("did not expect delete action when Lifetime.Cleanup=never")
		}
	}
}

func TestRunKubernetesJob_JobCreationErrorIsReturned(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fakeClient.PrependReactor("create", "jobs", func(_ clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "batch", Resource: "jobs"},
			"x",
			errors.New("forbidden by RBAC"),
		)
	})

	_, err := runK8sActivity(t, fakeClient, makeContainerTask(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden by RBAC")
}

func TestRunKubernetesJob_PodListErrorIsReturned(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	completedJobReactor(t, fakeClient)
	// Reactor matches only list (not create) so the helper can still seed a
	// pod via Create. Once the production code attempts to list, this
	// reactor surfaces the simulated API error.
	fakeClient.PrependReactor("list", "pods", func(_ clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("pod list failed")
	})

	_, err := runK8sActivity(t, fakeClient, makeContainerTask(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pod list failed")
}

func TestRunKubernetesJob_NoPodsFoundIsReturned(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	// Complete the Job but do NOT seed a Pod, so the production-side list
	// returns empty and the "no pods" branch fires.
	reactJobCreate(t, fakeClient, jobReactorOpts{
		status: &batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
	})

	_, err := runK8sActivity(t, fakeClient, makeContainerTask(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pods found")
}

// TestCallContainerActivity_RuntimeSelection_Kubernetes verifies the
// kubernetes runtime branch in CallContainerActivity dispatches to the
// fake-backed Kubernetes path rather than shelling out to Docker.
func TestCallContainerActivity_RuntimeSelection_Kubernetes(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	// The reactor now seeds a controller-owned Pod alongside the Job, so
	// getJobLogs locates it via OwnerReferences with no further fixture
	// plumbing required from this test.
	completedJobReactor(t, fakeClient)

	prev := kubernetesClientFactory
	kubernetesClientFactory = func() (kubernetes.Interface, error) { return fakeClient, nil }
	t.Cleanup(func() { kubernetesClientFactory = prev })

	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(run.CallContainerActivity)

	val, err := env.ExecuteActivity(
		run.CallContainerActivity,
		makeContainerTask(), nil, utils.NewState(),
		testKubeNamespace, ContainerRuntimeKubernetes, testKubeServiceAccount,
	)
	require.NoError(t, err)
	var out string
	require.NoError(t, val.Get(&out))
	assert.Equal(t, "fake logs", out)

	// The fake client must have observed the create — proving the
	// Kubernetes path was taken rather than the Docker path.
	var sawJobCreate bool
	for _, action := range fakeClient.Actions() {
		if action.GetVerb() == testVerbCreate && action.GetResource().Resource == testResourceJobs {
			sawJobCreate = true
			break
		}
	}
	assert.True(t, sawJobCreate, "kubernetes runtime must dispatch a Job create")
}

// TestCallContainerActivity_RuntimeSelection_DockerDoesNotTouchKubernetes
// asserts that the docker branch leaves the Kubernetes client factory
// untouched. PATH is emptied so the docker runtime fails fast on
// exec.LookPath, keeping the test deterministic and quick regardless of
// whether docker is installed on the host.
func TestCallContainerActivity_RuntimeSelection_DockerDoesNotTouchKubernetes(t *testing.T) {
	t.Setenv("PATH", "")

	fakeClient := fake.NewSimpleClientset()

	prev := kubernetesClientFactory
	kubernetesClientFactory = func() (kubernetes.Interface, error) { return fakeClient, nil }
	t.Cleanup(func() { kubernetesClientFactory = prev })

	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(run.CallContainerActivity)

	_, err := env.ExecuteActivity(
		run.CallContainerActivity,
		makeContainerTask(), nil, utils.NewState(),
		"", ContainerRuntimeDocker, "",
	)
	require.Error(t, err, "docker runtime should fail fast with PATH cleared")
	assert.Contains(t, err.Error(), "Docker not installed")

	for _, action := range fakeClient.Actions() {
		if action.GetVerb() == testVerbCreate && action.GetResource().Resource == testResourceJobs {
			t.Fatalf("docker runtime must not create a Kubernetes Job: %#v", action)
		}
	}
}

// makeOwnedPod is a small builder for pods used by the selection tests. It
// keeps each test focused on the scenario instead of OwnerReference plumbing.
func makeOwnedPod(name string, jobUID types.UID, created time.Time, controller bool) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         testKubeNamespace,
			CreationTimestamp: metav1.NewTime(created),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: testJobAPIVersion,
				Kind:       testJobKind,
				Name:       "owner",
				UID:        jobUID,
				Controller: utils.Ptr(controller),
			}},
		},
	}
}

// TestNewestPodControlledBy is the unit-level guard for the pod-selection
// rules. It covers every branch the production caller depends on so a
// future refactor cannot silently change which pod gets its logs read.
func TestNewestPodControlledBy(t *testing.T) {
	t.Parallel()

	run := &Run{}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-job",
			Namespace: testKubeNamespace,
			UID:       types.UID("this-job-uid"),
		},
	}
	otherJobUID := types.UID("other-job-uid")

	t0 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("returns nil when no pods are supplied", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, run.newestPodControlledBy(nil, job))
	})

	t.Run("returns nil when no pod is controlled by job", func(t *testing.T) {
		t.Parallel()
		pods := []corev1.Pod{
			makeOwnedPod("stale", otherJobUID, t0, true),
			{ObjectMeta: metav1.ObjectMeta{Name: "orphan"}}, // no owners
		}
		assert.Nil(t, run.newestPodControlledBy(pods, job))
	})

	t.Run("rejects non-controller owner references", func(t *testing.T) {
		t.Parallel()
		// controller=false is a "soft" reference in Kubernetes and must
		// never count as ownership.
		pods := []corev1.Pod{
			makeOwnedPod("weak", job.UID, t0, false),
		}
		assert.Nil(t, run.newestPodControlledBy(pods, job))
	})

	t.Run("picks the single matching pod", func(t *testing.T) {
		t.Parallel()
		pods := []corev1.Pod{
			makeOwnedPod("stale", otherJobUID, t0, true),
			makeOwnedPod("ours", job.UID, t0, true),
		}
		got := run.newestPodControlledBy(pods, job)
		require.NotNil(t, got)
		assert.Equal(t, "ours", got.Name)
	})

	t.Run("picks the newest matching pod by CreationTimestamp", func(t *testing.T) {
		t.Parallel()
		pods := []corev1.Pod{
			makeOwnedPod("older", job.UID, t0.Add(-1*time.Minute), true),
			makeOwnedPod("newer", job.UID, t0, true),
			makeOwnedPod("oldest", job.UID, t0.Add(-2*time.Minute), true),
		}
		got := run.newestPodControlledBy(pods, job)
		require.NotNil(t, got)
		assert.Equal(t, "newer", got.Name)
	})

	t.Run("ties on CreationTimestamp are broken by name for determinism", func(t *testing.T) {
		t.Parallel()
		pods := []corev1.Pod{
			makeOwnedPod("aaa", job.UID, t0, true),
			makeOwnedPod("zzz", job.UID, t0, true),
		}
		got := run.newestPodControlledBy(pods, job)
		require.NotNil(t, got)
		assert.Equal(t, "zzz", got.Name)
	})

	t.Run("a single stale pod under a different Job UID is rejected", func(t *testing.T) {
		t.Parallel()
		pods := []corev1.Pod{
			makeOwnedPod("retained-job-pod", otherJobUID, t0, true),
		}
		// Even when the labels match (selector concern, handled by the
		// caller) the owner-ref check must reject this pod. The unit test
		// supplies the pod directly so this guard is independent of the
		// label-selector code path.
		assert.Nil(t, run.newestPodControlledBy(pods, job))
	})
}

// TestGetJobLogs_StalePodFromEarlierJobIsIgnored is the integration-level
// regression: the activity completes successfully even when a pod with
// matching correlation labels but the wrong controllerRef is present.
// Before this fix, "pods.Items[0]" could have returned that pod's logs.
func TestGetJobLogs_StalePodFromEarlierJobIsIgnored(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	run := &Run{}
	stalePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stale-pod",
			Namespace: testKubeNamespace,
			// Same correlation labels the current activity will emit;
			// the test env supplies fixed IDs, so the selector alone
			// cannot tell this pod apart.
			Labels: map[string]string{
				labelKeyRunID:      run.sanitiseLabelValue("default-test-run-id"),
				labelKeyActivityID: run.sanitiseLabelValue("0"),
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: testJobAPIVersion,
				Kind:       testJobKind,
				Name:       "ancient-job",
				UID:        types.UID("ancient-job-uid"), // not our Job's UID
				Controller: utils.Ptr(true),
			}},
		},
	}

	reactJobCreate(t, fakeClient, jobReactorOpts{
		status: &batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
		seedPod:   true,
		extraPods: []corev1.Pod{stalePod},
	})

	out, err := runK8sActivity(t, fakeClient, makeContainerTask(), "")
	require.NoError(t, err)
	assert.Equal(t, "fake logs", out)
}

// TestGetJobLogs_PodWithMatchingLabelsButWrongOwnerSurfacesAsNoPods covers
// the orphan/retained-Job edge: when the only label-matching candidates
// have no controllerRef or point at a different Job, log retrieval reports
// "no pods found" rather than silently picking the wrong pod.
func TestGetJobLogs_PodWithMatchingLabelsButWrongOwnerSurfacesAsNoPods(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	run := &Run{}
	orphan := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphan-pod",
			Namespace: testKubeNamespace,
			Labels: map[string]string{
				labelKeyRunID:      run.sanitiseLabelValue("default-test-run-id"),
				labelKeyActivityID: run.sanitiseLabelValue("0"),
			},
		},
	}
	stale := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stale-pod",
			Namespace: testKubeNamespace,
			Labels: map[string]string{
				labelKeyRunID:      run.sanitiseLabelValue("default-test-run-id"),
				labelKeyActivityID: run.sanitiseLabelValue("0"),
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: testJobAPIVersion,
				Kind:       testJobKind,
				Name:       "ancient-job",
				UID:        types.UID("ancient-job-uid"),
				Controller: utils.Ptr(true),
			}},
		},
	}

	reactJobCreate(t, fakeClient, jobReactorOpts{
		status: &batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}},
		},
		// seedPod is false: the only label-matching pods belong to a
		// different Job or have no owner at all.
		extraPods: []corev1.Pod{orphan, stale},
	})

	_, err := runK8sActivity(t, fakeClient, makeContainerTask(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pods found for job",
		"label-matching pods without our Job's controllerRef must not be selected")
}

func TestRunKubernetesJob_JobNamePopulatedFromGenerateName(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	completedJobReactor(t, fakeClient)

	_, err := runK8sActivity(t, fakeClient, makeContainerTask(), "")
	require.NoError(t, err)

	// Verify the delete was called with the deterministic name set by the
	// reactor; this confirms the name plumbed through from create to
	// waitForKubernetesJobCompletion to deleteJob.
	var deleted clienttesting.DeleteAction
	for _, action := range fakeClient.Actions() {
		if action.GetVerb() == testVerbDelete && action.GetResource().Resource == testResourceJobs {
			deleted = action.(clienttesting.DeleteAction)
			break
		}
	}
	require.NotNil(t, deleted)
	assert.Equal(t, jobNamePrefix+"abc12", deleted.GetName())
}

// TestGetJobLogs_UsesSanitisedContainerNameForLogRequest is the regression
// guard for the review finding: a workflow container name that is not a
// valid DNS-1123 label must not flow into either the Pod's Container.Name or
// the Container field of PodLogOptions. If getJobLogs asked for logs by the
// raw name on a real cluster, the apiserver would reject the request because
// no container in the Pod has that name.
//
// The fake clientset records the PodLogOptions on the get/pods/log action,
// which lets the test assert the exact value passed in.
func TestGetJobLogs_UsesSanitisedContainerNameForLogRequest(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	completedJobReactor(t, fakeClient)

	const badContainer = "BAD/Container Name_value"
	task := makeContainerTask()
	task.Run.Container.Name = badContainer

	out, err := runK8sActivity(t, fakeClient, task, testKubeServiceAccount)
	require.NoError(t, err, "awkward raw container name must not break log retrieval")
	assert.Equal(t, "fake logs", out)

	// Walk the recorded actions and find the GetLogs request; its Value is
	// the PodLogOptions the production code constructed.
	var logOpts *corev1.PodLogOptions
	for _, action := range fakeClient.Actions() {
		if action.GetVerb() != "get" ||
			action.GetResource().Resource != "pods" ||
			action.GetSubresource() != "log" {
			continue
		}
		generic, ok := action.(clienttesting.GenericAction)
		require.True(t, ok, "GetLogs action should be a GenericAction")
		opts, ok := generic.GetValue().(*corev1.PodLogOptions)
		require.True(t, ok, "GetLogs Value should be *corev1.PodLogOptions")
		logOpts = opts
		break
	}
	require.NotNil(t, logOpts, "expected a get/pods/log action")

	require.Empty(t, validation.IsDNS1123Label(logOpts.Container),
		"PodLogOptions.Container %q must be a valid DNS-1123 label",
		logOpts.Container)
	assert.True(t, strings.HasPrefix(logOpts.Container, sanitisedContainerNamePrefix),
		"PodLogOptions.Container should use sanitised name, got %q",
		logOpts.Container)
	assert.NotEqual(t, badContainer, logOpts.Container,
		"raw workflow container name must not be passed to GetLogs")

	// And the same sanitised value must have landed in the Pod spec, so the
	// Container reference in PodLogOptions resolves on a real cluster.
	var createdJob *batchv1.Job
	for _, action := range fakeClient.Actions() {
		if action.GetVerb() == testVerbCreate && action.GetResource().Resource == testResourceJobs {
			createdJob = action.(clienttesting.CreateAction).GetObject().(*batchv1.Job)
			break
		}
	}
	require.NotNil(t, createdJob)
	require.Len(t, createdJob.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, logOpts.Container, createdJob.Spec.Template.Spec.Containers[0].Name,
		"Container.Name in the Pod must match Container field in PodLogOptions")

	// The raw value must still survive in the annotation for debugging.
	assert.Equal(t, badContainer, createdJob.Annotations[annotationKeyContainerName])
}
