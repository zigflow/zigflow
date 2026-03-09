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

package metadata

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var ErrInvalidType = fmt.Errorf("invalid type")

const (
	SearchAttributeDateTimeType    string = "datetime"
	SearchAttributeKeywordListType string = "keywordlist"
	SearchAttributeKeywordType     string = "keyword"
	SearchAttributeTextType        string = "text"
	SearchAttributeIntType         string = "int"
	SearchAttributeDoubleType      string = "double"
	SearchAttributeBooleanType     string = "bool"
)

type SearchAttribute struct {
	Type  string `json:"type"`
	Value any    `json:"value"` // If nil then the value is unset
}

func (v *SearchAttribute) newBooleanUpdate(key string) (temporal.SearchAttributeUpdate, error) {
	s := temporal.NewSearchAttributeKeyBool(key)
	if v.Value == nil {
		return s.ValueUnset(), nil
	}
	switch e := v.Value.(type) {
	case bool:
		return s.ValueSet(e), nil
	case string:
		i, err := strconv.ParseBool(e)
		if err != nil {
			return nil, fmt.Errorf("error converting string to bool")
		}
		return s.ValueSet(i), nil
	default:
		return nil, ErrInvalidType
	}
}

func (v *SearchAttribute) newDateTimeUpdate(key string) (temporal.SearchAttributeUpdate, error) {
	s := temporal.NewSearchAttributeKeyTime(key)
	if v.Value == nil {
		return s.ValueUnset(), nil
	}

	switch e := v.Value.(type) {
	case time.Time:
		return s.ValueSet(e), nil
	case string:
		t, err := time.Parse(time.RFC3339, e)
		if err != nil {
			return nil, fmt.Errorf("error parsing datetime string: %w", err)
		}
		return s.ValueSet(t), nil
	default:
		return nil, ErrInvalidType
	}
}

func (v *SearchAttribute) newFloatUpdate(key string) (temporal.SearchAttributeUpdate, error) {
	s := temporal.NewSearchAttributeKeyFloat64(key)
	if v.Value == nil {
		return s.ValueUnset(), nil
	}

	var val float64
	switch e := v.Value.(type) {
	case int:
		val = float64(e)
	case int32:
		val = float64(e)
	case int64:
		val = float64(e)
	case float32:
		val = float64(e)
	case float64:
		val = float64(e)
	case string:
		var err error
		val, err = strconv.ParseFloat(e, 64)
		if err != nil {
			return nil, fmt.Errorf("error converting string to float64: %w", err)
		}
	default:
		return nil, ErrInvalidType
	}

	return s.ValueSet(val), nil
}

func (v *SearchAttribute) newIntegerUpdate(key string) (temporal.SearchAttributeUpdate, error) {
	s := temporal.NewSearchAttributeKeyInt64(key)
	if v.Value == nil {
		return s.ValueUnset(), nil
	}

	var val int64
	switch e := v.Value.(type) {
	case int:
		val = int64(e)
	case int32:
		val = int64(e)
	case int64:
		val = e
	case float32:
		val = int64(e)
	case float64:
		val = int64(e)
	case string:
		var err error
		val, err = strconv.ParseInt(e, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error converting string to int64: %w", err)
		}
	default:
		return nil, ErrInvalidType
	}

	return s.ValueSet(val), nil
}

func (v *SearchAttribute) newKeywordListUpdate(key string) temporal.SearchAttributeUpdate {
	s := temporal.NewSearchAttributeKeyKeywordList(key)
	if v.Value == nil {
		return s.ValueUnset()
	}
	return s.ValueSet(v.Value.([]string))
}

func (v *SearchAttribute) newKeywordUpdate(key string) temporal.SearchAttributeUpdate {
	s := temporal.NewSearchAttributeKeyKeyword(key)
	if v.Value == nil {
		return s.ValueUnset()
	}
	return s.ValueSet(v.Value.(string))
}

func (v *SearchAttribute) newTextUpdate(key string) temporal.SearchAttributeUpdate {
	s := temporal.NewSearchAttributeKeyString(key)
	if v.Value == nil {
		return s.ValueUnset()
	}
	return s.ValueSet(v.Value.(string))
}

// Sets by type. See the Temporal documentation for what these all mean
// @link https://docs.temporal.io/search-attribute#custom-search-attribute-limits
func (v *SearchAttribute) setAttribute(key string) (temporal.SearchAttributeUpdate, error) {
	switch strings.ToLower(v.Type) {
	case SearchAttributeBooleanType:
		// Boolean
		return v.newBooleanUpdate(key)

	case SearchAttributeDateTimeType:
		// DateTime
		return v.newDateTimeUpdate(key)

	case SearchAttributeDoubleType:
		// Floating point number
		return v.newFloatUpdate(key)

	case SearchAttributeIntType:
		// Integer
		return v.newIntegerUpdate(key)

	case SearchAttributeKeywordType:
		// Keyword
		return v.newKeywordUpdate(key), nil

	case SearchAttributeKeywordListType:
		// Keyword List
		return v.newKeywordListUpdate(key), nil

	case SearchAttributeTextType:
		// Text
		return v.newTextUpdate(key), nil

	default:
		return nil, fmt.Errorf("unknown search attribute type: %s", v.Type)
	}
}

func ParseSearchAttributes(ctx workflow.Context, metadata any) error {
	logger := workflow.GetLogger(ctx)

	search, ok := metadata.(map[string]any)
	if !ok {
		return fmt.Errorf("search attributes in invalid format")
	}

	var searchAttributes map[string]*SearchAttribute
	if err := mapstructure.Decode(search, &searchAttributes); err != nil {
		return fmt.Errorf("error converting attributes to golang struct: %w", err)
	}

	signedAttributes := make([]temporal.SearchAttributeUpdate, 0)

	for k, v := range searchAttributes {
		if attr, err := v.setAttribute(k); err != nil {
			logger.Error("Error setting search attribute", "error", err)
			return fmt.Errorf("error setting search attribute: %w", err)
		} else {
			signedAttributes = append(signedAttributes, attr)
		}
	}

	if len(signedAttributes) == 0 {
		return nil
	}

	logger.Debug("setting search attribute")
	if err := workflow.UpsertTypedSearchAttributes(ctx, signedAttributes...); err != nil {
		logger.Error("Error upserting search attributes", "error", err)
		return fmt.Errorf("error upserting search attributes: %w", err)
	}

	return nil
}
