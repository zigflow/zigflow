/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/mrsimonemms/zigflow/graphs/contributors>
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
import { env } from '$env/dynamic/public';
import { fromYAML } from '$lib/tasks';
import { error, json } from '@sveltejs/kit';
import { writeFile } from 'fs/promises';
import { join, normalize } from 'path';

import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ params, request }) => {
  const workflowPath = params.workflowId;

  // Normalize the path to prevent directory traversal
  const normalizedPath = normalize(workflowPath).replace(
    /^(\.\.(\/|\\|$))+/,
    '',
  );
  const fullPath = join(env.PUBLIC_WORKFLOWS_DATA_DIR, normalizedPath);

  // Verify the resolved path is still within the data directory
  if (!fullPath.startsWith(env.PUBLIC_WORKFLOWS_DATA_DIR)) {
    throw error(400, 'Invalid workflow path: attempted directory traversal');
  }

  // Parse the request body
  const body = await request.json();
  const yamlContent = body.yaml;

  if (typeof yamlContent !== 'string') {
    throw error(400, 'Invalid request: yaml content must be a string');
  }

  // Validate the YAML by attempting to parse it
  try {
    fromYAML(yamlContent);
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    throw error(400, `Invalid workflow YAML: ${message}`);
  }

  // Write the file
  try {
    await writeFile(fullPath, yamlContent, 'utf-8');
    return json({ success: true });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    throw error(500, `Failed to save workflow: ${message}`);
  }
};
