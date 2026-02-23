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
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import React, { useEffect, useState } from 'react';

function formatStars(count) {
  let o = '';
  if (count >= 1000) {
    o += (count / 1000).toFixed(1).replace(/\.0$/, '') + 'k';
  } else {
    o += String(count);
  }

  return o;
}

export default function GitHubStars() {
  const { siteConfig } = useDocusaurusContext();
  const { githubDomain, githubURL } = siteConfig.customFields;
  const apiURL = `https://api.github.com/repos/${githubDomain}`;

  const [stars, setStars] = useState(null);

  useEffect(() => {
    fetch(apiURL)
      .then((res) => {
        if (!res.ok) throw new Error('fetch failed');
        return res.json();
      })
      .then((data) => setStars(data.stargazers_count))
      .catch(() => {
        // leave stars as null on error — falls back to icon-only display
      });
  }, [apiURL]);

  return (
    <a
      href={githubURL}
      target="_blank"
      rel="noreferrer"
      className="navbar__item navbar__link"
      aria-label={`GitHub repository${stars !== null ? ` — ${formatStars(stars)} stars` : ''}`}
      style={{ display: 'flex', alignItems: 'center', gap: '0.4em' }}
    >
      GitHub • ⭐ {stars !== null ? formatStars(stars) : ''}
    </a>
  );
}
