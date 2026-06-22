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

package zigflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/serverlessworkflow/sdk-go/v3/model"
	"github.com/zigflow/zigflow/pkg/utils"
	"github.com/zigflow/zigflow/pkg/zigflow/registry"
	"golang.org/x/sync/errgroup"
)

func ParseUse(wf *model.Workflow) (*registry.Use, error) {
	registry := &registry.Use{
		Catalog: make(map[string]*model.Workflow, 0),
	}

	if wf.Use != nil {
		log.Debug().Msg("Parsing workflow.use definition")
		if len(wf.Use.Catalogs) > 0 {
			if err := parseUseCatalog(registry, wf.Use.Catalogs); err != nil {
				return nil, err
			}
		}
	}

	return registry, nil
}

func parseUseCatalog(registry *registry.Use, catalog map[string]*model.Catalog) error {
	log.Debug().Msg("Parsing workflow catalog")

	var mu sync.Mutex

	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(5)

	for name, def := range catalog {
		g.Go(func() error {
			l := log.With().Str("catalog", name).Str("endpoint", def.Endpoint.String()).Logger()

			l.Debug().Msg("Downloading catalog")
			workflow, err := utils.ReadURLContents(ctx, def.Endpoint.String())
			if err != nil {
				return fmt.Errorf("error downloading catalog %q: %w", name, err)
			}

			defer func() {
				l.Debug().Msg("Unlocking mutex")
				mu.Unlock()
			}()

			l.Debug().Msg("Lock mutex")
			mu.Lock()

			l.Debug().Msg("Catalog found - parsing as workflow")

			if _, ok := registry.Catalog[name]; ok {
				return fmt.Errorf("catalog already exists: %q", name)
			}

			l.Debug().Msg("Loading workflow model")
			wf, _, err := LoadFromBytes(workflow) // @todo(sje): might need to pass in existing registry
			if err != nil {
				return fmt.Errorf("error parsing catalog workflow %q: %w", name, err)
			}

			l.Debug().Msg("Catalog valid - storing in registry")
			registry.Catalog[name] = wf

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
