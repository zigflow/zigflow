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
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import { usePluginData } from '@docusaurus/useGlobalData';
import CodeBlock from '@theme/CodeBlock';
import TabItem from '@theme/TabItem';
import Tabs from '@theme/Tabs';
import React from 'react';

import styles from './styles.module.css';

export default function Examples() {
  const { examples } = usePluginData('load-examples');
  const { siteConfig } = useDocusaurusContext();

  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row title">
          <h3>User-Friendly DSL: Temporal Made Simple</h3>
        </div>

        <div className="row title">
          <p>
            A collection of examples of how to use Zigflow&apos;s Temporal DSL
            to create Temporal workflows. These can be found in the{' '}
            <a
              href={siteConfig.customFields.githubURL}
              target="_blank"
              rel="noreferrer"
            >
              GitHub repo
            </a>
            .
          </p>
        </div>

        <Tabs className={styles.exampleTabs} queryString="examples">
          {examples.map((ex) => (
            <TabItem
              key={ex.name.name}
              value={ex.name.name}
              label={ex.workflow.document.title}
            >
              <h3>{ex.workflow.document.title}</h3>
              <p>{ex.workflow.document.summary}</p>

              <CodeBlock language="yaml" title="workflow.yaml">
                {ex.content}
              </CodeBlock>

              <ul>
                <li>
                  <a
                    href={`${siteConfig.customFields.githubURL}/tree/main/examples/${ex.name.name}`}
                    target="_blank"
                    rel="noreferrer"
                  >
                    View example in repo
                  </a>
                </li>
                <li>
                  <a
                    href={`${siteConfig.customFields.githubURL}/tree/main/examples`}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Additional examples
                  </a>
                </li>
              </ul>
            </TabItem>
          ))}
        </Tabs>
      </div>
    </section>
  );
}
