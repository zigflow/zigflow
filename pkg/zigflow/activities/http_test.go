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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/open-workflow-specification/sdk-go/v4/model"
	"github.com/stretchr/testify/assert"
)

func TestObjectOrRuntimeExprToMap(t *testing.T) {
	const (
		fieldHeaders = "headers"
		fieldQuery   = "query"
	)

	tests := []struct {
		name    string
		field   string
		input   *model.ObjectOrRuntimeExpr
		want    map[string]any
		wantErr bool
	}{
		{
			name:  "nil pointer returns nil map",
			field: fieldHeaders,
			input: nil,
			want:  nil,
		},
		{
			name:  "nil value returns nil map",
			field: fieldQuery,
			input: model.NewObjectOrRuntimeExpr(nil),
			want:  nil,
		},
		{
			name:  "valid static object is returned",
			field: fieldHeaders,
			input: model.NewObjectOrRuntimeExpr(map[string]any{"Authorization": "Bearer token"}),
			want:  map[string]any{"Authorization": "Bearer token"},
		},
		{
			name:    "non-object headers returns error",
			field:   fieldHeaders,
			input:   model.NewObjectOrRuntimeExpr("not-object"),
			wantErr: true,
		},
		{
			name:    "non-object query returns error",
			field:   fieldQuery,
			input:   model.NewObjectOrRuntimeExpr("not-object"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := objectOrRuntimeExprToMap(tt.field, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCallHTTPParseRetryAfter(t *testing.T) {
	c := &CallHTTP{}

	tests := []struct {
		name      string
		value     string
		want      time.Duration
		tolerance time.Duration
	}{
		{
			name:  "empty value returns zero",
			value: "",
			want:  0,
		},
		{
			name:  "positive integer seconds",
			value: "5",
			want:  5 * time.Second,
		},
		{
			name:  "single second",
			value: "1",
			want:  1 * time.Second,
		},
		{
			name:  "large integer seconds",
			value: "3600",
			want:  3600 * time.Second,
		},
		{
			name:  "zero seconds returns zero",
			value: "0",
			want:  0,
		},
		{
			name:  "negative integer returns zero",
			value: "-5",
			want:  0,
		},
		{
			name:  "extremely large integer returns zero",
			value: "999999999999999999999999999999",
			want:  0,
		},
		{
			name:  "non-numeric non-date string returns zero",
			value: "soon",
			want:  0,
		},
		{
			name:  "float-style value returns zero",
			value: "1.5",
			want:  0,
		},
		{
			name:  "past HTTP-date returns zero",
			value: "Wed, 21 Oct 1970 07:28:00 GMT",
			want:  0,
		},
		{
			name:  "parseable but overflowing integer seconds returns zero",
			value: "10000000000",
			want:  0,
		},
		{
			name:  "maximum safe integer seconds returns duration",
			value: fmt.Sprintf("%d", int64(time.Duration(1<<63-1)/time.Second)),
			want:  time.Duration(int64(time.Duration(1<<63-1)/time.Second)) * time.Second,
		},
		{
			name:  "integer seconds above maximum safe duration returns zero",
			value: fmt.Sprintf("%d", int64(time.Duration(1<<63-1)/time.Second)+1),
			want:  0,
		},
		{
			name:  "overflowing integer seconds that wraps positive returns zero",
			value: "27670116110",
			want:  0,
		},
		{
			name:  "whitespace around integer returns seconds",
			value: " 5 ",
			want:  5 * time.Second,
		},
		{
			name: "future HTTP-date returns seconds",
			value: func() string {
				return time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)
			}(),
			want:      time.Hour,
			tolerance: time.Second,
		},
		{
			name: "future HTTP-date with whitespace returns seconds",
			value: func() string {
				return fmt.Sprintf(" %s ", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
			}(),
			want:      time.Hour,
			tolerance: time.Second,
		},
		{
			name: "near-now HTTP-date returns zero",
			value: func() string {
				return time.Now().UTC().Format(http.TimeFormat)
			}(),
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.parseRetryAfter(tt.value)

			if tt.tolerance > 0 {
				assert.InDelta(t, tt.want.Seconds(), got.Seconds(), tt.tolerance.Seconds())
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
