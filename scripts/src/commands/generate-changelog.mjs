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
import { execSync } from 'node:child_process';
import fs from 'node:fs/promises';

export const command = 'generate-changelog <file>';
export const describe = 'Generate CHANGELOG.md from GitHub Releases';

export const builder = (y) =>
  y.positional('file', {
    type: 'string',
    describe: 'Path to write the changelog file',
  });

export const handler = async (argv) => {
  const repo = execSync('gh repo view --json nameWithOwner --jq .nameWithOwner', {
    encoding: 'utf-8',
  }).trim();

  // --paginate fetches all pages; --jq '.[]' emits one release object per line.
  const raw = execSync(`gh api --paginate "/repos/${repo}/releases" --jq '.[]'`, {
    encoding: 'utf-8',
    maxBuffer: 10 * 1024 * 1024,
  });

  const releases = raw
    .trim()
    .split('\n')
    .filter(Boolean)
    .map((line) => JSON.parse(line))
    .filter((r) => !r.draft)
    .sort((a, b) => new Date(b.published_at) - new Date(a.published_at));

  const releasesUrl = `https://github.com/${repo}/releases`;
  const lines = [
    '# Changelog',
    '',
    `This changelog is generated from [GitHub Releases](${releasesUrl}).`,
  ];

  for (const release of releases) {
    const tag = release.tag_name;
    const name = release.name ?? '';
    const published = release.published_at.split('T')[0];
    const body = (release.body ?? '')
      .replace(/\r\n/g, '\n')
      .replace(/\r/g, '\n')
      .trimEnd();

    lines.push('');

    if (name && name !== tag) {
      lines.push(`## ${tag} - ${name} - ${published}`);
    } else {
      lines.push(`## ${tag} - ${published}`);
    }

    if (body) {
      lines.push('');
      lines.push(body);
    }
  }

  await fs.writeFile(argv.file, lines.join('\n') + '\n', 'utf-8');
};
