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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	swUtil "github.com/serverlessworkflow/sdk-go/v3/impl/utils"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

func init() {
	Registry = append(Registry, &CallHTTP{})
}

type httpRetryability int

const (
	httpSuccess httpRetryability = iota
	httpRetryable
	httpNonRetryable
)

// @link: https://github.com/serverlessworkflow/specification/blob/main/dsl-reference.md#http-response
type HTTPResponse struct {
	Request    HTTPRequest       `json:"request"`
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Content    any               `json:"content,omitempty"`
}

// @link: https://github.com/serverlessworkflow/specification/blob/main/dsl-reference.md#http-request
type HTTPRequest struct {
	Method  string            `json:"method"`
	URI     string            `json:"uri"`
	Headers map[string]string `json:"headers,omitempty"`
}

var retryableHTTPStatusCodes = map[int]struct{}{
	408: {}, // Server timeout
	429: {}, // Rate-limited
}

var nonRetryableHTTPStatusCodes = map[int]struct{}{
	501: {}, // Not implemented - no point retrying
}

type CallHTTP struct{}

func (c *CallHTTP) CallHTTPActivity(ctx context.Context, task *model.CallHTTP, input any, state *utils.State) (any, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("Running call HTTP activity")

	stopHeartbeat := metadata.StartActivityHeartbeat(ctx, task.GetBase())
	defer stopHeartbeat()

	state = state.AddActivityInfo(ctx)

	info := activity.GetInfo(ctx)

	resp, method, url, reqHeaders, err := c.callHTTPAction(ctx, task, info.StartToCloseTimeout, state)
	if err != nil {
		logger.Error("Error making HTTP call", "method", method, "url", url, "error", err)
		return nil, err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.Error("Error closing body reader", "error", err)
		}
	}()

	bodyRes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error reading HTTP body", "method", method, "url", url, "error", err)
		return nil, err
	}

	// Try converting the body as JSON, returning as string if not possible
	var content any
	var bodyJSON map[string]any
	if err := json.Unmarshal(bodyRes, &bodyJSON); err != nil {
		// Log error
		logger.Debug("Error converting body to JSON", "error", err)
		content = string(bodyRes)
	} else {
		content = bodyJSON
	}

	switch c.classifyHTTPStatus(resp.StatusCode) {
	case httpRetryable:
		logger.Debug("CallHTTP returned retryable error", "statusCode", resp.StatusCode, "responseBody", content)
		return nil, temporal.NewApplicationError(
			"CallHTTP returned retryable error",
			"CallHTTP retryable error",
			errors.New(resp.Status),
			map[string]any{
				"statusCode": resp.StatusCode,
				"content":    content,
			},
		)
	case httpNonRetryable:
		logger.Error("CallHTTP returned non-retryable error", "statusCode", resp.StatusCode, "responseBody", content)
		return nil, temporal.NewNonRetryableApplicationError(
			"CallHTTP returned non-retryable error",
			"CallHTTP non-retryable error",
			errors.New(resp.Status),
			map[string]any{
				"statusCode": resp.StatusCode,
				"content":    content,
			},
		)
	}

	respHeader := map[string]string{}
	for k, v := range resp.Header {
		respHeader[k] = strings.Join(v, ", ")
	}

	httpResponse := HTTPResponse{
		Request: HTTPRequest{
			Method:  method,
			URI:     url,
			Headers: reqHeaders,
		},
		StatusCode: resp.StatusCode,
		Headers:    respHeader,
		Content:    content,
	}

	return ParseOutput(task.With.Output, httpResponse, bodyRes), err
}

func (c *CallHTTP) callHTTPAction(ctx context.Context, task *model.CallHTTP, timeout time.Duration, state *utils.State) (
	resp *http.Response,
	method, url string,
	reqHeaders map[string]string,
	err error,
) {
	logger := activity.GetLogger(ctx)

	args, err := ParseHTTPArguments(task, state)
	if err != nil {
		return resp,
			method, url,
			reqHeaders,
			err
	}

	method = strings.ToUpper(args.Method)
	url = args.Endpoint.String()
	body := args.Body

	logger.Debug("Making HTTP call", "method", method, "url", url)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Error making HTTP request", "method", method, "url", url, "error", err)
		return resp, method, url, reqHeaders, err
	}

	// Add in headers
	reqHeaders = map[string]string{}
	for k, v := range args.Headers {
		req.Header.Add(k, v)
		reqHeaders[k] = v
	}

	// Add in query strings
	q := req.URL.Query()
	for k, v := range args.Query {
		q.Add(k, v.(string))
	}
	req.URL.RawQuery = q.Encode()

	client := &http.Client{
		Timeout: timeout,
	}

	if !args.Redirect {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// #nosec G704 -- URL is operator-defined in workflow YAML; SSRF is a deployment concern, not a code defect
	resp, err = client.Do(req)
	if err != nil {
		return resp, method, url, reqHeaders, err
	}

	return resp, method, url, reqHeaders, err
}

func (c *CallHTTP) classifyHTTPStatus(code int) httpRetryability {
	if _, ok := retryableHTTPStatusCodes[code]; ok {
		return httpRetryable
	}

	if _, ok := nonRetryableHTTPStatusCodes[code]; ok {
		return httpNonRetryable
	}

	switch {
	case code >= 300 && code < 500:
		return httpNonRetryable
	case code >= 500 && code < 600:
		return httpRetryable
	default:
		return httpSuccess
	}
}

// ParseHTTPArguments note that I looked at the github.com/go-viper/mapstructure/v2.Decode
// function, but this wasn't able to decode some of the more complex data types. This is
// more heavyweight than I'd like, but it's fine for now.
func ParseHTTPArguments(task *model.CallHTTP, state *utils.State) (*model.HTTPArguments, error) {
	// First, we need to convert it to map[string]any
	b, err := json.Marshal(task.With)
	if err != nil {
		return nil, fmt.Errorf("error marshalling object to bytes: %w", err)
	}

	// Next, convert it to a map so we can traverse
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling data to map: %w", err)
	}

	// Clone and traverse, interpolating the data
	cloneData := swUtil.DeepClone(data)
	obj, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(cloneData), nil, state)
	if err != nil {
		return nil, fmt.Errorf("error traversing http data object: %w", err)
	}

	// Now, put it back to a JSON string
	e, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshalling object to bytes: %w", err)
	}

	// Finally, convert back to HTTPArguments
	var result model.HTTPArguments
	if err := json.Unmarshal(e, &result); err != nil {
		return nil, fmt.Errorf("error unmarshalling data to map: %w", err)
	}

	return &result, nil
}

func ParseOutput(outputType string, httpResp HTTPResponse, raw []byte) any {
	var output any
	switch outputType {
	case "raw":
		// Base64 encoded HTTP response content - use the bodyRes
		output = base64.StdEncoding.EncodeToString(raw)
	case "response":
		// HTTP response
		output = httpResp
	default:
		// Content
		output = httpResp.Content
	}

	return output
}
