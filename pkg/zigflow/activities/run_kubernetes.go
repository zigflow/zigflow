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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"github.com/zigflow/zigflow/pkg/zigflow/models"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// defaultKubernetesPollInterval is the cadence at which the Kubernetes runtime
// polls a Job for completion. Kept short enough to be responsive but long
// enough that fake-client-based unit tests still see useful behaviour without
// burning CPU.
const defaultKubernetesPollInterval = time.Second

// defaultTTLThreshold is the minimum lifetime.after value at which the
// Kubernetes runtime will set TTLSecondsAfterFinished on a Job. Anything
// shorter is treated as effectively unset.
const defaultTTLThreshold = 2 * time.Second

// lifetimeCleanup* mirror the cleanup values defined by the Serverless
// Workflow ContainerLifetime schema.
const (
	lifetimeCleanupAlways     = "always"
	lifetimeCleanupEventually = "eventually"
	lifetimeCleanupNever      = "never"
)

// argsKey and envKey are the field names used when collecting the
// container's args and env for runtime-expression evaluation.
const (
	argsKey = "args"
	envKey  = "env"
)

// labelKeyRunID, labelKeyActivityID and labelKeyContainerName are the labels
// used to identify Jobs and Pods created for a particular activity invocation.
// They MUST stay consistent with the selector used by getJobLogs so pods are
// reliably located after a Job starts. Values are passed through
// sanitiseLabelValue so user-controlled inputs cannot break Job creation with
// label validation errors; the raw values are preserved under the matching
// annotation keys below.
const (
	labelKeyAppName       = "app.kubernetes.io/name"
	labelKeyAppComponent  = "app.kubernetes.io/component"
	labelKeyWorkflowID    = "zigflow.dev/workflowId"
	labelKeyRunID         = "zigflow.dev/runId"
	labelKeyActivityID    = "zigflow.dev/activityId"
	labelKeyContainerName = "zigflow.dev/name"

	// annotationKey* mirror the labelKey* set but carry the raw, unsanitised
	// values. Annotations have a much larger size budget than labels and no
	// character-set restriction, so they are the right place for arbitrary
	// Temporal IDs and user-supplied container names that may not be
	// Kubernetes-label-safe.
	annotationKeyWorkflowID    = "zigflow.dev/workflowId"
	annotationKeyRunID         = "zigflow.dev/runId"
	annotationKeyActivityID    = "zigflow.dev/activityId"
	annotationKeyContainerName = "zigflow.dev/name"

	labelValueAppName      = "zigflow"
	labelValueAppComponent = "run-task"

	jobNamePrefix = "zigflow-run-task-"

	// hashedLabelPrefix tags label values that are not the original input
	// but a deterministic short hash of it. Selectors hash the same way, so
	// the prefix is purely a hint to operators reading kubectl output.
	hashedLabelPrefix = "h-"

	// hashedLabelHexLen is the number of hex characters kept from a SHA-256
	// digest for a label value. Together with hashedLabelPrefix this fits
	// well under the 63-char Kubernetes label-value limit and gives 64 bits
	// of collision resistance, which is far more than needed for
	// run/activity identifiers.
	hashedLabelHexLen = 16

	// sanitisedContainerNamePrefix tags pod-spec container names that are a
	// deterministic short hash of a raw workflow container name rather than
	// the raw value. Kubernetes container names must be DNS-1123 labels, so
	// workflow names that are uppercase, contain underscores/slashes/spaces
	// or exceed 63 chars are replaced with a label-safe value. The full raw
	// workflow container name is preserved in annotationKeyContainerName so
	// debugging tools can still recover it.
	sanitisedContainerNamePrefix = "container-"
)

// ContainerRuntime identifies the runtime used to execute run.container tasks.
type ContainerRuntime string

const (
	ContainerRuntimeDocker     ContainerRuntime = "docker"
	ContainerRuntimeKubernetes ContainerRuntime = "kubernetes"
)

// ValidContainerRuntimes lists every accepted container runtime. The set is
// validated in PreRunE so invalid values are rejected before any worker starts.
var ValidContainerRuntimes = map[ContainerRuntime]struct{}{
	ContainerRuntimeDocker:     {},
	ContainerRuntimeKubernetes: {},
}

// kubernetesClientFactory builds a Kubernetes clientset. It is a
// package-level variable so tests can substitute a fake clientset without
// requiring a real cluster or kubeconfig.
var kubernetesClientFactory = defaultKubernetesClientFactory

// defaultKubernetesClientFactory authenticates against the Kubernetes API
// using the in-cluster config. It is exported as a variable above so tests can
// replace it; do not call it directly outside that var initialiser.
func defaultKubernetesClientFactory() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"Failed to build in-cluster config",
			"In-cluster config error",
			err,
		)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"Failed to create clientset",
			"Clientset creation error",
			err,
		)
	}

	return clientset, nil
}

// kubernetesRuntimeOptions bundles the seams that need to be substituted in
// tests: the clientset factory and the poll interval used while waiting for a
// Job to complete. Production code uses defaultKubernetesRuntimeOptions().
type kubernetesRuntimeOptions struct {
	clientFactory func() (kubernetes.Interface, error)
	pollInterval  time.Duration
}

func defaultKubernetesRuntimeOptions() kubernetesRuntimeOptions {
	return kubernetesRuntimeOptions{
		clientFactory: kubernetesClientFactory,
		pollInterval:  defaultKubernetesPollInterval,
	}
}

// validateKubernetesContainer rejects run.container configurations the
// Kubernetes runtime cannot honour. Volumes are documented and supported by the
// Docker runtime but are not yet implemented for Kubernetes; silently dropping
// them would produce a Job that does not match the workflow definition, so the
// runtime fails fast with a non-retryable error instead. This check runs at the
// top of buildJobSpec so the rejection happens before any Job is created.
func (r *Run) validateKubernetesContainer(task *model.RunTask) error {
	if len(task.Run.Container.Volumes) > 0 {
		return temporal.NewNonRetryableApplicationError(
			"run.container.volumes are not currently supported by the Kubernetes runtime",
			"Unsupported container configuration",
			nil,
		)
	}
	return nil
}

// evaluateContainerArgs evaluates runtime expressions in the supplied container
// arguments and returns the resulting positional args. An omitted or empty
// input produces a nil slice, which the corev1.Container honours as "no args"
// without a defensive allocation on the caller side.
func (r *Run) evaluateContainerArgs(args []string, state *utils.State) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}

	d, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(map[string]any{
		argsKey: swUtil.DeepCloneValue(args),
	}), nil, state)
	if err != nil {
		return nil, fmt.Errorf("error traversing container arguments: %w", err)
	}

	parsed, ok := d.(map[string]any)
	if !ok {
		return nil, nil
	}
	items, ok := parsed[argsKey].([]any)
	if !ok {
		return nil, nil
	}

	out := make([]string, 0, len(items))
	for _, a := range items {
		out = append(out, fmt.Sprintf("%v", a))
	}
	return out, nil
}

// evaluateContainerEnv evaluates runtime expressions in the supplied container
// environment map and returns the resulting []corev1.EnvVar. An omitted or
// empty input produces a nil slice so the resulting container spec carries no
// env entries at all.
func (r *Run) evaluateContainerEnv(env map[string]string, state *utils.State) ([]corev1.EnvVar, error) {
	if len(env) == 0 {
		return nil, nil
	}

	d, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(map[string]any{
		envKey: swUtil.DeepCloneValue(env),
	}), nil, state)
	if err != nil {
		return nil, fmt.Errorf("error traversing container environment: %w", err)
	}

	parsed, ok := d.(map[string]any)
	if !ok {
		return nil, nil
	}
	evaluated, ok := parsed[envKey].(map[string]string)
	if !ok {
		return nil, nil
	}

	out := make([]corev1.EnvVar, 0, len(evaluated))
	for k, v := range evaluated {
		out = append(out, corev1.EnvVar{Name: k, Value: v})
	}
	return out, nil
}

// buildJobSpec assembles the batchv1.Job for a run.container task. It is
// separated from the API call so tests can assert against the spec directly
// without needing to exercise the clientset.
func (r *Run) buildJobSpec(
	ctx context.Context,
	task *model.RunTask,
	namespace, serviceAccount string,
	state *utils.State,
) (*batchv1.Job, error) {
	logger := activity.GetLogger(ctx)
	info := activity.GetInfo(ctx)

	if err := r.validateKubernetesContainer(task); err != nil {
		return nil, err
	}

	state = state.Clone().AddActivityInfo(ctx)

	logger.Debug("Interpolating command arguments and envvars")
	args, err := r.evaluateContainerArgs(task.Run.Container.Arguments, state)
	if err != nil {
		return nil, err
	}
	envvars, err := r.evaluateContainerEnv(task.Run.Container.Environment, state)
	if err != nil {
		return nil, err
	}

	// Labels carry sanitised values so Job creation never fails because of a
	// long Temporal ID or a container name with characters Kubernetes does
	// not allow. Annotations carry the raw values for debugging via kubectl
	// describe / events. Job and Pod template metadata share both maps so a
	// selector built from one matches resources rendered from the other.
	l := map[string]string{
		labelKeyAppName:       labelValueAppName,
		labelKeyAppComponent:  labelValueAppComponent,
		labelKeyWorkflowID:    r.sanitiseLabelValue(info.WorkflowExecution.ID),
		labelKeyRunID:         r.sanitiseLabelValue(info.WorkflowExecution.RunID),
		labelKeyActivityID:    r.sanitiseLabelValue(info.ActivityID),
		labelKeyContainerName: r.sanitiseLabelValue(task.Run.Container.Name),
	}
	a := map[string]string{
		annotationKeyWorkflowID:    info.WorkflowExecution.ID,
		annotationKeyRunID:         info.WorkflowExecution.RunID,
		annotationKeyActivityID:    info.ActivityID,
		annotationKeyContainerName: task.Run.Container.Name,
	}

	container := corev1.Container{
		Name:            r.sanitiseContainerName(task.Run.Container.Name),
		Image:           task.Run.Container.Image,
		ImagePullPolicy: r.pullPolicyToK8s(task.Run.Container.PullPolicy),
		Args:            args,
		Env:             envvars,
	}
	if c := task.Run.Container.Command; c != "" {
		container.Command = []string{c}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: jobNamePrefix,
			Namespace:    namespace,
			Labels:       l,
			Annotations:  a,
		},
		Spec: batchv1.JobSpec{
			Completions:  utils.Ptr[int32](1),
			Parallelism:  utils.Ptr[int32](1),
			BackoffLimit: utils.Ptr[int32](0), // Let Temporal Activity retries handle failures
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      l,
					Annotations: a,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: serviceAccount,
					// Workflow-defined containers should not be able to call
					// the Kubernetes API by default. The Helm chart's
					// workload ServiceAccount also disables automount, but
					// pinning it here means the runtime is safe even when a
					// caller wires it up against a token-mounted account.
					AutomountServiceAccountToken: utils.Ptr(false),
					Containers:                   []corev1.Container{container},
				},
			},
		},
	}

	if task.Run.Container.Lifetime != nil && task.Run.Container.Lifetime.Cleanup == lifetimeCleanupEventually {
		after := utils.ToDuration(task.Run.Container.Lifetime.After)
		if after >= defaultTTLThreshold {
			job.Spec.TTLSecondsAfterFinished = utils.Ptr(int32(after.Seconds()))
		}
	}

	return job, nil
}

func (r *Run) deleteJob(ctx context.Context, clientset kubernetes.Interface, task *model.RunTask, job *batchv1.Job) error {
	shouldDelete := true
	if task.Run.Container.Lifetime != nil {
		switch task.Run.Container.Lifetime.Cleanup {
		case lifetimeCleanupAlways:
			shouldDelete = true
		case lifetimeCleanupEventually:
			// If the requested lifetime is below the TTL threshold the Job
			// is deleted inline here; otherwise the cluster-side
			// TTLSecondsAfterFinished controller takes over and we leave it
			// alone.
			after := utils.ToDuration(task.Run.Container.Lifetime.After)
			shouldDelete = after < defaultTTLThreshold
		case lifetimeCleanupNever:
			shouldDelete = false
		}
	}

	if !shouldDelete {
		return nil
	}

	if err := clientset.BatchV1().
		Jobs(job.Namespace).
		Delete(ctx, job.Name, metav1.DeleteOptions{
			PropagationPolicy: utils.Ptr(metav1.DeletePropagationForeground),
		}); err != nil {
		return fmt.Errorf("failed to delete job %q: %w", job.Name, err)
	}

	return nil
}

func (r *Run) deployKubernetesJob(
	ctx context.Context,
	clientset kubernetes.Interface,
	task *model.RunTask,
	namespace, serviceAccount string,
	state *utils.State,
) (*batchv1.Job, error) {
	job, err := r.buildJobSpec(ctx, task, namespace, serviceAccount, state)
	if err != nil {
		return nil, err
	}

	j, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"Failed to create job",
			"Job creation error",
			err,
		)
	}

	return j, nil
}

// getJobLogs returns the logs from the pod owned by the supplied Job. It
// narrows the API query with the run/activity correlation labels (cheap
// server-side filter), then filters the returned pods down to those whose
// controllerRef matches the Job's UID. The selector alone is not enough:
// retained Jobs or stale pods from earlier activity attempts can share the
// same Temporal run/activity labels, and we must never read logs from a pod
// that does not belong to this exact Job instance. When more than one pod
// owned by the Job is present, the most recently created one wins; that
// matches the Job controller's own retry semantics.
func (r *Run) getJobLogs(
	ctx context.Context,
	clientset kubernetes.Interface,
	task *model.RunTask,
	job *batchv1.Job,
) (string, error) {
	logger := activity.GetLogger(ctx)

	tmplLabels := job.Spec.Template.Labels
	selector := labels.Set{
		labelKeyRunID:      tmplLabels[labelKeyRunID],
		labelKeyActivityID: tmplLabels[labelKeyActivityID],
	}

	pods, err := clientset.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods for job: %w", err)
	}

	pod := r.newestPodControlledBy(pods.Items, job)
	if pod == nil {
		return "", fmt.Errorf("no pods found for job %q", job.Name)
	}

	// Container must match the DNS-1123 name written into the Pod by
	// buildJobSpec, not the raw workflow name; sanitiseContainerName is
	// deterministic so passing the same task.Run.Container.Name reproduces
	// it exactly.
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: r.sanitiseContainerName(task.Run.Container.Name),
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to stream logs from pod %q: %w", pod.Name, err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Warn("Log streamer failed to close", "error", err)
		}
	}()

	logs, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("failed to read log stream: %w", err)
	}

	return strings.TrimSpace(string(logs)), nil
}

// newestPodControlledBy returns the most recently created pod in pods whose
// controllerRef refers to job, or nil if none match. Ties on CreationTimestamp
// are broken by pod name to keep selection deterministic across reruns of
// the same workload; in practice the Job controller never creates two pods
// at the exact same instant so the tie-breaker is rarely exercised.
func (r *Run) newestPodControlledBy(pods []corev1.Pod, job *batchv1.Job) *corev1.Pod {
	var newest *corev1.Pod
	for i := range pods {
		p := &pods[i]
		if !metav1.IsControlledBy(p, job) {
			continue
		}
		switch {
		case newest == nil:
			newest = p
		case p.CreationTimestamp.After(newest.CreationTimestamp.Time):
			newest = p
		case p.CreationTimestamp.Equal(&newest.CreationTimestamp) && p.Name > newest.Name:
			newest = p
		}
	}
	return newest
}

// runKubernetesJob executes a run.container task against Kubernetes using the
// in-cluster client. Tests use runKubernetesJobWithOptions with a fake client
// to exercise the same logic without a real cluster.
func (r *Run) runKubernetesJob(
	ctx context.Context,
	task *model.RunTask,
	state *utils.State,
	namespace, serviceAccount string,
) (any, error) {
	return r.runKubernetesJobWithOptions(ctx, task, state, namespace, serviceAccount, defaultKubernetesRuntimeOptions())
}

func (r *Run) runKubernetesJobWithOptions(
	ctx context.Context,
	task *model.RunTask,
	state *utils.State,
	namespace, serviceAccount string,
	opts kubernetesRuntimeOptions,
) (any, error) {
	logger := activity.GetLogger(ctx)

	stopHeartbeat := metadata.StartActivityHeartbeat(ctx, task.GetBase())
	defer stopHeartbeat()

	clientset, err := opts.clientFactory()
	if err != nil {
		logger.Error("Unable to authenticate to Kubernetes", "error", err)
		return nil, err
	}

	logger.Debug("Creating Kubernetes Job", "name", task.Run.Container.Name)
	job, err := r.deployKubernetesJob(ctx, clientset, task, namespace, serviceAccount, state)
	if err != nil {
		return nil, err
	}

	// context.WithoutCancel ensures the delete runs even when the activity
	// context has already been cancelled. Failing to delete is logged but does
	// not mask the original outcome.
	defer func() {
		if delErr := r.deleteJob(context.WithoutCancel(ctx), clientset, task, job); delErr != nil {
			logger.Error("error deleting kubernetes job", "error", delErr)
		}
	}()

	logger.Debug("Waiting for Job completion", "name", job.Name)
	if err := r.waitForKubernetesJobCompletion(ctx, clientset, job.Namespace, job.Name, opts.pollInterval); err != nil {
		logger.Error("Job did not complete successfully", "error", err)
		return nil, fmt.Errorf("job did not complete successfully: %w", err)
	}

	logger.Debug("Retrieving Job logs", "name", job.Name)
	logs, err := r.getJobLogs(ctx, clientset, task, job)
	if err != nil {
		logger.Error("Error retrieving Job logs", "name", job.Name, "error", err)
		return nil, fmt.Errorf("error retrieving job logs: %w", err)
	}

	return logs, nil
}

// sanitiseLabelValue returns a Kubernetes-safe label value derived from v.
//
// If v is already a valid Kubernetes label value (matches the alphanumeric
// /._- regex, starts/ends with alphanumeric, and is at most 63 chars) it is
// returned unchanged. Otherwise the function returns a deterministic short
// hash so distinct inputs do not collapse to the same label, and so the same
// raw value always produces the same selector-safe label. The empty input
// case is treated as invalid so selectors targeting it can still match
// reliably.
//
// The full raw value is preserved as an annotation by the caller.
func (r *Run) sanitiseLabelValue(v string) string {
	if v != "" && len(validation.IsValidLabelValue(v)) == 0 {
		return v
	}
	digest := sha256.Sum256([]byte(v))
	return hashedLabelPrefix + hex.EncodeToString(digest[:])[:hashedLabelHexLen]
}

// sanitiseContainerName returns a Kubernetes-safe container name derived from
// v.
//
// Kubernetes container names must be DNS-1123 labels: lowercase alphanumeric
// and '-', start and end with an alphanumeric, at most 63 characters.
// Workflow container names come from user-controlled YAML and may be
// uppercase, contain underscores, slashes, spaces, or exceed 63 chars; using
// them verbatim as corev1.Container.Name would make otherwise valid workflows
// fail Job creation at runtime.
//
// If v already satisfies the DNS-1123 label rules it is returned unchanged so
// readable names survive into kubectl output. Otherwise a deterministic short
// hash is returned, prefixed with sanitisedContainerNamePrefix; the prefix
// keeps the result a valid DNS-1123 label even when v starts with a digit and
// gives operators a hint that the value is derived rather than user-supplied.
// The full raw workflow container name is preserved by the caller in
// annotationKeyContainerName.
func (r *Run) sanitiseContainerName(v string) string {
	if v != "" && len(validation.IsDNS1123Label(v)) == 0 {
		return v
	}
	digest := sha256.Sum256([]byte(v))
	return sanitisedContainerNamePrefix + hex.EncodeToString(digest[:])[:hashedLabelHexLen]
}

func (r *Run) waitForKubernetesJobCompletion(
	ctx context.Context,
	clientset kubernetes.Interface,
	namespace, jobName string,
	interval time.Duration,
) error {
	return wait.PollUntilContextCancel(
		ctx,
		interval,
		true,
		func(ctx context.Context) (bool, error) {
			job, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			for _, condition := range job.Status.Conditions {
				switch condition.Type {
				case batchv1.JobComplete:
					if condition.Status == corev1.ConditionTrue {
						return true, nil
					}
				case batchv1.JobFailed:
					if condition.Status == corev1.ConditionTrue {
						return false, fmt.Errorf("job failed: %s", condition.Message)
					}
				}
			}
			return false, nil
		},
	)
}

func (r *Run) pullPolicyToK8s(p string) corev1.PullPolicy {
	switch p {
	case models.PullAlways:
		return corev1.PullAlways
	case models.PullNever:
		return corev1.PullNever
	case models.PullIfNotPresent:
		return corev1.PullIfNotPresent
	default:
		return corev1.PullIfNotPresent
	}
}
