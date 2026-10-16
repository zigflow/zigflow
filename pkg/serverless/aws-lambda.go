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

package serverless

import (
	"context"

	"go.temporal.io/sdk/contrib/aws/lambdaworker"
	"go.temporal.io/sdk/worker"
)

func NewAWSLambda(ctx context.Context, sOpts Options) error {
	lambdaworker.RunWorker(worker.WorkerDeploymentVersion{
		BuildID:        sOpts.BuildID,
		DeploymentName: sOpts.DeploymentName,
	}, func(opts *lambdaworker.Options) error {
		return nil
	})

	return nil
}
