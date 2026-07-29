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

package models

import "github.com/open-workflow-specification/sdk-go/v4/model"

const (
	ErrorTypeConfiguration  = "https://zigflow.dev/spec/1.0.0/errors/configuration"
	ErrorTypeValidation     = "https://zigflow.dev/spec/1.0.0/errors/validation"
	ErrorTypeExpression     = "https://zigflow.dev/spec/1.0.0/errors/expression"
	ErrorTypeAuthentication = "https://zigflow.dev/spec/1.0.0/errors/authentication"
	ErrorTypeAuthorization  = "https://zigflow.dev/spec/1.0.0/errors/authorization"
	ErrorTypeTimeout        = "https://zigflow.dev/spec/1.0.0/errors/timeout"
	ErrorTypeCommunication  = "https://zigflow.dev/spec/1.0.0/errors/communication"
	ErrorTypeRuntime        = "https://zigflow.dev/spec/1.0.0/errors/runtime"
)

func NewErrConfiguration(detail error, instance string) *model.Error {
	return newError(
		ErrorTypeConfiguration,
		400,
		"Configuration Error",
		detail,
		instance,
	)
}

func NewErrValidation(detail error, instance string) *model.Error {
	return newError(
		ErrorTypeValidation,
		400,
		"Validation Error",
		detail,
		instance,
	)
}

func NewErrExpression(detail error, instance string) *model.Error {
	return newError(
		ErrorTypeExpression,
		400,
		"Expression Error",
		detail,
		instance,
	)
}

func NewErrAuthentication(detail error, instance string) *model.Error {
	return newError(
		ErrorTypeAuthentication,
		401,
		"Authentication Error",
		detail,
		instance,
	)
}

func NewErrAuthorization(detail error, instance string) *model.Error {
	return newError(
		ErrorTypeAuthorization,
		403,
		"Authorization Error",
		detail,
		instance,
	)
}

func NewErrTimeout(detail error, instance string) *model.Error {
	return newError(
		ErrorTypeTimeout,
		408,
		"Timeout Error",
		detail,
		instance,
	)
}

func NewErrCommunication(detail error, instance string) *model.Error {
	return newError(
		ErrorTypeCommunication,
		500,
		"Communication Error",
		detail,
		instance,
	)
}

func NewErrRuntime(detail error, instance string) *model.Error {
	return newError(
		ErrorTypeRuntime,
		500,
		"Runtime Error",
		detail,
		instance,
	)
}

// newError creates a new structured error
func newError(errType string, status int, title string, detail error, instance string) *model.Error {
	if detail != nil {
		return &model.Error{
			Type:   model.NewUriTemplate(errType),
			Status: status,
			Title:  model.NewStringOrRuntimeExpr(title),
			Detail: model.NewStringOrRuntimeExpr(detail.Error()),
			Instance: &model.JsonPointerOrRuntimeExpression{
				Value: instance,
			},
		}
	}

	return &model.Error{
		Type:   model.NewUriTemplate(errType),
		Status: status,
		Title:  model.NewStringOrRuntimeExpr(title),
		Instance: &model.JsonPointerOrRuntimeExpression{
			Value: instance,
		},
	}
}
