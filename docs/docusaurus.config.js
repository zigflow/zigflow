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
// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config
import { themes as prismThemes } from 'prism-react-renderer';

import loadExamplesPlugin from './plugins/load-examples.mjs';

const organizationName = 'zigflow';
const projectName = 'zigflow';
const githubDomain = `${organizationName}/${projectName}`;
const githubURL = `https://github.com/${githubDomain}`;

// vscode-languageserver-protocol (pulled in via mermaid → @mermaid-js/parser → langium)
// CJS-requires vscode-languageserver-types, landing on the `default` export
// condition which points to the UMD build. The UMD factory pattern triggers a
// webpack "Critical dependency" warning. Alias to the ESM build to avoid it.
// lib/esm/main.js is not a named export in the package so we can't use
// require.resolve with the subpath directly; derive the path from the default
// resolution instead.
const vscodeLangserverTypesEsm = require
  .resolve('vscode-languageserver-types')
  .replace(/[\\/]lib[\\/]umd[\\/]main\.js$/, '/lib/esm/main.js');

/** @type {import('@docusaurus/types').PluginConfig[]} */
const plugins = [
  loadExamplesPlugin,
  () => ({
    name: 'vscode-languageserver-types-esm-alias',
    configureWebpack() {
      return {
        resolve: {
          alias: {
            'vscode-languageserver-types': vscodeLangserverTypesEsm,
          },
        },
      };
    },
  }),
];

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
  title: 'Zigflow',
  tagline: 'Define durable workflows in YAML',
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
        blog: {
          showLastUpdateAuthor: true,
          showLastUpdateTime: true,
          showReadingTime: true,
          routeBasePath: 'articles',
          path: './articles',
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
      algolia: {
        appId: 'MQUMW9FAV4',
        apiKey: '9f4d9b228ac7b327c04a19230fcd0b25',
        indexName: 'Zigflow docs',
        contextualSearch: true,
      },
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
            'Zigflow lets you define durable, production-ready workflows in YAML, powered by Temporal. Write the steps. Zigflow handles retries, failures and state.',
        },
        {
          name: 'keywords',
          content:
            'Temporal, YAML workflows, durable execution, workflow engine, orchestration, serverless workflow, Zigflow',
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
            type: 'docSidebar',
            sidebarId: 'guidesSidebar',
            position: 'left',
            label: 'Guides',
          },
          {
            to: 'articles',
            label: 'Articles',
            position: 'left',
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
