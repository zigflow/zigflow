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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func init() {
	Registry = append(Registry, &Run{})
}

const defaultTTLThreshold = 2 * time.Second

type Run struct{}

func (r *Run) CallContainerActivity(
	ctx context.Context,
	task *model.RunTask,
	input any,
	state *utils.State,
	namespace, runtime, serviceAccount string,
) (any, error) {
	logger := activity.GetLogger(ctx)

	if task.Run.Container.Name == "" {
		n := uuid.NewString()
		logger.Debug("Container name not set", "name", n)
		task.Run.Container.Name = n
	}

	if runtime == "kubernetes" {
		// Use Kubernetes
		logger.Debug("Running call Kubernetes job activity")

		return r.runKubernetesJob(ctx, task, state, namespace, serviceAccount)
	}

	// Default to Docker
	logger.Debug("Running call Docker container activity")

	return r.runDockerCommand(ctx, task, state)
}

func (r *Run) CallScriptActivity(ctx context.Context, task *model.RunTask, input any, state *utils.State) (any, error) {
	command := make([]string, 0)
	var file string

	logger := activity.GetLogger(ctx)
	logger.Debug("Running call script activity")

	logger.Debug("Creating temporary directory")
	dir, err := os.MkdirTemp("", "script")
	if err != nil {
		logger.Error("Error making temp dir", "error", err)
		return nil, fmt.Errorf("error making temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			logger.Warn("Generated script not deleted", "dir", dir, "error", err)
		}
	}()

	lang := task.Run.Script.Language
	logger.Debug("Detecting script language", "language", lang)
	switch lang {
	case "js":
		command = append(command, "node")
		file = "script.js"
	case "python":
		command = append(command, "python")
		file = "script.py"
	default:
		logger.Error("Unknown script language", "language", lang)
		return nil, fmt.Errorf("unknown script language: %s", lang)
	}

	fname := filepath.Join(dir, file)
	logger.Debug("Writing script to disk", "file", fname)
	command = append(command, fname)
	if err := os.WriteFile(fname, []byte(*task.Run.Script.InlineCode), 0o600); err != nil {
		logger.Error("Error writing script to disk", "file", fname, "error", err)
		return nil, fmt.Errorf("error writing code to script: %w", err)
	}

	return r.runExecCommand(
		ctx,
		command,
		task.Run.Script.Arguments,
		task.Run.Script.Environment,
		state,
		dir,
		task.GetBase(),
	)
}

func (r *Run) CallShellActivity(ctx context.Context, task *model.RunTask, input any, state *utils.State) (any, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("Running call script activity")

	return r.runExecCommand(
		ctx,
		[]string{task.Run.Shell.Command},
		task.Run.Shell.Arguments,
		task.Run.Shell.Environment,
		state,
		"",
		task.GetBase(),
	)
}

func (r *Run) authenticateKubernetes() (*kubernetes.Clientset, error) {
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

func (r *Run) deployKubernetesJob(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	task *model.RunTask,
	namespace, serviceAccount string,
	state *utils.State,
) (*batchv1.Job, error) {
	logger := activity.GetLogger(ctx)
	info := activity.GetInfo(ctx)

	state = state.Clone().AddActivityInfo(ctx)

	logger.Debug("Interpolating command arguments and envvars")
	d, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(map[string]any{
		"args": task.Run.Container.Arguments,
		"env":  task.Run.Container.Environment,
	}), nil, state)
	if err != nil {
		return nil, fmt.Errorf("error traversing task parameters: %w", err)
	}
	parsed := d.(map[string]any)
	envvars := parsed["env"].(map[string]string)

	fmt.Printf("%+v\n", task.Run.Container.Environment)
	fmt.Printf("%+v\n", parsed["env"])

	args := make([]string, 0)
	for _, a := range parsed["args"].([]any) {
		args = append(args, a.(string))
	}

	l := map[string]string{
		"app.kubernetes.io/name":      "zigflow",
		"app.kubernetes.io/component": "run-task",
	}
	for k, v := range map[string]string{
		"workflowId": info.WorkflowExecution.ID,
		"runId":      info.WorkflowExecution.RunID,
		"activityId": info.ActivityID,
		"name":       task.Run.Container.Name,
	} {
		l[fmt.Sprintf("zigflow.dev/%s", k)] = v
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "zigflow-run-task-",
			Namespace:    namespace,
			Labels:       l,
		},
		Spec: batchv1.JobSpec{
			Completions:  utils.Ptr[int32](1),
			Parallelism:  utils.Ptr[int32](1),
			BackoffLimit: utils.Ptr[int32](0), // Let Temporal handle the retry
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: l,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: serviceAccount,
					Containers:         []corev1.Container{},
				},
			},
		},
	}

	if task.Run.Container.Lifetime != nil && task.Run.Container.Lifetime.Cleanup == "eventually" {
		// Destroy after a period of time
		after := utils.ToDuration(task.Run.Container.Lifetime.After)
		if after >= defaultTTLThreshold {
			job.Spec.TTLSecondsAfterFinished = utils.Ptr(int32(after.Seconds()))
		}
	}

	// Build the container object
	container := corev1.Container{
		Name:            task.Run.Container.Name,
		Image:           task.Run.Container.Image,
		ImagePullPolicy: corev1.PullAlways, // Keep consistent with Docker, but schema should support this
		Command:         []string{},
		Args:            args,
		Env:             []corev1.EnvVar{},
	}
	if c := task.Run.Container.Command; c != "" {
		container.Command = append(container.Command, c)
	}
	fmt.Println("---")
	fmt.Printf("%+v\n", envvars)
	fmt.Println("---")

	if envs := envvars; envs != nil {
		for k, v := range envs {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}

	job.Spec.Template.Spec.Containers = append(job.Spec.Template.Spec.Containers, container)

	j, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"Failed to create job",
			"Job creation error",
			err,
		)
	}

	// Return the actual job - properties may be trimmed/defaulted
	return j, nil
}

func (r *Run) getJobLogs(
	ctx context.Context,
	clientset *kubernetes.Clientset,
	task *model.RunTask,
	namespace, runId, activityID string,
) (*string, error) {
	logger := activity.GetLogger(ctx)
	labelSelector := labels.Set{
		"zigflow.dev/runId":      runId,
		"zigflow.dev/activityId": activityID,
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods for job: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods found for job")
	}

	pod := pods.Items[0]

	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: task.Run.Container.Name,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to stream logs from pod %q:  %w", pod.Name, err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Warn("Log streamer failed to close", "error", err)
		}
	}()

	logs, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read log stream: %w", err)
	}

	return utils.Ptr(string(logs)), nil
}

// runDockerCommand runs the script on a container using the Docker runtime
func (r *Run) runDockerCommand(ctx context.Context, task *model.RunTask, state *utils.State) (any, error) {
	info := activity.GetInfo(ctx)

	if _, err := exec.LookPath("docker"); err != nil {
		return nil, temporal.NewNonRetryableApplicationError("Docker not installed", "container", err)
	}

	cmd := []string{
		"docker",
		"run",
		"--pull=always",
		fmt.Sprintf("--label=workflowId=%s", info.WorkflowExecution.ID),
		fmt.Sprintf("--label=runId=%s", info.WorkflowExecution.RunID),
		fmt.Sprintf("--label=activityId=%s", info.ActivityID),
		fmt.Sprintf("--name=%s", task.Run.Container.Name),
	}

	if c := task.Run.Container.Command; c != "" {
		cmd = append(cmd, fmt.Sprintf("--entrypoint=%s", c))
	}

	if task.Run.Container.Lifetime == nil || task.Run.Container.Lifetime.Cleanup == "always" {
		cmd = append(cmd, "--rm")
	}

	if envs := task.Run.Container.Environment; envs != nil {
		for k, v := range envs {
			cmd = append(cmd, fmt.Sprintf("--env=%s=%s", k, v))
		}
	}

	if vols := task.Run.Container.Volumes; vols != nil {
		for k, remote := range vols {
			local, err := filepath.Abs(k)
			if err != nil {
				return nil, fmt.Errorf("error getting volume absolute path: %w", err)
			}

			cmd = append(cmd, fmt.Sprintf("--volume=%s:%s", local, remote))
		}
	}
	// Add in the image
	cmd = append(cmd, task.Run.Container.Image)

	// Add in arguments
	cmd = append(cmd, task.Run.Container.Arguments...)

	return r.runExecCommand(ctx, []string{cmd[0]}, &model.RunArguments{Value: cmd[1:]}, nil, state, "", task.GetBase())
}

// runExecCommand a general purpose function to build and execute a command in an activity
func (r *Run) runExecCommand(
	ctx context.Context,
	command []string,
	args *model.RunArguments,
	env map[string]string,
	state *utils.State,
	dir string,
	task *model.TaskBase,
) (any, error) {
	logger := activity.GetLogger(ctx)

	stopHeartbeat := metadata.StartActivityHeartbeat(ctx, task)
	defer stopHeartbeat()

	if args == nil {
		args = &model.RunArguments{}
	}
	if env == nil {
		env = map[string]string{}
	}

	state = state.Clone().AddActivityInfo(ctx)

	logger.Debug("Interpolating command arguments and envvars")
	d, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(map[string]any{
		"args": swUtil.DeepCloneValue(args.AsSlice()),
		"env":  swUtil.DeepCloneValue(env),
	}), nil, state)
	if err != nil {
		return nil, fmt.Errorf("error traversing task parameters: %w", err)
	}
	data := d.(map[string]any)

	// Cast the arg to a string
	for _, v := range data["args"].([]any) {
		command = append(command, fmt.Sprintf("%v", v))
	}

	envvars := os.Environ()
	for k, v := range data["env"].(map[string]string) {
		envvars = append(envvars, fmt.Sprintf("%s=%v", k, v))
	}

	var stderr bytes.Buffer
	var stdout bytes.Buffer
	logWriter := utils.LogWriter{
		Logger: logger,
		Level:  "info",
		Msg:    "Run task response",
	}

	//nolint:gosec // Allow dynamic commands
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = envvars
	cmd.Stdout = io.MultiWriter(&stdout, logWriter.AddFields([]any{"type", "stdout"}))
	cmd.Stderr = io.MultiWriter(&stderr, logWriter.AddFields([]any{"type", "stderr"}))

	if dir != "" {
		cmd.Dir = dir
	}

	logger.Info("Running command on worker", "command", command)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// The command received an exit code above 0 - return as-is
			logger.Error("Shell error",
				"error", err,
				"command", command,
				"stderr", r.stdToString(stdout),
				"stdout", r.stdToString(stdout),
			)
			return nil, temporal.NewApplicationErrorWithCause(
				"Error calling command",
				"command",
				exitErr,
				map[string]any{
					"command": command,
					"stderr":  r.stdToString(stderr),
					"stdout":  r.stdToString(stdout),
				},
			)
		}
		logger.Error("Error running command", "error", err)
		return nil, fmt.Errorf("error running command: %w", err)
	}

	return r.stdToString(stdout), nil
}

func (r *Run) deleteJob(ctx context.Context, clientset *kubernetes.Clientset, task *model.RunTask, job *batchv1.Job) error {
	var shouldDelete bool
	if task.Run.Container.Lifetime == nil {
		shouldDelete = true
	} else {
		switch task.Run.Container.Lifetime.Cleanup {
		case "always":
			// Always delete
			shouldDelete = true
		case "eventually":
			// Destroy after a period of time
			after := utils.ToDuration(task.Run.Container.Lifetime.After)

			// TTL not set in job config
			shouldDelete = after < defaultTTLThreshold
		}
	}

	if !shouldDelete {
		return nil
	}

	if err := clientset.BatchV1().
		Jobs(job.ObjectMeta.Namespace).
		Delete(ctx, job.Name, metav1.DeleteOptions{
			PropagationPolicy: utils.Ptr(metav1.DeletePropagationForeground),
		}); err != nil {
		return fmt.Errorf("failed to delete job %q: %w", job.Name, err)
	}

	return nil
}

func (r *Run) runKubernetesJob(
	ctx context.Context,
	task *model.RunTask,
	state *utils.State,
	namespace, serviceAccount string,
) (any, error) {
	logger := activity.GetLogger(ctx)
	info := activity.GetInfo(ctx)

	stopHeartbeat := metadata.StartActivityHeartbeat(ctx, task.GetBase())
	defer stopHeartbeat()

	clientset, err := r.authenticateKubernetes()
	if err != nil {
		logger.Error("Unable to authenticate to Kubernetes", "error", err)
		return nil, err
	}

	logger.Debug("Creating Kubernetes Job", "name", task.Run.Container.Name)
	job, err := r.deployKubernetesJob(ctx, clientset, task, namespace, serviceAccount, state)
	if err != nil {
		return nil, err
	}

	jobName := job.Name
	runId := info.WorkflowExecution.RunID
	activityID := info.ActivityID

	// Update the job with the desired cleanup config
	defer func() {
		if err := r.deleteJob(context.WithoutCancel(ctx), clientset, task, job); err != nil {
			logger.Error("error deleting kubernetes job", "error", err)
		}
	}()

	logger.Debug("Waiting for Job completion", "name", jobName)
	if err := r.waitForKubernetesJobCompletion(ctx, clientset, namespace, jobName, time.Second); err != nil {
		logger.Error("Job did not complete successfully", "error", err)
		return nil, fmt.Errorf("job did not complete successfully: %w", err)
	}

	logger.Debug("Retrieving Job logs", "name", jobName)
	logs, err := r.getJobLogs(ctx, clientset, task, namespace, runId, activityID)
	if err != nil {
		logger.Error("Error retrieving Job logs", "name", jobName, "error", err)
		return nil, fmt.Errorf("error retrieving job logs: %w", err)
	}

	return *logs, nil
}

func (r *Run) stdToString(std bytes.Buffer) string {
	return strings.TrimSpace(std.String())
}

func (r *Run) waitForKubernetesJobCompletion(
	ctx context.Context,
	clientset *kubernetes.Clientset,
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
