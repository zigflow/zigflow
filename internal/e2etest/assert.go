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

package e2etest

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// These helpers make shape/type assertions on workflow fields whose values vary
// between runs (UUIDs, clocks). They keep the variable-field checks small and
// consistent across example tests. Workflow output decoded into any uses
// float64 for every JSON number, which is why the numeric helpers expect it.

var (
	uuidRE       = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	datetimeRE   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	iso8601UTCRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
)

// AssertValidUUID asserts the value is a non-empty UUID-looking string.
func AssertValidUUID(t *testing.T, value any) {
	t.Helper()

	s := AssertNonEmptyString(t, value)
	assert.Regexp(t, uuidRE, s, "expected a UUID-looking string")
}

// AssertNonEmptyString asserts the value is a non-empty string and returns it.
func AssertNonEmptyString(t *testing.T, value any) string {
	t.Helper()

	s, ok := value.(string)
	assert.Truef(t, ok, "expected a string, got %T", value)
	assert.NotEmpty(t, s, "expected a non-empty string")
	return s
}

// AssertNumeric asserts the value is a JSON number (decoded as float64).
func AssertNumeric(t *testing.T, value any) float64 {
	t.Helper()

	f, ok := value.(float64)
	assert.Truef(t, ok, "expected a numeric value, got %T", value)
	return f
}

// AssertIntegerLike asserts the value is numeric with no fractional part.
func AssertIntegerLike(t *testing.T, value any) {
	t.Helper()

	f := AssertNumeric(t, value)
	assert.Equalf(t, f, float64(int64(f)), "expected an integer-like value, got %v", f)
}

// AssertDatetimeString asserts the value matches "YYYY-MM-DD HH:MM:SS".
func AssertDatetimeString(t *testing.T, value any) {
	t.Helper()

	s := AssertNonEmptyString(t, value)
	assert.Regexp(t, datetimeRE, s, "expected a YYYY-MM-DD HH:MM:SS datetime")
}

// AssertISO8601UTC asserts the value matches "YYYY-MM-DDTHH:MM:SSZ".
func AssertISO8601UTC(t *testing.T, value any) {
	t.Helper()

	s := AssertNonEmptyString(t, value)
	assert.Regexp(t, iso8601UTCRE, s, "expected a YYYY-MM-DDTHH:MM:SSZ datetime")
}
