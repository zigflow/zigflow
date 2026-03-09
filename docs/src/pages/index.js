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
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Examples from '@site/src/components/Examples';
import HomepageFeatures from '@site/src/components/HomepageFeatures';
import Heading from '@theme/Heading';
import Layout from '@theme/Layout';
import clsx from 'clsx';
import React from 'react';

import styles from './index.module.css';

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className={clsx('hero__title', styles.textWhite)}>
          {siteConfig.title}
        </Heading>
        <p className={clsx('hero__subtitle', styles.textWhite)}>
          {siteConfig.tagline}
        </p>
        <div
          className={styles.buttons}
          style={{ display: 'flex', gap: '1rem' }}
        >
          <Link className="button button--primary button--lg" to="/docs/intro">
            Getting Started - 5min ⏱️
          </Link>
        </div>
      </div>

      <div className={clsx(styles.credit, 'hidden-xs hidden-sm')}>
        <a
          href="https://unsplash.com/@hamburgmeinefreundin?utm_source=unsplash&utm_medium=referral&utm_content=creditCopyText"
          target="_blank"
          rel="noopener noreferrer"
        >
          Photo by Wolfgang Weiser
        </a>
      </div>
    </header>
  );
}

export default function Home() {
  const { siteConfig } = useDocusaurusContext();

  return (
    <Layout title={siteConfig.tagline} description={siteConfig.tagline}>
      <HomepageHeader />
      <main>
        <HomepageFeatures />

        <section className={styles.info_box}>
          <div className="container">
            <div className="row">
              <p>
                Zigflow is a Temporal DSL — a domain-specific language for
                defining and running Temporal workflows declaratively.
              </p>
            </div>
          </div>
        </section>
        <Examples />
      </main>
    </Layout>
  );
}
