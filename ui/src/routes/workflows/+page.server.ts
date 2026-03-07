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
import { redirect } from '@sveltejs/kit';
import { promises as fs } from 'fs';
import yaml from 'js-yaml';
import { resolve } from 'path';

import type { Actions, PageServerLoad } from './$types';

const WORKFLOWS_DIR =
  env.PUBLIC_WORKFLOWS_DIR ?? resolve(process.cwd(), 'workflows');

export const load: PageServerLoad = async () => {
  let workflowFiles: string[] = [];
  try {
    const entries = await fs.readdir(WORKFLOWS_DIR);
    workflowFiles = entries
      .filter((f) => f.endsWith('.yaml') || f.endsWith('.yml'))
      .sort();
  } catch {
    // Directory does not exist yet — return empty list.
  }
  return { workflowFiles };
};

export const actions: Actions = {
  default: async ({ request }) => {
    const formData = await request.formData();
    const rawName = String(formData.get('name') ?? '').trim();
    if (!rawName) {
      return { error: 'Workflow name is required' };
    }

    // Sanitise: lowercase, alphanumeric and hyphens only.
    const name = rawName
      .toLowerCase()
      .replace(/[^a-z0-9-]/g, '-')
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, '');

    if (!name) {
      return {
        error: 'Workflow name must contain at least one valid character',
      };
    }

    const fileName = `${name}.yaml`;
    const filePath = resolve(WORKFLOWS_DIR, fileName);

    const skeleton = yaml.dump(
      {
        document: {
          dsl: '1.0.0',
          namespace: 'default',
          name,
          version: '0.0.1',
          title: name,
        },
        do: [],
      },
      { indent: 2, lineWidth: -1, noRefs: true, sortKeys: false },
    );

    await fs.mkdir(WORKFLOWS_DIR, { recursive: true });
    await fs.writeFile(filePath, skeleton, 'utf-8');

    redirect(302, `/workflows/${fileName}`);
  },
};
