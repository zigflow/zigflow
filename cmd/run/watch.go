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
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// newStoppedTimer creates a timer and immediately stops it, draining any
// pending tick. Use timer.Reset(d) to arm it for the first time.
func newStoppedTimer(d time.Duration) *time.Timer {
	t := time.NewTimer(d)
	if !t.Stop() {
		<-t.C
	}
	return t
}

// resetDebounce safely stops t and resets it to d, draining any pending tick
// so the timer fires exactly once after d has elapsed.
func resetDebounce(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

// isWatchableEvent reports whether the fsnotify event should trigger a reload.
// Write covers in-place saves; Create and Rename cover atomic editor patterns
// such as vim's rename-over-original approach.
func isWatchableEvent(e fsnotify.Event) bool {
	return e.Has(fsnotify.Write) || e.Has(fsnotify.Create) || e.Has(fsnotify.Rename)
}

// handleDebounce executes a reload cycle after the debounce timer fires.
// It logs the changed files, attempts to rebuild workers, swaps them on success,
// and always refreshes the watcher to recover inodes lost by rename-style saves.
// It returns the updated current workers and a fresh (empty) changedFiles map.
func handleDebounce(
	watcher *fsnotify.Watcher,
	temporalClient client.Client,
	opts *runOptions,
	envvars map[string]any,
	changedFiles map[string]struct{},
	current []worker.Worker,
) (nextWorkers []worker.Worker, remainingChanges map[string]struct{}) {
	if len(changedFiles) == 0 {
		return current, changedFiles
	}

	names := make([]string, 0, len(changedFiles))
	for f := range changedFiles {
		names = append(names, filepath.Base(f))
	}
	sort.Strings(names)
	log.Warn().Str("files", strings.Join(names, ", ")).Msg("Watch: reloading workers")

	next, loadErr := launchWorkers(temporalClient, opts, envvars)
	if loadErr != nil {
		log.Error().Err(loadErr).Msg("Watch: reload failed, keeping existing workers")
	} else {
		stopWorkerList(current)
		current = next
		log.Info().Int("count", len(current)).Msg("Watch: workers reloaded successfully")
	}
	if err := refreshWatcher(watcher, opts); err != nil {
		log.Error().Err(err).Msg("Watch: failed to refresh file watches")
	}
	return current, make(map[string]struct{})
}

// refreshWatcher removes all currently watched paths and re-adds the resolved
// workflow files. This is called after every debounce-triggered reload (whether
// it succeeded or failed) to recover watches that were lost because an editor
// replaced a file via a temp-file rename, which causes fsnotify to silently
// drop the watch on the original inode.
//
// Refresh is two-phase: all target files are added first (fsnotify.Add is
// idempotent for already-watched paths), and stale paths are only removed after
// every add succeeds. This means a failed add leaves the previous watch set
// intact rather than leaving watch mode partially disabled.
func refreshWatcher(w *fsnotify.Watcher, opts *runOptions) error {
	// Discover files first. If discovery fails, leave the watcher unchanged so
	// subsequent events can still be received.
	files, err := discoverWorkflowFiles(opts)
	if err != nil {
		return err
	}

	// Phase 1: add all target files. fsnotify.Add is idempotent for paths that
	// are already watched, so this also refreshes inodes lost by rename-style
	// saves. On any failure, return before touching the existing watch list.
	target := make(map[string]struct{}, len(files))
	for _, f := range files {
		target[f] = struct{}{}
		if err := w.Add(f); err != nil {
			return fmt.Errorf("watch: re-add %s: %w", f, err)
		}
	}

	// Phase 2: remove paths that are no longer in the target set.
	for _, p := range w.WatchList() {
		if _, ok := target[p]; !ok {
			_ = w.Remove(p)
		}
	}
	return nil
}

// runWatchMode watches files for changes and reloads workers on each change.
// It blocks until ctx is cancelled or an interrupt signal is received.
// On reload failure it logs the error and keeps the existing workers running
// so the system is never left with zero workers.
func runWatchMode(
	ctx context.Context,
	files []string,
	temporalClient client.Client,
	opts *runOptions,
	envvars map[string]any,
	current []worker.Worker,
) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watch: create watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()

	for _, f := range files {
		if err := watcher.Add(f); err != nil {
			return fmt.Errorf("watch: add %s: %w", f, err)
		}
	}
	log.Info().
		Int("count", len(files)).
		Dur("debounce", opts.WatchDebounce).
		Msg("Watch: watching workflow files for changes")

	defer func() { stopWorkerList(current) }()

	debounce := newStoppedTimer(opts.WatchDebounce)

	// List of changed files - use map to autodedupe
	changedFiles := make(map[string]struct{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-worker.InterruptCh():
			log.Info().Msg("Watch: received interrupt signal")
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if isWatchableEvent(event) {
				changedFiles[event.Name] = struct{}{}

				log.Debug().
					Str("file", event.Name).
					Str("op", event.Op.String()).
					Msg("Watch: file change detected, debouncing")
				resetDebounce(debounce, opts.WatchDebounce)
			}
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Error().Err(watchErr).Msg("Watch: watcher error")
		case <-debounce.C:
			current, changedFiles = handleDebounce(watcher, temporalClient, opts, envvars, changedFiles, current)
		}
	}
}
