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

package codec

import (
	"fmt"

	"go.temporal.io/sdk/converter"
)

type CodecType string

const (
	CodecNone   CodecType = ""
	CodecAES    CodecType = "aes"
	CodecRemote CodecType = "remote"
)

var codecs = map[CodecType]struct{}{
	CodecNone:   {},
	CodecAES:    {},
	CodecRemote: {},
}

func ParseCodecType(t string) (CodecType, error) {
	normalised := CodecType(t)

	if _, ok := codecs[normalised]; ok {
		return normalised, nil
	}

	return "", fmt.Errorf(
		"invalid codec type %q (must be %q, %q or %q)",
		t,
		CodecNone,
		CodecAES,
		CodecRemote,
	)
}

// NewDataConverter constructs a converter.DataConverter for the given CodecType.
// Returns nil for CodecNone, an AES converter for CodecAES (reading keys from
// keyPath), and a RemoteCodecDataConverter for CodecRemote (connecting to endpoint).
func NewDataConverter(codecType CodecType, endpoint, keyPath string, codecHeaders map[string]string) (converter.DataConverter, error) {
	switch codecType {
	case CodecNone:
		return nil, nil
	case CodecAES:
		return NewAESConverter(keyPath)
	case CodecRemote:
		return NewRemoteConverter(endpoint, codecHeaders)
	default:
		return nil, fmt.Errorf("unsupported codec type: %q", codecType)
	}
}
