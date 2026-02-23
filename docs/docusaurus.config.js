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
// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config
import { themes as prismThemes } from 'prism-react-renderer';

import loadExamplesPlugin from './plugins/load-examples.mjs';

const organizationName = 'mrsimonemms';
const projectName = 'zigflow';
const githubDomain = `${organizationName}/${projectName}`;
const githubURL = `https://github.com/${githubDomain}`;

/** @type {import('@docusaurus/types').PluginConfig[]} */
const plugins = [loadExamplesPlugin];

if (process.env.GA_TRACKING_ID) {
  plugins.push([
    '@docusaurus/plugin-google-gtag',
    {
      trackingID: process.env.GA_TRACKING_ID,
      anonymizeIP: true,
    },
  ]);
}

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Zigflow: A Temporal DSL',
  tagline:
    'A Temporal DSL for turning declarative YAML into production-ready workflows',
  favicon: 'img/favicon.ico',

  customFields: {
    githubDomain,
    githubURL,
  },

  // Future flags, see https://docusaurus.io/docs/api/docusaurus-config#future
  future: {
    v4: true, // Improve compatibility with the upcoming Docusaurus v4
  },

  // Set the production url of your site here
  url: 'https://zigflow.dev',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: process.env.ZIGFLOW_BASE_URL ?? '/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName, // Usually your GitHub org/user name.
  projectName, // Usually your repo name.

  onBrokenLinks: 'throw',

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  plugins,

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: './sidebars.js',
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl: `${githubURL}/tree/main/docs/`,
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  markdown: {
    mermaid: true,
  },

  themes: ['@docusaurus/theme-mermaid'],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      announcementBar: {
        content: `⭐ <a href="${githubURL}" target="_blank">Star Zigflow on GitHub</a> to help more developers discover it.`,
        backgroundColor: '#E1062C',
        textColor: '#FAFAFA',
      },
      image: 'img/social.png',
      colorMode: {
        respectPrefersColorScheme: true,
      },
      metadata: [
        {
          name: 'description',
          content:
            'Zigflow — a Temporal DSL that turns declarative YAML into production-ready workflows on Temporal. Learn how Temporal DSL simplifies workflow definitions.',
        },
        {
          name: 'keywords',
          content:
            'Temporal, DSL, YAML, workflows, workflow management system, serverless workflow, durable execution',
        },
      ],
      navbar: {
        logo: {
          alt: 'Zigflow',
          src: 'img/logo.png',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docsSidebar',
            position: 'left',
            label: 'Docs',
          },
          {
            type: 'docSidebar',
            sidebarId: 'dslSidebar',
            position: 'left',
            label: 'DSL',
          },
          {
            type: 'docSidebar',
            sidebarId: 'cliSidebar',
            position: 'left',
            label: 'CLI',
          },
          {
            type: 'docSidebar',
            sidebarId: 'deploymentSidebar',
            position: 'left',
            label: 'Deploying',
          },
          {
            type: 'custom-githubStars',
            position: 'right',
          },
          {
            label: '❤️ Sponsor',
            position: 'right',
            href: 'https://buymeacoffee.com/mrsimonemms',
          },
          {
            label: 'Built on Temporal',
            position: 'right',
            href: 'https://temporal.io',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Community',
            items: [
              {
                label: 'Slack',
                href: 'https://slack.zigflow.dev',
              },
              {
                label: 'GitHub Discussions',
                href: `${githubURL}/discussions`,
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'Temporal',
                href: 'https://temporal.io',
              },
              {
                label: 'Serverless Workflow',
                href: 'https://serverlessworkflow.io',
              },
              {
                label: 'GitHub',
                href: githubURL,
              },
            ],
          },
        ],
        copyright: `Licenced under <a href="${githubURL}/blob/main/LICENSE" target="_blank">Apache-2.0</a>
        <br />
        Copyright &copy; 2025 - ${new Date().getFullYear()} <a href="${githubURL}/graphs/contributors" target="_blank">Zigflow authors</a>.
        Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ['ruby'],
      },
    }),
};

export default config;
