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
	"log"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func main() {
	p, err := cloudevents.NewHTTP()
	if err != nil {
		log.Fatal(err)
	}

	c, err := cloudevents.NewClient(p)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(c.StartReceiver(context.Background(), receive))
}

func receive(ctx context.Context, event cloudevents.Event) {
	log.Printf("Received event %s of type %s from %s",
		event.ID(),
		event.Type(),
		event.Source(),
	)

	// Access event data
	var data map[string]any
	if err := event.DataAs(&data); err == nil {
		log.Printf("Data: %+v", data)
	}
}
