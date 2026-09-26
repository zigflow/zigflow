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
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	zigtemporal "github.com/zigflow/helpers"
)

func TestTestRunLockPath_differsByNamespace(t *testing.T) {
	a := testRunLockPath(&zigtemporal.TemporalOpts{
		Address:   "localhost:7233",
		Namespace: "ns-a",
	})

	b := testRunLockPath(&zigtemporal.TemporalOpts{
		Address:   "localhost:7233",
		Namespace: "ns-b",
	})

	assert.NotEqual(t, a, b)
}

func TestAcquireTestRunLock_serialisesConcurrentWaiters(t *testing.T) {
	opts := &zigtemporal.TemporalOpts{
		Address:   "127.0.0.1:1",
		Namespace: "lock-test-" + uuid.NewString(),
	}

	var order []int
	var mu sync.Mutex

	firstRelease, err := acquireTestRunLock(context.Background(), opts)
	require.NoError(t, err)

	go func() {
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		time.Sleep(80 * time.Millisecond)
		firstRelease()
	}()

	time.Sleep(20 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secondRelease, err := acquireTestRunLock(ctx, opts)
	require.NoError(t, err)
	defer secondRelease()

	mu.Lock()
	order = append(order, 2)
	mu.Unlock()

	assert.Equal(t, []int{1, 2}, order)
}

func TestAcquireTestRunLock_respectsContextCancel(t *testing.T) {
	opts := &zigtemporal.TemporalOpts{
		Address:   "127.0.0.1:2",
		Namespace: "lock-cancel-" + uuid.NewString(),
	}

	release, err := acquireTestRunLock(context.Background(), opts)
	require.NoError(t, err)
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_, err = acquireTestRunLock(ctx, opts)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
