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

package run

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/worker"
)

// ---- isWatchableEvent ----

func TestIsWatchableEvent(t *testing.T) {
	tests := []struct {
		name     string
		op       fsnotify.Op
		expected bool
	}{
		{"Write triggers reload", fsnotify.Write, true},
		{"Create triggers reload", fsnotify.Create, true},
		{"Rename triggers reload", fsnotify.Rename, true},
		{"Remove does not trigger reload", fsnotify.Remove, false},
		{"Chmod does not trigger reload", fsnotify.Chmod, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := fsnotify.Event{Name: "workflow.yaml", Op: tc.op}
			assert.Equal(t, tc.expected, isWatchableEvent(event))
		})
	}
}

// ---- resetDebounce ----

func TestResetDebounce_DoesNotDeadlock(t *testing.T) {
	// Create an already-fired timer and verify resetDebounce handles it safely.
	timer := time.NewTimer(1 * time.Nanosecond)
	time.Sleep(5 * time.Millisecond) // let it fire
	resetDebounce(timer, 10*time.Millisecond)
	// Stop immediately to avoid leaving a live goroutine timer; drain if needed.
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func TestResetDebounce_OnIdleTimer(t *testing.T) {
	// resetDebounce on a timer that has not fired must not panic or deadlock.
	timer := newStoppedTimer(10 * time.Millisecond)
	resetDebounce(timer, 20*time.Millisecond)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

// ---- changedFiles deduplication ----

func TestChangedFilesDeduplication(t *testing.T) {
	// Simulate the map-based set used in runWatchMode to accumulate changed files.
	changedFiles := make(map[string]struct{})

	events := []fsnotify.Event{
		{Name: "/a/workflow.yaml", Op: fsnotify.Write},
		{Name: "/a/workflow.yaml", Op: fsnotify.Write},
		{Name: "/a/workflow.yaml", Op: fsnotify.Rename},
		{Name: "/b/other.yaml", Op: fsnotify.Create},
	}

	for _, e := range events {
		if isWatchableEvent(e) {
			changedFiles[e.Name] = struct{}{}
		}
	}

	assert.Len(t, changedFiles, 2, "duplicate paths must be deduplicated")
	assert.Contains(t, changedFiles, "/a/workflow.yaml")
	assert.Contains(t, changedFiles, "/b/other.yaml")
}

func TestChangedFilesNonWatchableEventsIgnored(t *testing.T) {
	changedFiles := make(map[string]struct{})

	events := []fsnotify.Event{
		{Name: "/a/workflow.yaml", Op: fsnotify.Remove},
		{Name: "/a/workflow.yaml", Op: fsnotify.Chmod},
	}

	for _, e := range events {
		if isWatchableEvent(e) {
			changedFiles[e.Name] = struct{}{}
		}
	}

	assert.Empty(t, changedFiles, "Remove and Chmod events must not enter changedFiles")
}

// ---- refreshWatcher ----

func TestRefreshWatcher_AddsDiscoveredFiles(t *testing.T) {
	dir := t.TempDir()
	p := writeTempWorkflow(t, dir, "ns", "wf")

	w, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	opts := &runOptions{
		Files:         []string{p},
		DirectoryGlob: "*.yaml",
	}

	require.NoError(t, refreshWatcher(w, opts))

	watched := w.WatchList()
	require.Len(t, watched, 1)
	assert.Equal(t, p, watched[0])
}

func TestRefreshWatcher_RemovesStaleAndReAdds(t *testing.T) {
	dir := t.TempDir()
	p1 := writeTempWorkflow(t, dir, "ns", "wf1")
	p2 := writeTempWorkflow(t, dir, "ns", "wf2")

	w, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	// Seed the watcher with p1 only.
	require.NoError(t, w.Add(p1))
	assert.Len(t, w.WatchList(), 1)

	// refreshWatcher should remove p1 and add both p1 and p2.
	opts := &runOptions{
		Files:         []string{p1, p2},
		DirectoryGlob: "*.yaml",
	}
	require.NoError(t, refreshWatcher(w, opts))

	watched := w.WatchList()
	assert.Len(t, watched, 2)
	assert.Contains(t, watched, p1)
	assert.Contains(t, watched, p2)
}

func TestRefreshWatcher_ReturnsErrorWhenNoFiles(t *testing.T) {
	w, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	// No files and no directory: discoverWorkflowFiles must fail.
	opts := &runOptions{DirectoryGlob: "*.yaml"}
	err = refreshWatcher(w, opts)
	assert.Error(t, err)
}

// TestRefreshWatcher_PreservesWatchesOnAddFailure verifies that a failed
// watcher.Add during refresh does not leave the watch list in a partially
// disabled state. The previously-watched file must still be watched after the
// refresh returns an error.
//
// The test orders the failing path first in opts.Files so that, with the old
// remove-then-add logic, the watcher would end up with an empty watch list.
// The two-phase implementation must keep the original watch intact.
func TestRefreshWatcher_PreservesWatchesOnAddFailure(t *testing.T) {
	dir := t.TempDir()
	p := writeTempWorkflow(t, dir, "ns", "wf")

	w, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	// Seed the watcher with one real file.
	require.NoError(t, w.Add(p))
	require.Len(t, w.WatchList(), 1)

	// opts references a non-existent path first, then the real one.
	// discoverWorkflowFiles resolves paths via filepath.Abs (no existence check),
	// so both appear in the target list. watcher.Add fails on the missing path.
	// With the old code the real watch would be stripped before the failure;
	// with the two-phase code the real watch must survive.
	nonExistent := filepath.Join(dir, "does-not-exist.yaml")
	opts := &runOptions{
		Files:         []string{nonExistent, p},
		DirectoryGlob: "*.yaml",
	}

	err = refreshWatcher(w, opts)
	assert.Error(t, err, "refreshWatcher must return an error when a watch add fails")

	watched := w.WatchList()
	assert.Contains(t, watched, p, "previously-watched file must still be watched after a failed refresh")
}

// TestRefreshWatcher_RemovesStaleWatch verifies that paths no longer in the
// target set are removed once all new adds succeed.
func TestRefreshWatcher_RemovesStaleWatch(t *testing.T) {
	dir := t.TempDir()
	p1 := writeTempWorkflow(t, dir, "ns", "wf1")
	p2 := writeTempWorkflow(t, dir, "ns", "wf2")

	w, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	require.NoError(t, w.Add(p1))
	require.NoError(t, w.Add(p2))
	require.Len(t, w.WatchList(), 2)

	// Target only p1; p2 should be removed.
	opts := &runOptions{
		Files:         []string{p1},
		DirectoryGlob: "*.yaml",
	}
	require.NoError(t, refreshWatcher(w, opts))

	watched := w.WatchList()
	assert.Len(t, watched, 1)
	assert.Contains(t, watched, p1)
	assert.NotContains(t, watched, p2)
}

// ---- handleDebounce swap logic ----

func TestHandleDebounce_EmptyChangedFilesIsNoop(t *testing.T) {
	dir := t.TempDir()
	p := writeTempWorkflow(t, dir, "ns", "wf")

	w, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() { _ = w.Close() }()
	require.NoError(t, w.Add(p))

	// Use nil workers to prove no stop calls are attempted when changedFiles
	// is empty (stopWorkerList would panic on a non-nil entry).
	current := []worker.Worker(nil)
	changedFiles := make(map[string]struct{})

	next, remaining := handleDebounce(w, nil, &runOptions{Files: []string{p}, DirectoryGlob: "*.yaml"}, nil, changedFiles, current)

	assert.Equal(t, current, next, "worker slice must be unchanged when changedFiles is empty")
	assert.Empty(t, remaining)
}

func TestHandleDebounce_KeepsCurrentWorkersOnReloadFailure(t *testing.T) {
	// When launchWorkers fails, the returned worker slice must equal the
	// original and changedFiles must be cleared.
	// We force a failure at the discoverWorkflowFiles stage by providing opts
	// with no files and no directory, which returns "No workflow files found"
	// before any Temporal client interaction occurs.
	w, err := fsnotify.NewWatcher()
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	current := []worker.Worker(nil)
	// Use any non-empty path as the changed-file key; the actual value is only
	// used for logging in handleDebounce.
	changedFiles := map[string]struct{}{"/some/workflow.yaml": {}}

	// opts with no files: prepareRegistrations will fail before reaching
	// buildWorkersByTaskQueue, so no Temporal client is needed.
	opts := &runOptions{DirectoryGlob: "*.yaml"}
	next, remaining := handleDebounce(w, nil, opts, nil, changedFiles, current)

	assert.Empty(t, remaining, "changedFiles must be cleared regardless of reload outcome")
	assert.Equal(t, current, next, "current workers must be unchanged when reload fails")
}
