# Documentation source

This directory contains the Astro and Starlight documentation site for Amaquet.

```bash
cd docs
npm ci
npm run build
npm run dev
```

The production site is written to `docs/dist`. From the repository root, the
same workflows are available as `make docs-build` and `make docs-serve`. The
development site is served at `http://localhost:4321/`, and the production
site is built for the root of a domain or subdomain. Set `DOCS_SITE_URL` to the
public origin during deployment so canonical URLs, structured data, the
sitemap, and `robots.txt` use the deployment hostname:

```bash
DOCS_SITE_URL=https://docs.newfoundcodes.com npm run build
```

The build generates `public/llms.txt`, `public/llms-full.txt`,
`public/sitemap.xml`, and `public/robots.txt` from the Markdown source before
Astro renders the site. Run `npm run generate:ai` after editing documentation
when you want to refresh these checked-in files without performing a full
build. `npm run check:ai` verifies that the generated files are current.

Run the dependency-light repository checker with:

```bash
python3 docs/scripts/check_docs.py
```
