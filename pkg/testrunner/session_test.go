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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zigflow/zigflow/pkg/zigflow/activities"
	"github.com/zigflow/zigflow/pkg/zigflow/tasks"
)

func TestSessionTaskOptionsPreservesCallerConfiguration(t *testing.T) {
	configured := &tasks.TaskOpts{Run: &tasks.RunTaskOpts{Namespace: "sdk-namespace", ServiceAccount: "sdk-service"}}
	original := *configured.Run
	options, err := sessionTaskOptions(configured)
	require.NoError(t, err)
	assert.Equal(t, activities.ContainerRuntimeDocker, options.Run.Runtime)
	assert.Equal(t, original.Namespace, options.Run.Namespace)
	assert.Equal(t, original.ServiceAccount, options.Run.ServiceAccount)
	assert.Equal(t, original, *configured.Run)

	configured.Run.Runtime = activities.ContainerRuntimeKubernetes
	options, err = sessionTaskOptions(configured)
	require.NoError(t, err)
	assert.Equal(t, activities.ContainerRuntimeKubernetes, options.Run.Runtime)
}

func TestSessionTaskOptionsRejectsInvalidRuntime(t *testing.T) {
	options, err := sessionTaskOptions(&tasks.TaskOpts{Run: &tasks.RunTaskOpts{Runtime: "invalid"}})
	require.ErrorContains(t, err, "invalid container runtime")
	assert.Nil(t, options)
}
