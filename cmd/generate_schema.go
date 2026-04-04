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

package cmd

import (
	"encoding/json"
	"fmt"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zigflow/zigflow/pkg/schema"
	"sigs.k8s.io/yaml"
)

func newGenerateSchemaCmd() *cobra.Command {
	var opts struct {
		Format string
	}

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print the Zigflow JSON schema to stdout",
		Long: `Print the Zigflow JSON schema to stdout.

The schema reflects only the workflow constructs that Zigflow actually supports.
Redirect output to a file if needed, e.g.:

  zigflow generate schema > docs/static/schema.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := marshalSchema(opts.Format)
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), string(data))
			return err
		},
	}

	viper.SetDefault("format", "json")
	cmd.Flags().StringVarP(
		&opts.Format, "format", "o",
		viper.GetString("format"), "Output format. One of: (json, yaml)",
	)

	return cmd
}

// marshalSchema builds the schema and serialises it to the requested format.
func marshalSchema(format string) ([]byte, error) {
	s := schema.BuildSchema(Version, format)

	switch format {
	case "json":
		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return nil, gh.FatalError{
				Cause: err,
				Msg:   "Error marshalling schema to JSON",
			}
		}
		return append(data, '\n'), nil

	case "yaml":
		data, err := yaml.Marshal(s)
		if err != nil {
			return nil, gh.FatalError{
				Cause: err,
				Msg:   "Error marshalling schema to JSON for YAML conversion",
			}
		}
		return data, nil

	default:
		return nil, gh.FatalError{
			Msg: "Invalid format",
			WithParams: func(l *zerolog.Event) *zerolog.Event {
				return l.Str("format", format)
			},
		}
	}
}
