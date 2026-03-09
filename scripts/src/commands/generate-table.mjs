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
import fs from 'node:fs/promises';
import path from 'node:path';
import yaml from 'yaml';

export const command = 'generate-table <file> <dir>';
export const describe = 'Generate the Examples template';

export const builder = (y) =>
  y
    .positional('file', {
      type: 'string',
      describe: 'Path to readme file',
    })
    .positional('dir', {
      type: 'string',
      describe: 'Path to examples directory',
    });

export const handler = async (argv) => {
  // Get all the data
  const files = (
    await Promise.all(
      (await fs.readdir(argv.dir, { withFileTypes: true }))
        .filter((d) => d.isDirectory())
        .map(async (d) => {
          const workflowFile = path.join(argv.dir, d.name, 'workflow.yaml');

          const { document } = yaml.parse(
            await fs.readFile(workflowFile, 'utf-8'),
          );

          return {
            name: `[${document.title ?? document.name}](./${d.name})`,
            description: document.summary,
          };
        }),
    )
  ).sort((a, b) => {
    if (a.title > b.title) {
      return 1;
    }
    if (a.title < b.title) {
      return -1;
    }
    return 0;
  });

  // Generate the table
  const table = ['| Name | Description |', '| --- | --- |'];

  files.forEach(({ name, description }) => {
    table.push(`| ${name} | ${description} |`);
  });

  const fileContents = await fs.readFile(argv.file, 'utf-8');

  const updated = replaceBetweenAnchors(fileContents, table.join('\n'));

  await fs.writeFile(argv.file, updated);
};

function replaceBetweenAnchors(content, replacement) {
  const pattern = /<!-- apps-start -->([\s\S]*?)<!-- apps-end -->/;
  return content.replace(
    pattern,
    `<!-- apps-start -->\n\n${replacement}\n\n<!-- apps-end -->`,
  );
}
