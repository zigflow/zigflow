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
    title: 'Simplified Workflow Authoring',
    description: (
      <>
        Focus on the business logic of what you want to achieve rather than
        focusing on how to use Temporal.
      </>
    ),
  },
  {
    title: 'Consistency and Reusability',
    description: (
      <>
        Zigflow enforces consistent patterns across your Temporal estate,
        allowing you to reuse definitions, share components and make your entry
        to the world of Temporal easier.
      </>
    ),
  },
  {
    title: 'Low Code',
    description: (
      <>
        Get all the benefits of{' '}
        <a
          href="https://docs.temporal.io/evaluate/why-temporal"
          target="_blank"
          rel="noreferrer"
        >
          Temporal
        </a>{' '}
        - reliability, speed and consistency - without having to learn the
        nuances of writing code
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
