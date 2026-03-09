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
	"time"

	basic "app/v1"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// grpc.ServerStreamingServer[Command2Response]) error
func (c *Commands) Command2(request *basic.Command2Request, srv grpc.ServerStreamingServer[basic.Command2Response]) error {
	if err := srv.Send(&basic.Command2Response{
		Message: "Request received - sending no data",
	}); err != nil {
		return fmt.Errorf("errored sending to grpc server: %w", err)
	}

	res1, err := c.client.Run(request.Input1)
	if err != nil {
		return fmt.Errorf("error running db: %w", err)
	}

	if err := srv.Send(&basic.Command2Response{
		Message: "Returning the input2",
		Data:    res1,
	}); err != nil {
		return fmt.Errorf("errored sending to grpc server: %w", err)
	}

	sleep := time.Second * 10
	logrus.WithField("timeout", sleep).Info("Sleeping for effect")

	time.Sleep(sleep)

	res2, err := c.client.Run(request.Input2)
	if err != nil {
		return fmt.Errorf("error running db: %w", err)
	}

	if err := srv.Send(&basic.Command2Response{
		Message: "Returning the input2",
		Data:    res2,
	}); err != nil {
		return fmt.Errorf("errored sending to grpc server: %w", err)
	}

	return nil
}
