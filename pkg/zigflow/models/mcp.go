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

import "github.com/serverlessworkflow/sdk-go/v3/model"

type CallMCP struct {
	model.TaskBase `json:",inline"`
	Call           string       `json:"call"`
	With           MCPArguments `json:"with"`
}

func (c *CallMCP) GetBase() *model.TaskBase {
	return &c.TaskBase
}

type MCPArguments struct {
	Method     string          `json:"method"`
	Parameters any             `json:"parameters"`
	Timeout    *model.Duration `json:"duration"`
	Transport  *MCPTransport   `json:"transport"`
	Client     *MCPClient      `json:"client"`
}

type MCPTransport struct {
	HTTP    *MCPTransportHTTP  `json:"http,omitempty"`
	STDIO   *MCPTransportSTDIO `json:"stdio,omitempty"`
	Options map[string]any     `json:"options"`
}

type MCPTransportHTTP struct {
	Endpoint *model.Endpoint `json:"endpoint"`
	Headers  map[string]any  `json:"headers"`
}

type MCPTransportSTDIO struct {
	Command     string   `json:"command"`
	Arguments   []string `json:"arguments"`
	Environment []string `json:"environment"`
}

type MCPClient struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
