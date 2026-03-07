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
import { exportToYaml } from '$lib/export/yaml';
import { parseWorkflowFile } from '$lib/tasks/parse';
import { error } from '@sveltejs/kit';
import { promises as fs } from 'fs';
import { resolve } from 'path';

import type { PageServerLoad } from './$types';

const WORKFLOWS_DIR =
  env.PUBLIC_WORKFLOWS_DIR ?? resolve(process.cwd(), 'workflows');

export const load: PageServerLoad = async ({ params }) => {
  const fileName = params.workflowId;
  const filePath = resolve(WORKFLOWS_DIR, fileName);

  let content: string;
  try {
    content = await fs.readFile(filePath, 'utf-8');
  } catch {
    throw error(404, `Workflow file not found: ${fileName}`);
  }

  let workflowFile, modified;
  try {
    ({ workflowFile, modified } = parseWorkflowFile(content, fileName));
  } catch (err) {
    throw error(400, `Failed to parse workflow "${fileName}": ${err}`);
  }

  // Write back immediately if any IDs were generated, so deep links remain
  // stable within the same file across server restarts.
  if (modified) {
    const result = exportToYaml(workflowFile);
    if (result.ok) {
      await fs.writeFile(filePath, result.yaml, 'utf-8');
    }
  }

  return { workflowFile };
};
