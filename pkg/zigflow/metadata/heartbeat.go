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
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"go.temporal.io/sdk/activity"
)

const HeartbeatDurationWarning = time.Second * 10

func StartActivityHeartbeat(ctx context.Context, task *model.TaskBase) (stop func()) {
	stop = func() {
		log.Trace().Msg("Activity heartbeat noop")
	}

	if hb, ok := task.Metadata[MetadataHeartbeat]; ok {
		var heartbeat *model.Duration
		if err := utils.ToType(hb, &heartbeat); err != nil {
			// Ignore an invalid heartbeat duration with warning
			log.Warn().Err(err).Any("heartbeat", hb).Msg("Heartbeat metadata not a Duration type")
			return stop
		}

		// Each heartbeat is one action. At scale, this may exceed your allocation and cost extra
		// @link https://temporal.io/pricing
		heartbeatDuration := utils.ToDuration(heartbeat)

		heartbeatTimeout := activity.GetInfo(ctx).HeartbeatTimeout
		l := log.With().
			Dur("duration", heartbeatDuration).
			Dur("heartbeatTimeout", heartbeatTimeout).
			Logger()

		if heartbeatDuration < HeartbeatDurationWarning {
			l.Warn().
				Dur("threshold", HeartbeatDurationWarning).
				Msg("Heartbeat time is below warning threshold - this may increase your Temporal costs")
		}

		if heartbeatTimeout <= heartbeatDuration {
			l.Error().Msg("Heartbeat duration is less than the timeout.")
		}

		count := 0
		_, cancel := utils.ExecuteEvery(ctx, heartbeatDuration, func(hctx context.Context) {
			l.Trace().Int("count", count).Msg("Triggering heartbeat")
			activity.RecordHeartbeat(hctx)
		})

		stop = func() {
			l.Debug().Int("count", count).Msg("Ending heartbeat")
			cancel()
		}
	}

	return stop
}
