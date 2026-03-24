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
import Heading from '@theme/Heading';
import clsx from 'clsx';
import PropTypes from 'prop-types';
import React from 'react';

import styles from './styles.module.css';

const FeatureList = [
  {
    title: 'Write steps, not plumbing',
    description: (
      <>
        Define your workflow as a sequence of named tasks in YAML. No SDK, no
        orchestration scaffolding, no boilerplate. Your workflow file is the
        entire implementation.
      </>
    ),
  },
  {
    title: 'Durability out of the box',
    description: (
      <>
        Every workflow runs on{' '}
        <a href="https://temporal.io" target="_blank" rel="noreferrer">
          Temporal
        </a>
        . Retries, crash recovery and execution history are handled for you,
        without any extra configuration.
      </>
    ),
  },
  {
    title: 'Catch errors early',
    description: (
      <>
        Zigflow validates your workflow file before execution starts. Invalid
        constructs and unsupported fields are rejected with clear, actionable
        error messages.
      </>
    ),
  },
];

function Feature({ title, description }) {
  return (
    <div className={clsx('col col--4')}>
      <div className="padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

Feature.propTypes = {
  title: PropTypes.string.isRequired,
  description: PropTypes.string.isRequired,
};

export default function HomepageFeatures() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
