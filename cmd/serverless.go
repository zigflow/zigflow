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
	"github.com/go-playground/validator/v10"
	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zigflow/zigflow/pkg/serverless"
)

func newServerlessCmd() *cobra.Command {
	var opts serverless.Options

	cmd := &cobra.Command{
		Use:   "serverless <provider>",
		Short: "Run Zigflow workers using serverless compute",
		Long: `Run Zigflow workers using a supported serverless compute provider.

Serverless workers use provider-managed lifecycle behaviour rather than
running as persistent worker processes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			validate := validator.New(validator.WithRequiredStructEnabled())
			if err := validate.Struct(opts); err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Invalid options for Serverless worker",
				}
			}

			if len(args) == 0 {
				return gh.FatalError{
					Msg: "Serverless worker provider is required",
				}
			}

			provider := args[0]

			log.Info().
				Str("provider", provider).
				Str("build-id", opts.BuildID).
				Str("deployment-name", opts.DeploymentName).
				Msg("Running Zigflow as a Serverless Worker")

			var err error
			switch provider {
			case "aws-lambda", "lambda":
				err = serverless.NewAWSLambda(cmd.Context(), opts)
			default:
				return gh.FatalError{
					WithParams: func(l *zerolog.Event) *zerolog.Event {
						return l.Str("provider", provider)
					},
					Msg: "Unknown Serverless provider",
				}
			}
			if err != nil {
				return gh.FatalError{
					Cause: err,
					WithParams: func(l *zerolog.Event) *zerolog.Event {
						return l.Str("provider", provider)
					},
					Msg: "Error running Serverless worker",
				}
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(
		&opts.BuildID, "build-id",
		viper.GetString("build_id"), "The build id specific to this worker",
	)

	cmd.PersistentFlags().StringVar(
		&opts.DeploymentName, "deployment-name",
		viper.GetString("deployment_name"), "The name of the deployment this worker version belongs to",
	)

	return cmd
}
