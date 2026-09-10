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

package activities

import (
	"encoding/json"
	"testing"
)

// The body is held as JSON regardless of the declared Content-Type, so an
// endpoint expecting application/x-www-form-urlencoded receives a JSON object
// under a header promising a form. encodeHTTPBody picks the wire encoding from
// the declared Content-Type instead.
func TestEncodeHTTPBodyFormURLEncoded(t *testing.T) {
	body := json.RawMessage(`{"grant_type":"client_credentials","scope":"read write"}`)

	got := string(encodeHTTPBody(body, "application/x-www-form-urlencoded"))

	// url.Values.Encode sorts keys, so the expectation is stable.
	want := "grant_type=client_credentials&scope=read+write"
	if got != want {
		t.Errorf("form encoding\n got: %s\nwant: %s", got, want)
	}
}

func TestEncodeHTTPBodyLeavesJSONUntouched(t *testing.T) {
	body := json.RawMessage(`{"a":"b"}`)

	for _, contentType := range []string{
		"application/json",
		"application/json; charset=utf-8",
		"application/vnd.api+json",
		"", // unset -- JSON remains the default
	} {
		if got := string(encodeHTTPBody(body, contentType)); got != string(body) {
			t.Errorf("Content-Type %q: body changed\n got: %s\nwant: %s", contentType, got, body)
		}
	}
}

// Header lookup must be case-insensitive: workflows write `content-type` as
// often as `Content-Type`, and Go maps are not case-folding.
func TestHeaderValueIsCaseInsensitive(t *testing.T) {
	for _, key := range []string{"Content-Type", "content-type", "CONTENT-TYPE"} {
		headers := map[string]string{key: "application/x-www-form-urlencoded"}
		if got := headerValue(headers, "Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("header %q not found, got %q", key, got)
		}
	}
}

// A pre-encoded or non-JSON payload must reach the wire as-is rather than
// JSON-quoted, or an XML endpoint receives `"<x/>"` including the quotes.
func TestEncodeHTTPBodyPassesThroughNonJSONPayloads(t *testing.T) {
	body := json.RawMessage(`"<note><to>a</to></note>"`)

	got := string(encodeHTTPBody(body, "application/xml"))

	want := "<note><to>a</to></note>"
	if got != want {
		t.Errorf("XML passthrough\n got: %s\nwant: %s", got, want)
	}
}
