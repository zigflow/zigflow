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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errorRedirectsPath is the documentation redirect mapping consumed by the
// Docusaurus client-redirects plugin (see docs/docusaurus.config.js). It maps
// the /errors/<slug> URL onto an existing documentation page.
var errorRedirectsPath = filepath.Join(
	"..", "..", "docs", "src", "data", "errorRedirects.json",
)

// TestErrorCodesHaveDocumentationRoute is a drift guard: every documentation
// URL the validator can emit must have a corresponding /errors/<slug> redirect
// in the documentation site. If a new error code is added without a route, this
// test fails.
func TestErrorCodesHaveDocumentationRoute(t *testing.T) {
	raw, err := os.ReadFile(errorRedirectsPath)
	require.NoError(t, err, "reading %s", errorRedirectsPath)

	var redirects map[string]string
	require.NoError(t, json.Unmarshal(raw, &redirects))

	for _, code := range ErrorCodes() {
		url := DocumentationURL(code)
		require.NotEmpty(t, url, "code %s must derive a documentation URL", code)

		slug := strings.TrimPrefix(url, errorDocumentationBaseURL)
		require.NotEqual(t, url, slug, "URL %s must use the documentation base URL", url)

		to, ok := redirects[slug]
		assert.Truef(
			t, ok,
			"error code %s (slug %q) has no /errors/%s redirect in %s",
			code, slug, slug, errorRedirectsPath,
		)
		assert.NotEmptyf(t, to, "redirect for slug %q must have a target", slug)
	}
}
