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
import CodeBlock from '@theme/CodeBlock';
import Heading from '@theme/Heading';
import Layout from '@theme/Layout';
import clsx from 'clsx';
import React from 'react';

import { EXAMPLE_WORKFLOW } from '.';
import indexStyle from './index.module.css';
import styles from './temporal-dsl.module.css';

function Hero() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className={clsx('hero__title', styles.textWhite)}>
          Looking for a Temporal DSL?
        </Heading>
        <p className={clsx('hero__subtitle', styles.textWhite)}>
          Zigflow lets you define Temporal workflows declaratively in YAML.
        </p>
        <p className={styles.heroSupporting}>
          Temporal handles retries, failures and long-running execution. No SDK
          code required.
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

      <div className={indexStyle.credit}>
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

function Validation() {
  return (
    <section className={styles.validation}>
      <div className="container">
        <div className={styles.validationInner}>
          <p>
            Yes. Zigflow is a Temporal DSL. It is the fastest way to get started
            with Temporal without writing SDK code.
          </p>
          <p>
            Write a workflow in YAML. Zigflow validates it and runs it on your
            Temporal cluster. It is production-ready and open source.
          </p>
        </div>
      </div>
    </section>
  );
}

function Comparison() {
  return (
    <section className={styles.comparison}>
      <div className="container">
        <Heading as="h2">Before and after</Heading>
        <div className={styles.comparisonGrid}>
          <div className={styles.comparisonPanel}>
            <h3>Without Zigflow</h3>
            <ul>
              <li>
                Write retry logic, timeout handling and failure recovery in code
              </li>
              <li>
                Register workers, activities and task queues before running
                anything
              </li>
              <li>
                Redeploy your application to change a retry count or add a step
              </li>
              <li>
                Debug failures by reading SDK stack traces and execution state
              </li>
              <li>
                Teach new engineers the Temporal SDK before they can touch
                workflows
              </li>
            </ul>
          </div>
          <div className={styles.comparisonPanel}>
            <h3>With Zigflow</h3>
            <ul>
              <li>Write a YAML file describing the steps</li>
              <li>Validate it with a single command</li>
              <li>Run it on Temporal with no SDK code</li>
              <li>Change behaviour by editing the file</li>
              <li>Share and version workflows like any other configuration</li>
            </ul>
          </div>
        </div>
      </div>
    </section>
  );
}

function Example() {
  return (
    <section className={styles.example}>
      <div className="container">
        <div className={styles.exampleGrid}>
          <div className={styles.exampleText}>
            <Heading as="h2">See how it works</Heading>
            <p>
              Two steps. Fetch a user profile, then send a welcome email. The
              email step retries up to three times if it fails. Temporal handles
              retries automatically based on the retry policy defined in the
              YAML.
            </p>
            <p>
              Your workflow is a single file. Validate it, run it and share it.
            </p>
            <Link
              className="button button--outline button--primary"
              to="/docs/getting-started/quickstart"
            >
              Run your first workflow
            </Link>
          </div>
          <div className={styles.exampleCode}>
            <CodeBlock language="yaml" title="workflow.yaml">
              {EXAMPLE_WORKFLOW}
            </CodeBlock>
          </div>
        </div>
      </div>
    </section>
  );
}

function Why() {
  return (
    <section className={styles.why}>
      <div className="container">
        <Heading as="h2">Why use Zigflow with Temporal</Heading>
        <div className={styles.whyGrid}>
          <div className={styles.whyCard}>
            <h3>Readable</h3>
            <p>
              A workflow is a single file, not logic scattered across activity
              handlers and worker registrations. Anyone can read it and
              understand what it does.
            </p>
          </div>
          <div className={styles.whyCard}>
            <h3>Consistent</h3>
            <p>
              Every workflow follows the same structure. Teams stop arguing
              about how to organise workflow code and start shipping.
            </p>
          </div>
          <div className={styles.whyCard}>
            <h3>Reusable</h3>
            <p>
              Workflow files are plain YAML. Version them, share them, review
              them in pull requests and deploy them independently of your
              application code.
            </p>
          </div>
          <div className={styles.whyCard}>
            <h3>Faster onboarding</h3>
            <p>
              New team members can read and understand a Zigflow workflow
              without knowing the Temporal SDK. The file explains itself.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}

function TemporalPositioning() {
  return (
    <section className={styles.temporal}>
      <div className="container">
        <Heading as="h2">Built on Temporal</Heading>
        <p>
          Every Zigflow workflow runs on{' '}
          <a href="https://temporal.io" target="_blank" rel="noreferrer">
            Temporal
          </a>
          , a battle-tested engine for durable execution. Temporal provides
          automatic retries, crash recovery and full execution history. Zigflow
          gives you a declarative interface to that engine, without starting
          with the SDK.
        </p>
      </div>
    </section>
  );
}

function CTA() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <section className={styles.cta}>
      <div className="container">
        <Heading as="h2">Ready to try it?</Heading>
        <p>
          The quickstart takes five minutes. Run your first workflow locally or
          against your Temporal cluster.
        </p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
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
    </section>
  );
}

export default function TemporalDSL() {
  return (
    <Layout
      title="Temporal DSL"
      description="Define durable Temporal workflows in YAML. Zigflow is a production-ready Temporal DSL with built-in retries and execution."
    >
      <Hero />
      <main>
        <Validation />
        <Comparison />
        <Example />
        <Why />
        <TemporalPositioning />
        <CTA />
      </main>
    </Layout>
  );
}
