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

package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCloudflareTrace(t *testing.T) {
	tests := []struct {
		Name     string
		Body     string
		Expected string
	}{
		{
			Name: "valid response with loc=GB",
			Body: "fl=123abc\nh=www.cloudflare.com\nip=1.2.3.4\nts=1234567890.123\n" +
				"visit_scheme=https\nuag=Go-http-client/1.1\ncolo=LHR\nsliver=none\n" +
				"http=http/1.1\nloc=GB\ntls=TLSv1.3\nsni=plaintext\nwarp=off",
			Expected: "GB",
		},
		{
			Name:     "lowercase country code",
			Body:     "fl=123abc\nip=1.2.3.4\nloc=gb",
			Expected: "",
		},
		{
			Name:     "mixed case country code",
			Body:     "fl=123abc\nip=1.2.3.4\nloc=Gb",
			Expected: "",
		},
		{
			Name:     "country code with space",
			Body:     "fl=123abc\nip=1.2.3.4\nloc=G B",
			Expected: "",
		},
		{
			Name:     "country code with subdivision",
			Body:     "fl=123abc\nip=1.2.3.4\nloc=GB-LND",
			Expected: "",
		},
		{
			Name:     "country code as full name",
			Body:     "fl=123abc\nip=1.2.3.4\nloc=London",
			Expected: "",
		},
		{
			Name:     "missing loc",
			Body:     "fl=123abc\nip=1.2.3.4\nts=1234567890.123",
			Expected: "",
		},
		{
			Name:     "malformed input",
			Body:     "this is not valid",
			Expected: "",
		},
		{
			Name:     "empty body",
			Body:     "",
			Expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, parseCloudflareTrace(test.Body))
		})
	}
}

func TestFetchCountry(t *testing.T) {
	validBody := "fl=123abc\nh=www.cloudflare.com\nip=1.2.3.4\nts=1234567890.123\n" +
		"visit_scheme=https\nuag=Go-http-client/1.1\ncolo=LHR\nsliver=none\n" +
		"http=http/1.1\nloc=GB\ntls=TLSv1.3\nsni=plaintext\nwarp=off"

	tests := []struct {
		Name     string
		Status   int
		Body     string
		Expected string
	}{
		{
			Name:     "200 with valid loc",
			Status:   http.StatusOK,
			Body:     validBody,
			Expected: "GB",
		},
		{
			Name:     "500 response returns empty",
			Status:   http.StatusInternalServerError,
			Body:     validBody,
			Expected: "",
		},
		{
			Name:     "404 response returns empty",
			Status:   http.StatusNotFound,
			Body:     validBody,
			Expected: "",
		},
		{
			Name:     "200 with no loc line returns empty",
			Status:   http.StatusOK,
			Body:     "fl=123abc\nip=1.2.3.4\nts=1234567890.123",
			Expected: "",
		},
		{
			Name:   "200 with oversized body containing loc near start",
			Status: http.StatusOK,
			// loc=GB within the first 4KB, padded well beyond the limit
			Body:     "fl=123abc\nloc=GB\npadding=" + strings.Repeat("x", 12*1024),
			Expected: "GB",
		},
		{
			Name:   "200 with oversized body where loc is beyond limit",
			Status: http.StatusOK,
			// loc=GB pushed past the 4KB read limit
			Body:     strings.Repeat("x=y\n", 2*1024) + "loc=GB\n",
			Expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.Status)
				_, _ = w.Write([]byte(test.Body))
			}))
			defer srv.Close()

			result := fetchCountry(srv.Client(), srv.URL)
			assert.Equal(t, test.Expected, result)
		})
	}
}
