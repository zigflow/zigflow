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

	script := task.Run.Script

	lang := script.Language
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

	contents, err := r.resolveScriptContents(ctx, script, state)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(fname, contents, 0o600); err != nil {
		logger.Error("Error writing script to disk", "file", fname, "error", err)
		return nil, fmt.Errorf("error writing code to script: %w", err)
	}

	return r.runExecCommand(
		ctx,
		command,
		script.Arguments,
		script.Environment,
		state,
		dir,
		task.GetBase(),
	)
}

func (r *Run) resolveScriptContents(ctx context.Context, script *model.Script, state *utils.State) ([]byte, error) {
	if ext := script.External; ext != nil {
		if ext.Endpoint == nil {
			return nil, fmt.Errorf("external script source has no endpoint")
		}
		logger := activity.GetLogger(ctx)
		enrichedState := state.Clone().AddActivityInfo(ctx)
		rawEndpoint := ext.Endpoint.String()
		logger.Debug("Evaluating script source endpoint", "raw", rawEndpoint)

		evaluated, err := utils.EvaluateString(rawEndpoint, nil, enrichedState)
		if err != nil {
			return nil, fmt.Errorf("error evaluating script source endpoint: %w", err)
		}
		endpoint, ok := evaluated.(string)
		if !ok || endpoint == "" {
			return nil, fmt.Errorf("script source endpoint evaluated to empty or non-string value")
		}

		logger.Debug("Reading file contents from endpoint", "endpoint", endpoint)
		c, err := utils.ReadURLContents(ctx, endpoint)
		if err != nil {
			logger.Error("Error reading file from endpoint", "endpoint", endpoint, "error", err)
			return nil, fmt.Errorf("error reading file: %w", err)
		}
		return c, nil
	}
	return []byte(*script.InlineCode), nil
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
		enrichedState := state.Clone().AddActivityInfo(ctx)

		for k, v := range envs {
			parsedV, err := utils.EvaluateString(v, nil, enrichedState)
			if err != nil {
				return nil, temporal.NewNonRetryableApplicationError("Error parsing Docker container envvar", "container", err)
			}
			var value string
			if parsedV == nil {
				value = ""
			} else {
				value = fmt.Sprintf("%v", parsedV)
			}
			cmd = append(cmd, fmt.Sprintf("--env=%s=%s", k, value))
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

func (r *Run) stdToString(std bytes.Buffer) string {
	return strings.TrimSpace(std.String())
}
