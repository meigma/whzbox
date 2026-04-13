import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'whzbox',
  tagline: 'CLI documentation',

  future: {
    v4: true,
  },

  url: 'https://example.com',
  baseUrl: '/',

  onBrokenLinks: 'throw',
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          path: '.',
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
          include: [
            'index.md',
            'tutorials/**/*.md',
            'how-to/**/*.md',
            'reference/**/*.md',
            'explanation/**/*.md',
          ],
          editUrl: 'https://github.com/meigma/whzbox/edit/main/docs/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'whzbox',
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docs',
          label: 'Documentation',
          position: 'left',
        },
        {
          href: 'https://github.com/meigma/whzbox',
          label: 'GitHub',
          position: 'right',
          className: 'navbar__item--github',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Diataxis',
          items: [
            {label: 'Tutorials', to: '/tutorials'},
            {label: 'How-to Guides', to: '/how-to'},
            {label: 'Reference', to: '/reference'},
            {label: 'Explanation', to: '/explanation'},
          ],
        },
        {
          title: 'Project',
          items: [
            {label: 'Repository', href: 'https://github.com/meigma/whzbox'},
            {label: 'README', href: 'https://github.com/meigma/whzbox/blob/main/README.md'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} whzbox`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'go', 'json', 'yaml'],
    },
    docs: {
      sidebar: {
        hideable: true,
      },
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
