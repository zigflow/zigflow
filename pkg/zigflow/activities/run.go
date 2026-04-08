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

	"github.com/google/uuid"
	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

func init() {
	Registry = append(Registry, &Run{})
}

type Run struct{}

func (r *Run) CallContainerActivity(ctx context.Context, task *model.RunTask, input any, state *utils.State) (any, error) {
	logger := activity.GetLogger(ctx)
	// @todo(sje): support Kubernetes (and other container runtimes) in addition to Docker #181
	logger.Debug("Running call Docker container activity")

	if task.Run.Container.Name == "" {
		n := uuid.NewString()
		logger.Debug("Container name not set", "name", n)
		task.Run.Container.Name = n
	}

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

// runDockerCommand runs the script on a container using the Docker runtime
func (r *Run) runDockerCommand(ctx context.Context, task *model.RunTask, state *utils.State) (any, error) {
	info := activity.GetInfo(ctx)

	if _, err := exec.LookPath("docker"); err != nil {
		return nil, temporal.NewNonRetryableApplicationError("Docker not installed", "container", err)
	}

	state = state.Clone().AddActivityInfo(ctx)

	containerData, err := interpolateContainerConfig(task, state)
	if err != nil {
		return nil, err
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

	entrypoint, hasEntrypoint, err := dockerEntrypointFromInterpolated(containerData["command"])
	if err != nil {
		return nil, err
	}
	if hasEntrypoint {
		cmd = append(cmd, fmt.Sprintf("--entrypoint=%s", entrypoint))
	}

	if task.Run.Container.Lifetime == nil || task.Run.Container.Lifetime.Cleanup == "always" {
		cmd = append(cmd, "--rm")
	}

	cmd = appendDockerEnvFlags(cmd, containerData["environment"])
	cmd, err = appendDockerVolumeFlags(cmd, containerData["volumes"])
	if err != nil {
		return nil, err
	}

	image, err := dockerImageFromInterpolated(containerData["image"])
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, image)

	args, err := dockerArgsFromInterpolated(containerData["arguments"])
	if err != nil {
		return nil, err
	}
	cmd = append(cmd, args...)

	return r.runExecCommand(ctx, []string{cmd[0]}, &model.RunArguments{Value: cmd[1:]}, nil, state, "", task.GetBase())
}

func interpolateContainerConfig(task *model.RunTask, state *utils.State) (map[string]any, error) {
	interpolatedContainer, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(map[string]any{
		"image":       task.Run.Container.Image,
		"command":     task.Run.Container.Command,
		"environment": swUtil.DeepCloneValue(task.Run.Container.Environment),
		"volumes":     swUtil.DeepCloneValue(task.Run.Container.Volumes),
		"arguments":   swUtil.DeepCloneValue(task.Run.Container.Arguments),
	}), nil, state)
	if err != nil {
		return nil, fmt.Errorf("error traversing container parameters: %w", err)
	}
	return interpolatedContainer.(map[string]any), nil
}

func appendDockerEnvFlags(cmd []string, envsRaw any) []string {
	switch envs := envsRaw.(type) {
	case nil:
		return cmd
	case map[string]string:
		for k, v := range envs {
			cmd = append(cmd, fmt.Sprintf("--env=%s=%s", k, v))
		}
	case map[string]any:
		for k, v := range envs {
			cmd = append(cmd, fmt.Sprintf("--env=%s=%s", k, fmt.Sprint(v)))
		}
	}
	return cmd
}

func appendDockerVolumeFlags(cmd []string, volsRaw any) ([]string, error) {
	var vols map[string]string
	switch data := volsRaw.(type) {
	case nil:
		return cmd, nil
	case map[string]string:
		vols = data
	case map[string]any:
		vols = map[string]string{}
		for local, remoteRaw := range data {
			if remoteRaw == nil {
				return nil, fmt.Errorf("invalid container volume for %q: destination cannot be null", local)
			}
			remote, ok := remoteRaw.(string)
			if !ok {
				return nil, fmt.Errorf("invalid container volume for %q: expected string destination, got %T", local, remoteRaw)
			}
			vols[local] = remote
		}
	default:
		return nil, fmt.Errorf("invalid container volumes: expected map, got %T", volsRaw)
	}

	for k, remote := range vols {
		local, err := filepath.Abs(k)
		if err != nil {
			return nil, fmt.Errorf("error getting volume absolute path: %w", err)
		}

		cmd = append(cmd, fmt.Sprintf("--volume=%s:%s", local, remote))
	}
	return cmd, nil
}

func dockerEntrypointFromInterpolated(commandRaw any) (entrypoint string, hasEntrypoint bool, err error) {
	if commandRaw == nil {
		return "", false, nil
	}
	command, ok := commandRaw.(string)
	if !ok {
		return "", false, fmt.Errorf("invalid container command: expected string, got %T", commandRaw)
	}
	if command == "" {
		return "", false, nil
	}
	return command, true, nil
}

func dockerImageFromInterpolated(imageRaw any) (string, error) {
	if imageRaw == nil {
		return "", fmt.Errorf("invalid container image: value is required and cannot be null")
	}
	image, ok := imageRaw.(string)
	if !ok {
		return "", fmt.Errorf("invalid container image: expected string, got %T", imageRaw)
	}
	if strings.TrimSpace(image) == "" {
		return "", fmt.Errorf("invalid container image: value is required and cannot be empty")
	}
	return image, nil
}

func dockerArgsFromInterpolated(argsRaw any) ([]string, error) {
	if argsRaw == nil {
		return nil, nil
	}

	switch args := argsRaw.(type) {
	case []string:
		return args, nil
	case []any:
		containerArgs := make([]string, 0, len(args))
		for i, arg := range args {
			if arg == nil {
				return nil, fmt.Errorf("invalid container argument at index %d: value cannot be null", i)
			}
			containerArgs = append(containerArgs, fmt.Sprint(arg))
		}
		return containerArgs, nil
	default:
		return nil, fmt.Errorf("invalid container arguments: expected array, got %T", argsRaw)
	}
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

func (r *Run) stdToString(std bytes.Buffer) string {
	return strings.TrimSpace(std.String())
}
