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

package tasks

import (
	"sync"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
)

// Tracks (worker, name) pairs already registered so repeated Build
// calls do not double-register. Temporal's RegisterActivityWithOptions
// panics on duplicate names.
var activityRegistry sync.Map

type activityRegistryKey struct {
	worker worker.Worker
	name   string
}

// Registers fn on w under name, skipping if (w, name) was already seen.
func registerActivityOnce(w worker.Worker, fn any, name string) {
	if w == nil || name == "" {
		return
	}

	key := activityRegistryKey{worker: w, name: name}
	if _, loaded := activityRegistry.LoadOrStore(key, struct{}{}); loaded {
		return
	}

	w.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: name})
}
