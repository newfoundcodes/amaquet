import { defineConfig } from "astro/config"
import { unified } from "@astrojs/markdown-remark"
import starlight from "@astrojs/starlight"
import mermaid from "astro-mermaid"
import rewriteDocLinks from "./src/plugins/remark-rewrite-doc-links.mjs"

export default defineConfig({
  site: process.env.DOCS_SITE_URL || "https://amaquet.newfoundcodes.com",
  base: "/",
  vite: {
    build: {
      chunkSizeWarningLimit: 700,
    },
  },
  markdown: {
    processor: unified({ remarkPlugins: [rewriteDocLinks] }),
  },
  integrations: [
    mermaid({
      theme: "dark",
      autoTheme: false,
      enableLog: false,
      mermaidConfig: {
        securityLevel: "strict",
        flowchart: { htmlLabels: false, useMaxWidth: true },
      },
    }),
    starlight({
      title: "Amaquet",
      description: "Standalone in-memory database and Amaquet protocol documentation",
      routeMiddleware: "./src/seo-route-data.ts",
      credits: false,
      customCss: ["./src/styles/shadcn.css"],
      expressiveCode: {
        themes: ["starlight-dark"],
        useStarlightDarkModeSwitch: false,
      },
      head: [
        {
          tag: "link",
          attrs: { rel: "sitemap", type: "application/xml", href: "/sitemap.xml" },
        },
        {
          tag: "link",
          attrs: {
            rel: "alternate",
            type: "text/plain",
            href: "/llms.txt",
            title: "Amaquet documentation for language models",
          },
        },
        {
          tag: "link",
          attrs: {
            rel: "alternate",
            type: "text/plain",
            href: "/llms-full.txt",
            title: "Complete Amaquet documentation for language models",
          },
        },
        {
          tag: "meta",
          attrs: { name: "author", content: "Amaquet contributors" },
        },
        {
          tag: "meta",
          attrs: { name: "twitter:card", content: "summary_large_image" },
        },
        {
          tag: "meta",
          attrs: { name: "color-scheme", content: "dark" },
        },
        {
          tag: "meta",
          attrs: { name: "theme-color", content: "#171717" },
        },
      ],
      editLink: {
        baseUrl: "https://github.com/newfoundcodes/amaquet/edit/main/docs/",
      },
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/newfoundcodes/amaquet",
        },
      ],
      markdown: {
        processedDirs: ["./"],
      },
      components: {
        DraftContentNotice: "./src/components/DraftContentNotice.astro",
        SiteTitle: "./src/components/SiteTitle.astro",
        ThemeProvider: "./src/components/DarkThemeProvider.astro",
        ThemeSelect: "./src/components/Empty.astro",
      },
      sidebar: [
        { label: "Home", slug: "index" },
        {
          label: "Getting started",
          items: [
            { label: "Installation", slug: "getting-started/installation" },
            { label: "Quick start", slug: "getting-started/quickstart" },
            { label: "Configuration", slug: "getting-started/configuration" },
          ],
        },
        {
          label: "Concepts",
          items: [
            { label: "Why Amaquet?", slug: "concepts/why-amaquet" },
            { label: "Architecture", slug: "concepts/architecture" },
            { label: "Engine internals", slug: "concepts/engine-internals" },
            { label: "Memory and concurrency", slug: "concepts/memory-model" },
            { label: "Adaptive compression", slug: "concepts/compression" },
            { label: "Keyspace and TTL", slug: "concepts/keyspace" },
            { label: "Persistence", slug: "concepts/persistence" },
            { label: "Security", slug: "concepts/security" },
          ],
        },
        {
          label: "Amaquet protocol",
          items: [
            { label: "Overview", slug: "protocol/overview" },
            { label: "Binary framing", slug: "protocol/framing" },
            { label: "Commands", slug: "protocol/commands" },
            { label: "Connection URIs", slug: "protocol/uri" },
            { label: "Authentication and identity", slug: "protocol/authentication" },
            { label: "Wire values and snapshots", slug: "protocol/wire-values" },
          ],
        },
        {
          label: "Data types",
          items: [
            { label: "Catalog", slug: "data-types/index" },
            { label: "Construction and operations", slug: "data-types/operations" },
            { label: "Scalars", slug: "data-types/scalar" },
            { label: "Collections", slug: "data-types/collections" },
            { label: "Structured", slug: "data-types/structured" },
            { label: "Queues and messaging", slug: "data-types/queues-and-messaging" },
            { label: "Bit structures", slug: "data-types/bit-structures" },
            { label: "Probabilistic", slug: "data-types/probabilistic" },
            { label: "Time series", slug: "data-types/time-series" },
            { label: "Geospatial", slug: "data-types/geospatial" },
            { label: "Vectors", slug: "data-types/vectors" },
            { label: "Indexes and search", slug: "data-types/indexes-and-search" },
            { label: "Graph, synchronization, network", slug: "data-types/graph-synchronization-and-network" },
          ],
        },
        {
          label: "Operations",
          items: [
            { label: "Deployment", slug: "operations/deployment" },
            { label: "Docker", slug: "operations/docker" },
            { label: "TLS and identity", slug: "operations/tls" },
            { label: "Backups", slug: "operations/backups" },
            { label: "Observability", slug: "operations/observability" },
            { label: "Capacity", slug: "operations/capacity" },
            { label: "Production hardening", slug: "operations/production-hardening" },
            { label: "Troubleshooting", slug: "operations/troubleshooting" },
          ],
        },
        {
          label: "Development",
          items: [
            { label: "Build", slug: "development/build" },
            { label: "Testing and fuzzing", slug: "development/testing-fuzzing" },
            { label: "Developer workflow fuzzing", slug: "development/developer-fuzzing" },
            { label: "Adding data types", slug: "development/adding-types" },
            { label: "Repository layout", slug: "development/repository-layout" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "Configuration", slug: "reference/configuration" },
            { label: "CLI", slug: "reference/cli" },
            { label: "Go client library", slug: "reference/go-client" },
            { label: "Admin API", slug: "reference/admin-api" },
            { label: "Errors", slug: "reference/errors" },
            { label: "Implementation coverage", slug: "reference/implementation-coverage" },
            { label: "Glossary", slug: "reference/glossary" },
          ],
        },
        {
          label: "AI resources",
          items: [
            {
              label: "llms.txt",
              link: "/llms.txt",
              attrs: { target: "_blank", rel: "noopener noreferrer" },
            },
            {
              label: "llms-full.txt",
              link: "/llms-full.txt",
              attrs: { target: "_blank", rel: "noopener noreferrer" },
            },
          ],
        },
      ],
    }),
  ],
})
