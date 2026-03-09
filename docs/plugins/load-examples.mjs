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
import fs from 'fs/promises';
import yaml from 'js-yaml';
import path from 'path';

export default function loadExamplesPlugin() {
  return {
    name: 'load-examples',

    async loadContent() {
      const examplesDir = path.resolve(process.cwd(), '..', 'examples');
      const files = await fs.readdir(examplesDir, { withFileTypes: true });

      return (
        await Promise.all(
          files
            .map((item) => {
              if (!item.isDirectory()) {
                return;
              }
              return item;
            })
            .filter((item) => item)
            .map(async (item) => {
              const content = await fs.readFile(
                path.join(item.parentPath, item.name, 'workflow.yaml'),
                'utf8',
              );

              const workflow = yaml.load(content);

              if (!(workflow.document.metadata?.display ?? true)) {
                return;
              }

              return {
                name: item,
                content,
                workflow,
              };
            }),
        )
      )
        .filter((item) => item)
        .sort((a, b) => {
          if (a.workflow.document.title > b.workflow.document.title) {
            return 1;
          }
          if (a.workflow.document.title < b.workflow.document.title) {
            return -1;
          }
          return 0;
        });
    },

    async contentLoaded({ content, actions }) {
      const { setGlobalData } = actions;

      setGlobalData({
        examples: content,
      });
    },
  };
}
