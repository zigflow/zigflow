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
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/testsuite"
)

func TestStdToString(t *testing.T) {
	r := &Run{}
	cases := []struct{ input, want string }{
		{"", ""},
		{"hello", "hello"},
		{"\nhello\n", "hello"},
		{"  hello world  ", "hello world"},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		buf.WriteString(c.input)
		assert.Equal(t, c.want, r.stdToString(buf))
	}
}

func TestCallShellActivity(t *testing.T) {
	tests := []struct {
		name    string
		task    *model.RunTask
		state   *utils.State
		want    string
		wantErr bool
	}{
		{
			name: "stdout is returned trimmed",
			task: &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
				Command:   "sh",
				Arguments: &model.RunArguments{Value: []string{"-c", "echo hello"}},
			}}},
			state: utils.NewState(),
			want:  "hello",
		},
		{
			name: "nil args are tolerated",
			task: &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
				Command:   "true",
				Arguments: nil,
			}}},
			state: utils.NewState(),
			want:  "",
		},
		{
			name: "environment variables are passed through",
			task: &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
				Command:     "sh",
				Arguments:   &model.RunArguments{Value: []string{"-c", `printf "%s" "$MY_VAR"`}},
				Environment: map[string]string{"MY_VAR": "my-value"},
			}}},
			state: utils.NewState(),
			want:  "my-value",
		},
		{
			name: "state expressions in environment are interpolated",
			task: &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
				Command:     "sh",
				Arguments:   &model.RunArguments{Value: []string{"-c", `printf "%s" "$OPENAI_API_KEY"`}},
				Environment: map[string]string{"OPENAI_API_KEY": "${ $env.OPENAI_API_KEY }"},
			}}},
			state: func() *utils.State {
				s := utils.NewState()
				s.Env["OPENAI_API_KEY"] = "secret-key"
				return s
			}(),
			want: "secret-key",
		},
		{
			name: "command field is used as the executable with arguments forwarded",
			task: &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
				Command:   "echo",
				Arguments: &model.RunArguments{Value: []string{"hello"}},
			}}},
			state: utils.NewState(),
			want:  "hello",
		},
		{
			name: "non-zero exit code is surfaced as error",
			task: &model.RunTask{Run: model.RunTaskConfiguration{Shell: &model.Shell{
				Command:   "sh",
				Arguments: &model.RunArguments{Value: []string{"-c", "exit 1"}},
			}}},
			state:   utils.NewState(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &Run{}
			var s testsuite.WorkflowTestSuite
			env := s.NewTestActivityEnvironment()
			env.RegisterActivity(run.CallShellActivity)

			val, err := env.ExecuteActivity(run.CallShellActivity, tt.task, nil, tt.state)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			var output string
			require.NoError(t, val.Get(&output))
			assert.Equal(t, tt.want, output)
		})
	}
}

// TestCallContainerActivityEnvExpressionsUseActivityEnrichedState verifies that
// container env expressions are evaluated with activity-enriched state, so that
// $data.activity.* references resolve correctly rather than returning null.
//
// The expression deliberately errors when $data.activity is absent: before the fix
// this produces "Error parsing Docker container envvar"; after the fix the expression
// resolves and docker fails in the normal way ("Error calling command").
func TestCallContainerActivityEnvExpressionsUseActivityEnrichedState(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(run.CallContainerActivity)

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Image: "zigflow-test-does-not-exist:never",
				Environment: map[string]string{
					// Raises a jq error when $data.activity is absent (raw state).
					// Resolves cleanly when state is enriched with activity info.
					"WORKFLOW_ID": `${ $data.activity.workflow_execution_id | if . == null then error("activity data missing") else . end }`,
				},
			},
		},
	}

	_, err := env.ExecuteActivity(run.CallContainerActivity, task, nil, utils.NewState())
	require.Error(t, err)
	// With enriched state the expression resolves; docker then fails normally.
	assert.Contains(t, err.Error(), "Error calling command")
	assert.NotContains(t, err.Error(), "Error parsing Docker container envvar")
}

// TestCallContainerActivityNonStringEnvVarFormattedCorrectly verifies that
// container env expressions returning non-string values (e.g. integers or
// booleans) are formatted with %v, not %s, so the result is "1" not
// "%!s(int=1)".
func TestCallContainerActivityNonStringEnvVarFormattedCorrectly(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(run.CallContainerActivity)

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				Image: "zigflow-test-does-not-exist:never",
				Environment: map[string]string{
					"INT_VAR":  "${ 1 }",
					"BOOL_VAR": "${ true }",
				},
			},
		},
	}

	_, err := env.ExecuteActivity(run.CallContainerActivity, task, nil, utils.NewState())
	require.Error(t, err)
	// Docker fails (image not found) — but the env flags must not carry %!s(...)
	// format artefacts from incorrect %s interpolation of non-string values.
	assert.NotContains(t, err.Error(), "%!s(")
}

// TestCallContainerActivityNilEnvVarCoercedToEmpty verifies that a container
// environment variable whose expression evaluates to null is coerced to an
// empty string, producing "--env=KEY=" rather than "--env=KEY=%!s(<nil>)".
func TestCallContainerActivityNilEnvVarCoercedToEmpty(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(run.CallContainerActivity)

	task := &model.RunTask{
		Run: model.RunTaskConfiguration{
			Container: &model.Container{
				// Deliberately nonexistent image — docker fails fast without running anything.
				Image: "zigflow-test-does-not-exist:never",
				Environment: map[string]string{
					"MY_VAR": "${ null }", // jq null evaluates to nil in Go
				},
			},
		},
	}

	_, err := env.ExecuteActivity(run.CallContainerActivity, task, nil, utils.NewState())
	// Docker will fail (image not found), but the error must not contain the
	// nil format artefact that would appear if the nil guard were absent.
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "%!s(<nil>)")
}

// TestRunExecCommandRespectsWorkingDirectory proves that the dir parameter is
// honoured by the exec. runExecCommand is unexported, so it is wrapped in a
// local closure to satisfy the test activity environment.
func TestRunExecCommandRespectsWorkingDirectory(t *testing.T) {
	dir := t.TempDir()

	run := &Run{}
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()

	testActivity := func(ctx context.Context) (any, error) {
		return run.runExecCommand(
			ctx,
			[]string{"sh"}, &model.RunArguments{Value: []string{"-c", "pwd"}},
			nil, utils.NewState(), dir, &model.TaskBase{},
		)
	}
	env.RegisterActivity(testActivity)

	val, err := env.ExecuteActivity(testActivity)
	require.NoError(t, err)
	var output string
	require.NoError(t, val.Get(&output))
	assert.Equal(t, dir, output)
}

// TestCallScriptActivityExternalSource covers the external-script fetch path:
// file sources, HTTP sources, unsupported schemes, and runtime-expression endpoints.
func TestCallScriptActivityExternalSource(t *testing.T) {
	tests := []struct {
		name      string
		binary    string
		want      string
		wantErr   bool
		errSubstr string
		makeTask  func(t *testing.T) (*model.RunTask, *utils.State)
	}{
		{
			name:   "external file source is fetched and executed",
			binary: "node",
			want:   "hello from file",
			makeTask: func(t *testing.T) (*model.RunTask, *utils.State) {
				f, err := os.CreateTemp("", "*.js")
				require.NoError(t, err)
				t.Cleanup(func() { _ = os.Remove(f.Name()) })
				_, err = fmt.Fprint(f, `process.stdout.write("hello from file")`)
				require.NoError(t, err)
				require.NoError(t, f.Close())
				return &model.RunTask{
					Run: model.RunTaskConfiguration{
						Script: &model.Script{
							Language: "js",
							External: &model.ExternalResource{
								Endpoint: model.NewEndpoint("file://" + f.Name()),
							},
						},
					},
				}, utils.NewState()
			},
		},
		{
			name:   "external HTTP source is fetched and executed",
			binary: "node",
			want:   "hello from http",
			makeTask: func(t *testing.T) (*model.RunTask, *utils.State) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = fmt.Fprint(w, `process.stdout.write("hello from http")`)
				}))
				t.Cleanup(srv.Close)
				return &model.RunTask{
					Run: model.RunTaskConfiguration{
						Script: &model.Script{
							Language: "js",
							External: &model.ExternalResource{
								Endpoint: model.NewEndpoint(srv.URL + "/script.js"),
							},
						},
					},
				}, utils.NewState()
			},
		},
		{
			name:      "unsupported scheme returns clear error",
			wantErr:   true,
			errSubstr: "error reading file",
			makeTask: func(t *testing.T) (*model.RunTask, *utils.State) {
				return &model.RunTask{
					Run: model.RunTaskConfiguration{
						Script: &model.Script{
							Language: "js",
							External: &model.ExternalResource{
								Endpoint: &model.Endpoint{
									URITemplate: &model.LiteralUri{Value: "ftp://example.com/script.js"},
								},
							},
						},
					},
				}, utils.NewState()
			},
		},
		{
			// The endpoint is a runtime expression that resolves to the test server URL
			// via $env.SCRIPT_URL. Verifies EvaluateString is called before ReadURLContents.
			name:   "runtime-expression endpoint is evaluated before fetching",
			binary: "node",
			want:   "hello from expression",
			makeTask: func(t *testing.T) (*model.RunTask, *utils.State) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = fmt.Fprint(w, `process.stdout.write("hello from expression")`)
				}))
				t.Cleanup(srv.Close)
				state := utils.NewState()
				state.Env["SCRIPT_URL"] = srv.URL + "/script.js"
				return &model.RunTask{
					Run: model.RunTaskConfiguration{
						Script: &model.Script{
							Language: "js",
							External: &model.ExternalResource{
								Endpoint: &model.Endpoint{
									RuntimeExpression: model.NewExpr("${ $env.SCRIPT_URL }"),
								},
							},
						},
					},
				}, state
			},
		},
		{
			name:      "nil endpoint returns clean error without panic",
			wantErr:   true,
			errSubstr: "external script source has no endpoint",
			makeTask: func(t *testing.T) (*model.RunTask, *utils.State) {
				return &model.RunTask{
					Run: model.RunTaskConfiguration{
						Script: &model.Script{
							Language: "js",
							External: &model.ExternalResource{
								// Endpoint intentionally nil
							},
						},
					},
				}, utils.NewState()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.binary != "" {
				if _, err := exec.LookPath(tt.binary); err != nil {
					t.Skipf("%s not available", tt.binary)
				}
			}

			task, state := tt.makeTask(t)
			run := &Run{}
			var s testsuite.WorkflowTestSuite
			env := s.NewTestActivityEnvironment()
			env.RegisterActivity(run.CallScriptActivity)

			val, err := env.ExecuteActivity(run.CallScriptActivity, task, nil, state)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}
			require.NoError(t, err)
			var output string
			require.NoError(t, val.Get(&output))
			assert.Equal(t, tt.want, output)
		})
	}
}

func TestCallScriptActivity(t *testing.T) {
	tests := []struct {
		name      string
		lang      string
		code      string
		args      *model.RunArguments
		binary    string
		want      string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "unknown language returns error",
			lang:      "ruby",
			code:      `puts "hello"`,
			wantErr:   true,
			errSubstr: "unknown script language",
		},
		{
			name:   "js script is executed and stdout is captured",
			lang:   "js",
			code:   `process.stdout.write("hello from js")`,
			binary: "node",
			want:   "hello from js",
		},
		{
			name:   "python script is executed and stdout is captured",
			lang:   "python",
			code:   `import sys; sys.stdout.write("hello from python")`,
			binary: "python",
			want:   "hello from python",
		},
		{
			name:   "script arguments are forwarded after the script file",
			lang:   "js",
			code:   `process.stdout.write(process.argv.slice(2).join(","))`,
			args:   &model.RunArguments{Value: []string{"x", "y"}},
			binary: "node",
			want:   "x,y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.binary != "" {
				if _, err := exec.LookPath(tt.binary); err != nil {
					t.Skipf("%s not available", tt.binary)
				}
			}

			run := &Run{}
			var s testsuite.WorkflowTestSuite
			env := s.NewTestActivityEnvironment()
			env.RegisterActivity(run.CallScriptActivity)

			code := tt.code
			task := &model.RunTask{
				Run: model.RunTaskConfiguration{
					Script: &model.Script{
						Language:   tt.lang,
						InlineCode: &code,
						Arguments:  tt.args,
					},
				},
			}

			val, err := env.ExecuteActivity(run.CallScriptActivity, task, nil, utils.NewState())
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}
			require.NoError(t, err)
			var output string
			require.NoError(t, val.Get(&output))
			assert.Equal(t, tt.want, output)
		})
	}
}
