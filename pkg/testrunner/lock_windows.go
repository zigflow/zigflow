//go:build windows

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
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// openTestRunLockFile opens the lock file with FILE_FLAG_OVERLAPPED so
// LockFileEx honours the overlapped structure. Without that flag Windows
// ignores lpOverlapped and can block in LockFileEx under contention, which
// prevents acquireTestRunLock from observing context cancellation.
func openTestRunLockFile(path string) (*os.File, error) {
	pathUTF16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	h, err := windows.CreateFile(
		pathUTF16,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OVERLAPPED,
		0,
	)
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(h), path), nil
}

func tryLockFile(f *os.File) (bool, fileLockUnlock, error) {
	h := windows.Handle(f.Fd())

	ol := &windows.Overlapped{}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return false, nil, err
	}
	ol.HEvent = event

	err = windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, ol)
	switch err {
	case nil:
	case windows.ERROR_IO_PENDING:
		wait, err := windows.WaitForSingleObject(ol.HEvent, 0)
		if err != nil {
			_ = windows.CloseHandle(ol.HEvent)
			return false, nil, err
		}
		if wait == uint32(windows.WAIT_TIMEOUT) {
			_ = windows.CancelIoEx(h, ol)
			_ = windows.CloseHandle(ol.HEvent)
			return false, nil, nil
		}
		if wait != uint32(windows.WAIT_OBJECT_0) {
			_ = windows.CloseHandle(ol.HEvent)
			return false, nil, fmt.Errorf("unexpected lock wait result: %d", wait)
		}
	case windows.ERROR_LOCK_VIOLATION:
		_ = windows.CloseHandle(ol.HEvent)
		return false, nil, nil
	default:
		_ = windows.CloseHandle(ol.HEvent)
		return false, nil, err
	}

	unlock := func() error {
		defer func() { _ = windows.CloseHandle(ol.HEvent) }()
		return windows.UnlockFileEx(h, 0, 1, 0, ol)
	}
	return true, unlock, nil
}
