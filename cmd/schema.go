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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newSchemaCmd() *cobra.Command {
	var opts struct {
		Output string
	}

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Output the Zigflow JSON schema.",
		Long: `Output the JSON Schema for the Zigflow workflow specification.

The schema can be used by editors, validation tools and AI code
generators to produce structurally valid Zigflow workflows. It defines
required fields, supported properties and constraints enforced by the
Zigflow CLI.

By exposing the schema programmatically, Zigflow enables reliable
validation, structured generation and automated tooling integration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := marshalSchema(opts.Output)
			if err != nil {
				return err
			}

			fmt.Print(string(data))

			return nil
		},
	}

	viper.Set("output", "json")
	cmd.Flags().StringVarP(
		&opts.Output, "output", "o",
		viper.GetString("output"), "Output format. One of: (json, yaml)",
	)

	return cmd
}
