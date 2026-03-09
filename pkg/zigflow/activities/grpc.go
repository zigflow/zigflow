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
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/fullstorydev/grpcurl"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/metadata"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	Registry = append(Registry, &CallGRPC{})
}

type CallGRPC struct{}

func (c *CallGRPC) CallGRPCActivity(
	ctx context.Context, task *model.CallGRPC, input any, state *utils.State,
) (any, error) {
	logger := activity.GetLogger(ctx)

	stopHeartbeat := metadata.StartActivityHeartbeat(ctx, task.GetBase())
	defer stopHeartbeat()

	ob, err := utils.TraverseAndEvaluateObj(model.NewObjectOrRuntimeExpr(map[string]any{
		"service": task.With.Service,
		"args":    task.With.Arguments,
		"method":  task.With.Method,
		"proto":   task.With.Proto,
	}), nil, state)
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError("error traversing grpc data object", "CallGRPC error", err)
	}

	obj := ob.(map[string]any)

	service := obj["service"].(model.GRPCService)
	args := obj["args"].(map[string]any)
	method := obj["method"].(string)
	proto := obj["proto"].(*model.ExternalResource)

	address := fmt.Sprintf("%s:%d", service.Host, service.Port)

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Error creating gRPC connection", "error", err)
		return nil, err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			logger.Error("Error closing body reader", "error", err)
		}
	}()

	u, err := url.Parse(proto.Endpoint.String())
	if err != nil {
		return nil, err
	}

	descriptorSource, err := grpcurl.DescriptorSourceFromProtoFiles([]string{"/"}, u.Path)
	if err != nil {
		logger.Error("Error loading proto file", "error", err, "file", u.Path)
		return nil, temporal.NewNonRetryableApplicationError("error loading protofile", "CallGRPC error", err)
	}

	jsonRequest, err := json.Marshal(args)
	if err != nil {
		logger.Error("Error converting arguments to JSON", "error", err)
		return nil, temporal.NewNonRetryableApplicationError("error converting arguments to json", "CallGRPC error", err)
	}

	options := grpcurl.FormatOptions{EmitJSONDefaultFields: true}
	jsonRequestReader := strings.NewReader(string(jsonRequest))
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.Format("json"), descriptorSource, jsonRequestReader, options)
	if err != nil {
		return nil, err
	}
	var resp bytes.Buffer
	eventHandler := &grpcurl.DefaultEventHandler{
		Out:            &resp,
		Formatter:      formatter,
		VerbosityLevel: 0,
	}

	methodFullName := fmt.Sprintf("%s/%s", service.Name, method)

	if err := grpcurl.InvokeRPC(
		ctx,
		descriptorSource,
		conn,
		methodFullName,
		[]string{},
		eventHandler,
		rf.Next,
	); err != nil {
		return nil, temporal.NewNonRetryableApplicationError("error loading protofile", "CallGRPC error", err)
	}

	var output map[string]any
	if err := json.Unmarshal(resp.Bytes(), &output); err != nil {
		logger.Warn("Cannot convert gRPC response to JSON - returning as string")
		return resp.String(), err
	}

	logger.Debug("Returning gRPC response as JSON")
	return output, err
}
