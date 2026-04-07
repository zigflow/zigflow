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
import CodeBlock from '@theme/CodeBlock';
import Heading from '@theme/Heading';
import Layout from '@theme/Layout';
import clsx from 'clsx';
import React from 'react';

import styles from './index.module.css';

export const EXAMPLE_WORKFLOW = `document:
  dsl: 1.0.0
  taskQueue: acme
  workflowType: onboard-user
  version: 1.0.0
do:
  - fetchProfile:
      call: http
      with:
        method: get
        endpoint: \${ "https://api.acme.com/users/" + ($input.userId | tostring) }
      output:
        as:
          profile: \${ . }
  - sendWelcome:
      call: http
      metadata:
        activityOptions:
          retryPolicy:
            maximumAttempts: 3
      with:
        method: post
        endpoint: https://api.acme.com/emails
        body:
          to: \${ .profile.email }
          template: welcome`;

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className={clsx('hero__title', styles.textWhite)}>
          Define durable workflows in YAML
        </Heading>
        <p className={clsx('hero__subtitle', styles.textWhite)}>
          Run workflows with built-in retries, failure handling and long-running
          execution, powered by Temporal.
        </p>
        <div className={styles.buttons}>
          <Link
            className="button button--primary button--lg"
            to="/docs/getting-started/quickstart"
          >
            Start in 5 minutes
          </Link>
          <a
            className={clsx('button button--lg', styles.buttonSecondary)}
            href={siteConfig.customFields.githubURL}
            target="_blank"
            rel="noopener noreferrer"
          >
            ⭐ Star on GitHub
          </a>
        </div>
      </div>

      <div className={styles.credit}>
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

function WorkflowExample() {
  return (
    <section className={styles.workflowExample}>
      <div className="container">
        <div className={styles.workflowExampleGrid}>
          <div className={styles.workflowExampleText}>
            <Heading as="h2">See how it works</Heading>
            <p>
              Two steps. Fetch a user profile, then send a welcome email. If
              either step fails, Temporal retries it automatically using the
              retry policy defined in the YAML.
            </p>
            <p>
              Your workflow is a single file. Validate it, run it and share it.
            </p>
            <Link
              className="button button--outline button--primary"
              to="/docs/getting-started/quickstart"
            >
              Try your first workflow
            </Link>
          </div>
          <div className={styles.workflowExampleCode}>
            <CodeBlock language="yaml" title="workflow.yaml">
              {EXAMPLE_WORKFLOW}
            </CodeBlock>
          </div>
        </div>
      </div>
    </section>
  );
}

export default function Home() {
  const { siteConfig } = useDocusaurusContext();

  return (
    <Layout
      title={siteConfig.tagline}
      description="Zigflow lets you define durable, production-ready workflows in YAML, powered by Temporal."
    >
      <HomepageHeader />
      <main>
        <HomepageFeatures />
        <WorkflowExample />

        <section className={styles.info_box}>
          <div className="container">
            <div className="row">
              <p>
                Every Zigflow workflow runs on Temporal, a battle-tested engine
                for durable execution. You get automatic retries, crash recovery
                and full execution history without writing SDK code.
              </p>
            </div>
          </div>
        </section>

        <Examples />
      </main>
    </Layout>
  );
}
