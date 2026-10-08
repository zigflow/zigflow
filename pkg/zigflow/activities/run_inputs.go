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
	"fmt"
	"slices"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/zigflow/zigflow/pkg/utils"
)

// The parts of a run task's input the run activities evaluate, keyed by where
// they sit in the tree ResolveRunInputs resolves. Each activity evaluates its
// part under "/<key>", so the locations the workflow defers line up.
const (
	runInputExec        = "exec"        // shell, script: the command's arguments and environment
	runInputSource      = "source"      // script: the external source endpoint
	runInputImage       = "image"       // container on Docker
	runInputArguments   = "arguments"   // container
	runInputEnvironment = "environment" // container

	execArgs = "args"
	execEnv  = "env"
)

// ResolveRunInputs returns a copy of task with the inputs its run activity
// would evaluate resolved in the workflow, leaving expressions that read
// $data.activity for the activity. Only what the run activity evaluates is
// resolved: list-form arguments and the environment of a shell or script, a
// script's external source, and a container's arguments, environment and, on
// Docker only, image.
func ResolveRunInputs(
	task *model.RunTask, runtime ContainerRuntime, state *utils.State,
) (*model.RunTask, *utils.ActivityInputs, error) {
	out, inputs, err := utils.ResolveActivityInputs(runInputs(task, runtime), state, true)
	if err != nil {
		return nil, nil, fmt.Errorf("error traversing task parameters: %w", err)
	}
	tree := out.(map[string]any)

	resolved := *task
	run := &resolved.Run
	switch {
	case run.Shell != nil:
		shell := *run.Shell
		shell.Arguments, shell.Environment = applyExecInputs(shell.Arguments, shell.Environment, tree[runInputExec])
		run.Shell = &shell
	case run.Script != nil:
		script := *run.Script
		script.Arguments, script.Environment = applyExecInputs(script.Arguments, script.Environment, tree[runInputExec])
		if source, ok := tree[runInputSource]; ok && !slices.Contains(inputs.Deferred, "/"+runInputSource) {
			endpoint, ok := source.(string)
			if !ok || endpoint == "" {
				return nil, nil, fmt.Errorf("script source endpoint evaluated to empty or non-string value")
			}
			external := *script.External
			external.Endpoint = model.NewEndpoint(endpoint)
			script.External = &external
		}
		run.Script = &script
	case run.Container != nil:
		container := *run.Container
		if container.Arguments != nil {
			container.Arguments = asStrings(tree[runInputArguments])
		}
		if container.Environment != nil {
			container.Environment, _ = tree[runInputEnvironment].(map[string]string)
		}
		if image, ok := tree[runInputImage]; ok {
			container.Image = fmt.Sprintf("%v", image)
		}
		run.Container = &container
	}

	return &resolved, inputs, nil
}

// runInputs is the tree of what the task's run activity evaluates.
func runInputs(task *model.RunTask, runtime ContainerRuntime) map[string]any {
	run := task.Run
	tree := map[string]any{}
	switch {
	case run.Shell != nil:
		tree[runInputExec] = execInputs(run.Shell.Arguments, run.Shell.Environment)
	case run.Script != nil:
		tree[runInputExec] = execInputs(run.Script.Arguments, run.Script.Environment)
		if ext := run.Script.External; ext != nil && ext.Endpoint != nil {
			tree[runInputSource] = ext.Endpoint.String()
		}
	case run.Container != nil:
		tree[runInputArguments] = run.Container.Arguments
		tree[runInputEnvironment] = run.Container.Environment
		if runtime != ContainerRuntimeKubernetes {
			tree[runInputImage] = run.Container.Image
		}
	}
	return tree
}

// execInputs is the input runExecCommand evaluates: the command's list-form
// arguments (map-form arguments are not passed to the command) and environment.
func execInputs(args *model.RunArguments, env map[string]string) map[string]any {
	if args == nil {
		args = &model.RunArguments{}
	}
	if env == nil {
		env = map[string]string{}
	}
	return map[string]any{
		execArgs: args.AsSlice(),
		execEnv:  env,
	}
}

func applyExecInputs(
	args *model.RunArguments, env map[string]string, resolved any,
) (resolvedArgs *model.RunArguments, resolvedEnv map[string]string) {
	exec := resolved.(map[string]any)
	resolvedArgs, resolvedEnv = args, env
	if args != nil && args.AsSlice() != nil {
		resolvedArgs = &model.RunArguments{Value: asStrings(exec[execArgs])}
	}
	if env != nil {
		resolvedEnv, _ = exec[execEnv].(map[string]string)
	}
	return resolvedArgs, resolvedEnv
}

// asStrings formats evaluated arguments the way the run activities pass them on.
func asStrings(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%v", item))
	}
	return out
}
