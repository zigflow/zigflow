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

package schema

// taskBaseSchema returns the common fields shared by all task types. These
// fields are folded into each task schema via AllOf so they appear in editor
// autocompletion alongside the task-specific fields.
func taskBaseSchema() *Schema {
	return Object(map[string]*Schema{
		"if":     WithDescription(String(), "Runtime expression evaluated before the task runs. Task is skipped when false."),
		"input":  WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Transforms the task input before execution."),
		"output": WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Transforms the task output after execution."),
		"export": WithDescription(
			&Schema{Type: "object", AdditionalProperties: true},
			"Exports data to the workflow context after the task completes.",
		),
		// timeout maps to model.TimeoutOrReference: either an inline {after: duration}
		// object or a string reference to a named timeout in the use block.
		// Note: Zigflow does not currently act on task-level timeout at runtime;
		// activity timeouts are controlled via metadata.activityOptions instead.
		"timeout":  WithDescription(taskTimeoutSchema(), "Task-level timeout configuration."),
		"metadata": Ref("#/$defs/metadata"),
		"then":     WithDescription(String(), "Name of the next task to execute, or a flow directive (end, exit, continue)."),
	})
}

// taskTimeoutSchema returns the schema for the task-level timeout field. The
// model accepts either an inline timeout object (with a single required after
// field) or a string reference to a named timeout defined in the use block.
// This matches model.TimeoutOrReference.UnmarshalJSON which tries inline first
// then falls back to a string. The inline branch carries additionalProperties:
// false because model.Timeout only has one field (After).
func taskTimeoutSchema() *Schema {
	inlineTimeout := Object(
		map[string]*Schema{
			"after": WithDescription(Ref("#/$defs/duration"), "Duration after which the task times out."),
		},
		"after",
	)
	inlineTimeout.AdditionalProperties = false

	return AnyOf(
		WithDescription(inlineTimeout, "Inline timeout with an after duration."),
		WithDescription(String(), "Reference to a named timeout defined in the use block."),
	)
}

// callActivityTaskSchema returns the schema for a Zigflow custom Temporal activity call.
// The call field must be the literal string "activity". The with object maps to
// models.ActivityCallWith.
func callActivityTaskSchema() *Schema {
	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"call": WithDescription(StringConst("activity"), "Identifies this as a custom Temporal activity call."),
				"with": WithDescription(
					Object(
						map[string]*Schema{
							"name":      WithDescription(String(), "Registered name of the Temporal activity to invoke."),
							"arguments": WithDescription(Array(Any()), "Positional arguments passed to the activity."),
							"taskQueue": WithDescription(String(), "Temporal task queue that handles this activity."),
						},
						"name", "taskQueue",
					),
					"Activity call configuration.",
				),
			},
			"call", "with",
		),
	)
}

// callHTTPTaskSchema returns the schema for an HTTP activity call.
func callHTTPTaskSchema() *Schema {
	endpoint := AnyOf(
		WithDescription(String(), "Literal URL string."),
		Object(
			map[string]*Schema{
				"uri":            WithDescription(String(), "The request URI."),
				"authentication": WithDescription(String(), "Named authentication policy to apply."),
			},
			"uri",
		),
	)

	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"call": WithDescription(StringConst("http"), "Identifies this as an HTTP call."),
				"with": WithDescription(
					Object(
						map[string]*Schema{
							"method":   WithDescription(StringEnum("GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"), "HTTP method."),
							"endpoint": WithDescription(endpoint, "Target URL or endpoint object."),
							"headers":  WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Request headers."),
							"query":    WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Query string parameters."),
							"body":     WithDescription(Any(), "Request body (any serialisable value)."),
						},
						"method", "endpoint",
					),
					"HTTP call configuration.",
				),
			},
			"call", "with",
		),
	)
}

// callGRPCTaskSchema returns the schema for a gRPC activity call.
func callGRPCTaskSchema() *Schema {
	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"call": WithDescription(StringConst("grpc"), "Identifies this as a gRPC call."),
				"with": WithDescription(
					Object(
						map[string]*Schema{
							"proto": WithDescription(
								AnyOf(String(), &Schema{Type: "object", AdditionalProperties: true}),
								"Protobuf service descriptor or inline proto definition.",
							),
							"service": WithDescription(
								Object(
									map[string]*Schema{
										"name":           WithDescription(String(), "Fully qualified gRPC service name."),
										"host":           WithDescription(String(), "Service host (default: localhost)."),
										"port":           WithDescription(Integer(), "Service port (default: 50051)."),
										"authentication": WithDescription(String(), "Named authentication policy."),
									},
									"name",
								),
								"gRPC service location.",
							),
							"method":    WithDescription(String(), "gRPC method name."),
							"arguments": WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Method arguments."),
						},
						"service", "method",
					),
					"gRPC call configuration.",
				),
			},
			"call", "with",
		),
	)
}

// doTaskSchema returns the schema for a sequential task group.
func doTaskSchema() *Schema {
	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"do": WithDescription(Ref("#/$defs/taskList"), "Sequential list of tasks to execute."),
			},
			"do",
		),
	)
}

// forTaskSchema returns the schema for a loop task.
func forTaskSchema() *Schema {
	forDef := Object(
		map[string]*Schema{
			"each": WithDescription(String(), "Variable name bound to the current iteration value."),
			"in":   WithDescription(String(), "Runtime expression that evaluates to the collection to iterate over."),
			"at":   WithDescription(String(), "Variable name bound to the current iteration index."),
		},
		"each", "in",
	)

	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"for": WithDescription(forDef, "Loop definition."),
				"do":  WithDescription(Ref("#/$defs/taskList"), "Tasks executed on each iteration."),
			},
			"for", "do",
		),
	)
}

// forkTaskSchema returns the schema for a parallel task group.
func forkTaskSchema() *Schema {
	// ForkTaskConfiguration.Branches is *TaskList (a single flat TaskList, not
	// an array of TaskLists). All tasks in the list run concurrently.
	forkDef := Object(
		map[string]*Schema{
			"branches": WithDescription(Ref("#/$defs/taskList"), "Tasks to execute in parallel."),
			"compete":  WithDescription(Boolean(), "When true, the first task to complete cancels the remaining tasks."),
		},
		"branches",
	)

	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"fork": WithDescription(forkDef, "Parallel execution configuration."),
			},
			"fork",
		),
	)
}

// listenTaskSchema returns the schema for a task that waits for a Temporal
// query, signal, or update. Supports one, any, and all consumption strategies.
func listenTaskSchema() *Schema {
	// eventFilterSchema describes a single event listener. The runtime requires
	// id (maps to the Temporal signal/query/update name) and type (one of the
	// three supported Temporal mechanism types). acceptIf and data come from
	// EventProperties.Additional and are documented as optional known extensions.
	eventFilter := Object(
		map[string]*Schema{
			"with": WithDescription(
				&Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"id": WithDescription(String(), "Temporal signal, query, or update name. Required by the runtime."),
						"type": WithDescription(
							StringEnum("query", "signal", "update"),
							"Temporal mechanism type.",
						),
						"source":   WithDescription(String(), "CloudEvent source filter."),
						"acceptIf": WithDescription(Any(), "Runtime expression evaluated to decide whether the event is accepted."),
						"data":     WithDescription(Any(), "Runtime expression or value used as the reply payload."),
					},
					Required:             []string{"id", "type"},
					AdditionalProperties: true,
				},
				"Event filter criteria.",
			),
		},
		"with",
	)

	// toDef enforces that exactly one of one, any, or all is present. oneOf
	// rejects an empty object (no branch matches) and rejects multiple keys
	// (more than one branch matches). additionalProperties: false prevents
	// unknown strategy keys from being silently accepted.
	toDef := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"one": WithDescription(eventFilter, "Wait for a single matching event."),
			"any": WithDescription(Array(eventFilter), "Complete when any one of the listed events arrives."),
			"all": WithDescription(Array(eventFilter), "Complete when all listed events have arrived."),
		},
		AdditionalProperties: false,
		OneOf: []*Schema{
			{Required: []string{"one"}},
			{Required: []string{"any"}},
			{Required: []string{"all"}},
		},
	}

	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"listen": WithDescription(
					Object(
						map[string]*Schema{
							"to": WithDescription(toDef, "Event consumption strategy. Exactly one of one, any, or all must be specified."),
						},
						"to",
					),
					"Listen configuration.",
				),
			},
			"listen",
		),
	)
}

// raiseTaskSchema returns the schema for a task that raises an error.
func raiseTaskSchema() *Schema {
	errorDef := Object(
		map[string]*Schema{
			"type":     WithDescription(String(), "Error type (e.g. https://zigflow.dev/errors/validation)."),
			"status":   WithDescription(Integer(), "HTTP-style status code for the error."),
			"title":    WithDescription(String(), "Short human-readable error title."),
			"detail":   WithDescription(String(), "Detailed human-readable error description."),
			"instance": WithDescription(String(), "URI identifying the specific error occurrence."),
		},
		"type", "status",
	)

	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"raise": WithDescription(
					Object(
						map[string]*Schema{
							"error": WithDescription(errorDef, "The error to raise."),
						},
						"error",
					),
					"Raise configuration.",
				),
			},
			"raise",
		),
	)
}

// runArguments is the schema for script and shell arguments. model.RunArguments
// accepts either a list of strings or a string-keyed map.
func runArguments() *Schema {
	return WithDescription(
		AnyOf(
			WithDescription(Array(String()), "Positional argument list."),
			WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Named argument map."),
		),
		"Arguments passed to the script or shell command.",
	)
}

// runTaskSchema returns the schema for a task that executes a container, script,
// shell command, or child workflow. Exactly one of container, script, shell, or
// workflow must be specified. This is enforced both by the runtime
// (RunTaskConfiguration.UnmarshalJSON counts non-nil fields and rejects count != 1)
// and by the oneOf constraint on runDef below.
func runTaskSchema() *Schema {
	// containerBase rejects the ports field: the runtime returns an error
	// ("ports are not allowed on containers") when any port mappings are present.
	// not:{required:[ports]} catches this at schema validation time.
	// model.Container.Arguments is []string (not RunArguments), so arguments
	// here is Array(String()) only — not the anyOf map form.
	// model.Container also has a name field (optional label).
	containerBase := Object(
		map[string]*Schema{
			"image":       WithDescription(String(), "Container image reference."),
			"name":        WithDescription(String(), "Optional container name label."),
			"command":     WithDescription(String(), "Command to run inside the container."),
			"arguments":   WithDescription(Array(String()), "Positional arguments passed to the container command."),
			"environment": WithDescription(&Schema{Type: "object", AdditionalProperties: &Schema{Type: "string"}}, "Environment variables."),
			"volumes":     WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Volume mounts."),
		},
		"image",
	)
	containerBase.Extra = map[string]any{
		"not": map[string]any{"required": []string{"ports"}},
	}
	containerDef := WithDescription(containerBase, "Container execution configuration.")

	// scriptDef requires code (runtime rejects missing code: "run script has no
	// code defined"). Language enum includes "javascript" in addition to "js" and
	// "python" because model.Script validates oneof=javascript js python.
	scriptDef := WithDescription(
		Object(
			map[string]*Schema{
				"language":    WithDescription(StringEnum("js", "javascript", "python"), "Script language."),
				"code":        WithDescription(String(), "Inline script source code."),
				"arguments":   runArguments(),
				"environment": WithDescription(&Schema{Type: "object", AdditionalProperties: &Schema{Type: "string"}}, "Environment variables."),
			},
			"language", "code",
		),
		"Inline script execution configuration.",
	)

	shellDef := WithDescription(
		Object(
			map[string]*Schema{
				"command":     WithDescription(String(), "Shell command to run."),
				"arguments":   runArguments(),
				"environment": WithDescription(&Schema{Type: "object", AdditionalProperties: &Schema{Type: "string"}}, "Environment variables."),
			},
			"command",
		),
		"Shell command execution configuration.",
	)

	workflowDef := WithDescription(
		Object(
			map[string]*Schema{
				"name":      WithDescription(String(), "Name of the child workflow to execute."),
				"namespace": WithDescription(String(), "Temporal namespace of the child workflow."),
				"version":   WithDescription(String(), "Semantic version of the child workflow."),
				"input":     WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Input passed to the child workflow."),
			},
			"name", "namespace", "version",
		),
		"Child workflow execution configuration.",
	)

	// runBase declares all valid properties. runOneOf enforces that exactly one
	// of container/script/shell/workflow is present. An object with two run-mode
	// keys satisfies two oneOf branches, which fails the oneOf constraint.
	// The script branch carries the await != false constraint because the
	// runtime rejects await:false when a script is present.
	runBase := Object(
		map[string]*Schema{
			"await":     WithDescription(Boolean(), "Wait for the run to finish before continuing (default: true)."),
			"container": containerDef,
			"script":    scriptDef,
			"shell":     shellDef,
			"workflow":  workflowDef,
		},
	)

	scriptBranch := &Schema{
		Type:     "object",
		Required: []string{"script"},
		Properties: map[string]*Schema{
			// await: false is not permitted for scripts. The runtime rejects it
			// ("run scripts must be run with await").
			"await": {Extra: map[string]any{"not": map[string]any{"const": false}}},
		},
	}
	runOneOf := OneOf(
		WithDescription(&Schema{Type: "object", Required: []string{"container"}}, "Container run mode."),
		WithDescription(scriptBranch, "Script run mode. await: false is not permitted."),
		WithDescription(&Schema{Type: "object", Required: []string{"shell"}}, "Shell run mode."),
		WithDescription(&Schema{Type: "object", Required: []string{"workflow"}}, "Workflow run mode."),
	)

	runDef := WithDescription(
		AllOf(runBase, runOneOf),
		"Run configuration. Exactly one of container, script, shell, or workflow must be set.",
	)

	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"run": runDef,
			},
			"run",
		),
	)
}

// setTaskSchema returns the schema for a task that assigns values to workflow variables.
func setTaskSchema() *Schema {
	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"set": WithDescription(
					&Schema{Type: "object", AdditionalProperties: true},
					"Key-value pairs to assign into the workflow context.",
				),
			},
			"set",
		),
	)
}

// switchTaskSchema returns the schema for a conditional branching task.
func switchTaskSchema() *Schema {
	caseItem := Object(
		map[string]*Schema{
			"when": WithDescription(String(), "Runtime expression that must evaluate to true for this case to match. Omit for the default case."),
			"then": WithDescription(String(), "Name of the task to execute when this case matches, or a flow directive."),
		},
		"then",
	)

	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"switch": WithDescription(
					Array(caseItem),
					"Ordered list of cases evaluated top to bottom. The first matching case is taken.",
				),
			},
			"switch",
		),
	)
}

// tryTaskSchema returns the schema for a try-catch error handling task.
func tryTaskSchema() *Schema {
	catchDef := Object(
		map[string]*Schema{
			"as":    WithDescription(String(), "Variable name that holds the caught error object."),
			"when":  WithDescription(String(), "Runtime expression to filter which errors are caught."),
			"retry": WithDescription(&Schema{Type: "object", AdditionalProperties: true}, "Inline retry policy for the caught error."),
			"do":    WithDescription(Ref("#/$defs/taskList"), "Tasks to execute when an error is caught."),
		},
	)

	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"try":   WithDescription(Ref("#/$defs/taskList"), "Tasks to attempt."),
				"catch": WithDescription(catchDef, "Error handling configuration."),
			},
			"try", "catch",
		),
	)
}

// waitTaskSchema returns the schema for a sleep/delay task.
func waitTaskSchema() *Schema {
	return AllOf(
		Ref("#/$defs/taskBase"),
		Object(
			map[string]*Schema{
				"wait": WithDescription(Ref("#/$defs/duration"), "Duration to wait before continuing."),
			},
			"wait",
		),
	)
}
