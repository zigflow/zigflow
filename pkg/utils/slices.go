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

package utils

// Ensures that all slices are equal to the given value
func SlicesEqual[T comparable](s []T, v T) bool {
	for _, r := range s {
		if r != v {
			return false
		}
	}
	return true
}

// Functionally equivalent to JS's Array.prorotype.every function
// @link https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/every
func SliceEvery[T any](items []T, predicate func(T) bool) bool {
	for _, v := range items {
		if !predicate(v) {
			return false
		}
	}
	return true
}
