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

package utils

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
)

// ExecuteEvery executes the given function on the duration until the context has stopped.
// The returned cancel function stops the background ticker; callers must call it when done.
func ExecuteEvery(ctx context.Context, duration time.Duration, fn func(context.Context)) (cctx context.Context, cancel func()) {
	doneCh := make(chan struct{})
	var once sync.Once
	cctx = ctx
	cancel = func() { once.Do(func() { close(doneCh) }) }

	go func() {
		ticker := time.NewTicker(duration)
		defer ticker.Stop()

		l := log.With().Ctx(ctx).Dur("duration", duration).Logger()

		for {
			select {
			case <-ticker.C:
				l.Debug().Msg("Triggering background function")
				fn(ctx)
			case <-doneCh:
				l.Debug().Msg("Stopping background function")
				return
			case <-ctx.Done():
				l.Debug().Msg("Stopping background function")
				return
			}
		}
	}()

	return cctx, cancel
}

// Convert the Serverless Workflow duration into a time Duration
func ToDuration(v *model.Duration) (duration time.Duration) {
	if v != nil {
		inline := v.AsInline()

		if inline != nil {
			duration += time.Millisecond * time.Duration(inline.Milliseconds)
			duration += time.Second * time.Duration(inline.Seconds)
			duration += time.Minute * time.Duration(inline.Minutes)
			duration += time.Hour * time.Duration(inline.Hours)
			duration += (time.Hour * 24) * time.Duration(inline.Days)
		}
	}

	return duration
}
