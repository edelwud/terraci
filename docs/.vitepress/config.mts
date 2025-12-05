import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "TerraCi",
  description: "Blazing fast Terraform/OpenTofu pipeline generator with dependency resolution",

  base: "/terraci",

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#5f67ee' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'TerraCi' }],
  ],

  locales: {
    root: {
      label: 'English',
      lang: 'en',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/guide/getting-started' },
          { text: 'Configuration', link: '/config/' },
          { text: 'CLI', link: '/cli/' },
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Introduction',
              items: [
                { text: 'What is TerraCi?', link: '/guide/what-is-terraci' },
                { text: 'Getting Started', link: '/guide/getting-started' },
              ]
            },
            {
              text: 'Core Concepts',
              items: [
                { text: 'Project Structure', link: '/guide/project-structure' },
                { text: 'Dependency Resolution', link: '/guide/dependencies' },
                { text: 'Pipeline Generation', link: '/guide/pipeline-generation' },
              ]
            },
            {
              text: 'Advanced',
              items: [
                { text: 'Git Integration', link: '/guide/git-integration' },
                { text: 'OpenTofu Support', link: '/guide/opentofu' },
                { text: 'Submodules', link: '/guide/submodules' },
              ]
            }
          ],
          '/config/': [
            {
              text: 'Configuration',
              items: [
                { text: 'Overview', link: '/config/' },
                { text: 'Structure', link: '/config/structure' },
                { text: 'GitLab CI', link: '/config/gitlab' },
                { text: 'Filters', link: '/config/filters' },
              ]
            }
          ],
          '/cli/': [
            {
              text: 'CLI Reference',
              items: [
                { text: 'Overview', link: '/cli/' },
                { text: 'generate', link: '/cli/generate' },
                { text: 'validate', link: '/cli/validate' },
                { text: 'graph', link: '/cli/graph' },
                { text: 'init', link: '/cli/init' },
              ]
            }
          ]
        },
      }
    },
    ru: {
      label: 'Русский',
      lang: 'ru',
      link: '/ru/',
      themeConfig: {
        nav: [
          { text: 'Руководство', link: '/ru/guide/getting-started' },
          { text: 'Конфигурация', link: '/ru/config/' },
          { text: 'CLI', link: '/ru/cli/' },
        ],
        sidebar: {
          '/ru/guide/': [
            {
              text: 'Введение',
              items: [
                { text: 'Что такое TerraCi?', link: '/ru/guide/what-is-terraci' },
                { text: 'Быстрый старт', link: '/ru/guide/getting-started' },
              ]
            },
            {
              text: 'Основные концепции',
              items: [
                { text: 'Структура проекта', link: '/ru/guide/project-structure' },
                { text: 'Разрешение зависимостей', link: '/ru/guide/dependencies' },
                { text: 'Генерация пайплайнов', link: '/ru/guide/pipeline-generation' },
              ]
            },
            {
              text: 'Продвинутое',
              items: [
                { text: 'Git интеграция', link: '/ru/guide/git-integration' },
                { text: 'Поддержка OpenTofu', link: '/ru/guide/opentofu' },
                { text: 'Сабмодули', link: '/ru/guide/submodules' },
              ]
            }
          ],
          '/ru/config/': [
            {
              text: 'Конфигурация',
              items: [
                { text: 'Обзор', link: '/ru/config/' },
                { text: 'Структура', link: '/ru/config/structure' },
                { text: 'GitLab CI', link: '/ru/config/gitlab' },
                { text: 'Фильтры', link: '/ru/config/filters' },
              ]
            }
          ],
          '/ru/cli/': [
            {
              text: 'CLI справочник',
              items: [
                { text: 'Обзор', link: '/ru/cli/' },
                { text: 'generate', link: '/ru/cli/generate' },
                { text: 'validate', link: '/ru/cli/validate' },
                { text: 'graph', link: '/ru/cli/graph' },
                { text: 'init', link: '/ru/cli/init' },
              ]
            }
          ]
        },
        outline: {
          label: 'На этой странице'
        },
        docFooter: {
          prev: 'Предыдущая',
          next: 'Следующая'
        },
        lastUpdated: {
          text: 'Обновлено'
        },
        editLink: {
          pattern: 'https://github.com/edelwud/terraci/edit/main/docs/:path',
          text: 'Редактировать на GitHub'
        },
        search: {
          provider: 'local',
          options: {
            translations: {
              button: {
                buttonText: 'Поиск',
                buttonAriaLabel: 'Поиск'
              },
              modal: {
                noResultsText: 'Нет результатов для',
                resetButtonTitle: 'Сбросить',
                footer: {
                  selectText: 'выбрать',
                  navigateText: 'перейти'
                }
              }
            }
          }
        }
      }
    }
  },

  themeConfig: {
    logo: '/logo.svg',

    socialLinks: [
      { icon: 'github', link: 'https://github.com/edelwud/terraci' }
    ],

    search: {
      provider: 'local'
    },

    editLink: {
      pattern: 'https://github.com/edelwud/terraci/edit/main/docs/:path',
      text: 'Edit this page on GitHub'
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 TerraCi Contributors'
    }
  }
})
