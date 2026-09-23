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

package testrunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	zigtemporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/cmd/run"
	"github.com/zigflow/zigflow/pkg/telemetry"
	"github.com/zigflow/zigflow/pkg/zigflow"
	"go.temporal.io/sdk/client"
)

// Status is the outcome of a single zigflow test run.
type Status string

const (
	StatusCompleted Status = "Completed"
	StatusFailed    Status = "Failed"
	StatusTimeout   Status = "Timeout"

	testTimeoutTerminateReason = "zigflow test timeout"
	testTimeoutCleanupDuration = 30 * time.Second
)

// ErrTestTimeout indicates the workflow did not complete before Config.Timeout.
// It is wrapped with the configured duration and context.DeadlineExceeded.
var ErrTestTimeout = errors.New("workflow test timed out")

// Config controls a single workflow test execution.
type Config struct {
	WorkflowFile string
	Input        any
	Timeout      time.Duration
	Temporal     *zigtemporal.TemporalOpts
	TemporalUI   string
	Telemetry    *telemetry.Telemetry
}

// Result is returned when Run completes or fails in a controlled way.
type Result struct {
	WorkflowType   string
	Status         Status
	Duration       time.Duration
	Output         any
	Err            error
	WorkflowID     string
	RunID          string
	TemporalUIURL  string
	TemporalUIBase string
}

// Run validates the workflow file, starts an in-process worker, executes the
// workflow once, waits for completion, and stops the worker.
func Run(ctx context.Context, cfg Config) (*Result, error) {
	if cfg.WorkflowFile == "" {
		return nil, fmt.Errorf("workflow file is required")
	}

	data, err := os.ReadFile(filepath.Clean(cfg.WorkflowFile))
	if err != nil {
		return nil, fmt.Errorf("error loading file: %w", err)
	}

	return RunBytes(ctx, data, cfg)
}

// RunBytes validates a workflow definition from raw YAML or JSON bytes, starts
// an in-process worker, executes the workflow once, waits for completion, and
// stops the worker. WorkflowFile is used only as a diagnostic source label when
// set; RunBytes never reads it.
func RunBytes(ctx context.Context, data []byte, cfg Config) (*Result, error) {
	source := cfg.WorkflowFile
	if source == "" {
		source = "<memory>"
	}
	return runBytes(ctx, data, cfg, source)
}

func runBytes(ctx context.Context, data []byte, cfg Config, source string) (*Result, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Minute
	}
	if cfg.TemporalUI == "" {
		cfg.TemporalUI = "http://localhost:8233"
	}
	if cfg.Temporal == nil {
		cfg.Temporal = &zigtemporal.TemporalOpts{}
	}
	if err := mergeTemporalEnv(cfg.Temporal); err != nil {
		return nil, fmt.Errorf("load Temporal environment: %w", err)
	}

	start := time.Now()

	if err := zigflow.ValidateBytes(data); err != nil {
		return nil, err
	}

	wf, err := zigflow.LoadFromBytes(data)
	if err != nil {
		return nil, err
	}

	workflowType := wf.Document.Name

	runCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	releaseLock, err := acquireTestRunLock(runCtx, cfg.Temporal)
	if err != nil {
		return nil, err
	}
	defer releaseLock()

	session, err := run.StartSession(runCtx, run.SessionConfig{
		Workflow:  wf,
		Source:    source,
		Temporal:  cfg.Temporal,
		Telemetry: cfg.Telemetry,
	})
	if err != nil {
		return nil, err
	}
	defer session.Stop()

	workflowID := NewWorkflowID(workflowType)
	we, err := session.Client().ExecuteWorkflow(runCtx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: session.TaskQueue(),
	}, workflowType, cfg.Input)
	if err != nil {
		return &Result{
			WorkflowType:   workflowType,
			Status:         StatusFailed,
			Duration:       time.Since(start),
			Err:            err,
			WorkflowID:     workflowID,
			TemporalUIBase: cfg.TemporalUI,
		}, nil
	}

	var output any
	getErr := we.Get(runCtx, &output)
	duration := time.Since(start)

	uiURL := HistoryURL(cfg.TemporalUI, cfg.Temporal.Namespace, we.GetID(), we.GetRunID())

	res := &Result{
		WorkflowType:   workflowType,
		Status:         StatusCompleted,
		Duration:       duration,
		Output:         output,
		WorkflowID:     we.GetID(),
		RunID:          we.GetRunID(),
		TemporalUIURL:  uiURL,
		TemporalUIBase: cfg.TemporalUI,
	}

	if getErr != nil {
		res.Status = StatusFailed
		res.Err = getErr
		res.Output = failureOutputForRun(output, getErr)
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			res.Status = StatusTimeout
			res.Err = fmt.Errorf("%w after %s: %w", ErrTestTimeout, cfg.Timeout, context.DeadlineExceeded)
			if termErr := terminateWorkflowAfterTestTimeout(ctx, session.Client(), we.GetID(), we.GetRunID()); termErr != nil {
				res.Err = fmt.Errorf("%w: %v", res.Err, termErr)
			}
		}
	}

	return res, nil
}

// terminateWorkflowAfterTestTimeout stops a timed-out execution before the test
// worker stops polling zigflow-test. TerminateWorkflow is used rather than
// CancelWorkflow so the run ends promptly even when the workflow or activities
// do not honour cancellation.
func terminateWorkflowAfterTestTimeout(parentCtx context.Context, c client.Client, workflowID, runID string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), testTimeoutCleanupDuration)
	defer cancel()

	if err := c.TerminateWorkflow(cleanupCtx, workflowID, runID, testTimeoutTerminateReason); err != nil {
		return fmt.Errorf("terminate workflow after test timeout: %w", err)
	}
	return nil
}

// LoadInputFile reads a JSON object from path for use as workflow input.
func LoadInputFile(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var input any
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parse input JSON: %w", err)
	}
	return input, nil
}

// ParseInputJSON parses inline JSON workflow input.
func ParseInputJSON(raw string) (any, error) {
	var input any
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return nil, fmt.Errorf("parse input JSON: %w", err)
	}
	return input, nil
}

// NewWorkflowID returns a unique workflow id for a test run.
func NewWorkflowID(workflowType string) string {
	shortID, _, _ := strings.Cut(uuid.NewString(), "-")
	return fmt.Sprintf("%s-%s", workflowType, shortID)
}

// HistoryURL builds a Temporal Web UI link to workflow history.
func HistoryURL(uiBase, namespace, workflowID, runID string) string {
	uiBase = strings.TrimRight(uiBase, "/")
	if namespace == "" {
		namespace = "default"
	}
	return fmt.Sprintf("%s/namespaces/%s/workflows/%s/%s/history", uiBase, namespace, workflowID, runID)
}
