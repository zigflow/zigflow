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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	zigtemporal "github.com/zigflow/helpers"
)

const lockPollInterval = 100 * time.Millisecond

type fileLockUnlock func() error

// acquireTestRunLock serialises zigflow test runs that share the same Temporal
// server and namespace. Only one process may poll the zigflow-test task queue
// at a time for that pair, so workers do not steal tasks for workflows they
// have not registered.
func acquireTestRunLock(ctx context.Context, temporal *zigtemporal.TemporalOpts) (release func(), err error) {
	path := testRunLockPath(temporal)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create zigflow test lock directory: %w", err)
	}

	f, err := openTestRunLockFile(path)
	if err != nil {
		return nil, fmt.Errorf("open zigflow test lock file: %w", err)
	}

	for {
		acquired, unlock, lockErr := tryLockFile(f)
		if lockErr != nil {
			_ = f.Close()
			return nil, lockErr
		}
		if acquired {
			return func() {
				if unlock != nil {
					_ = unlock()
				}
				_ = f.Close()
			}, nil
		}

		if err := ctx.Err(); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("waiting for zigflow test lock: %w", err)
		}

		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, fmt.Errorf("waiting for zigflow test lock: %w", ctx.Err())
		case <-time.After(lockPollInterval):
		}
	}
}

func testRunLockPath(temporal *zigtemporal.TemporalOpts) string {
	addr := "localhost:7233"
	namespace := "default"
	if temporal != nil {
		if temporal.Address != "" {
			addr = temporal.Address
		}
		if temporal.Namespace != "" {
			namespace = temporal.Namespace
		}
	}

	key := sha256.Sum256([]byte(addr + "\n" + namespace))
	name := "zigflow-test-" + hex.EncodeToString(key[:8]) + ".lock"

	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}

	return filepath.Join(base, "zigflow", "test-locks", name)
}
