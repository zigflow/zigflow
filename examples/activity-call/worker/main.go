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

package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
)

func fetchProfile(ctx context.Context, userID, requestID string) (map[string]any, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Fetching profile", "userId", userID)

	tiers := []string{"gold", "silver", "bronze"}
	tier := tiers[rand.IntN(len(tiers))] //nolint:gosec // pseudo-random data is fine for demo purposes

	return map[string]any{
		"userId":    userID,
		"requestId": requestID,
		"tier":      tier,
		"visits":    rand.IntN(50) + 1, //nolint:gosec // pseudo-random data is fine for demo purposes
		"metadata": map[string]string{
			"lastLogin": time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

func generateWelcome(_ context.Context, profile map[string]any) (map[string]string, error) {
	userID := profile["userId"].(string)
	tier := profile["tier"].(string)
	visits := 0
	if v, ok := profile["visits"].(float64); ok {
		visits = int(v)
	}
	message := fmt.Sprintf("Welcome back %s! Your tier is %s and you've visited %d times.", userID, tier, visits)
	return map[string]string{
		"message": message,
		"tier":    tier,
	}, nil
}

func main() {
	const activityTaskQueue = "activity-call-worker"
	c, err := temporal.NewConnectionWithEnvvars(temporal.WithZerolog(&log.Logger))
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create Temporal client")
	}
	defer c.Close()

	w := worker.New(c, activityTaskQueue, worker.Options{})

	w.RegisterActivityWithOptions(fetchProfile, activity.RegisterOptions{Name: "activitycall.FetchProfile"})
	w.RegisterActivityWithOptions(generateWelcome, activity.RegisterOptions{Name: "activitycall.GenerateWelcome"})

	log.Info().Str("taskQueue", activityTaskQueue).Msg("Activity worker started - Waiting for commands")
	if err := w.Run(worker.InterruptCh()); err != nil {
		//nolint:gocritic
		log.Fatal().Err(err).Msg("Worker exited with error")
	}
}
