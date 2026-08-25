# Publishing the static portal

`toudocu build` creates read-only output that requires no Toudocu backend,
Node.js, database, or CDN after generation.

```bash
toudocu build ./docs --output ./site --clean
```

Upload the entire `site/` directory to ordinary HTTP(S) static hosting such as
nginx, GitHub Pages, S3-compatible storage, or an internal web server. Do not
upload HTML alone: `assets/`, `data/`, `report.json`, and local project assets
are all part of the portal.

Generated portals are not stored in Git. When the `deploy site to pages`
workflow is started manually, GitHub Actions creates a temporary Pages artifact:
it first builds the Russian portal from canonical `docs/`, then English from
`docs-en/`, then adds the language landing page. The parent `project-docs/`
output is always cleaned and built first; otherwise, cleaning it later would
delete the already-built `project-docs/en/`. Demo documentation is not included
in the Pages artifact.

The public artifact structure is:

- `/` — locale gateway;
- `/ru/` and `/en/` — Russian and English landing pages;
- `/project-docs/` — the portal built from canonical `docs/`;
- `/project-docs/en/` — the portal built from `docs-en/`.

The locale gateway first honors the `toudocu-landing-locale` key saved only by
an explicit language switch, then checks the browser's primary locale. `ru` and
`ru-*` open `/ru/`; every other or unknown value opens `/en/`. Language detection
runs only at `/`: a direct locale URL always keeps the language selected by its
URL, and with JavaScript disabled the root displays both links.

The portal uses relative document links and relative asset/data bases supplied
by Go, so the same output can be hosted at the root or below a nested path such
as `/docs/` or `/projects/my-project/` without mandatory `baseURL`
configuration.

Static output is a potentially public artifact. The generator does not include
absolute filesystem paths, server configuration, editor metadata, credentials,
or data outside the permitted documentation model.

For local viewing, use `toudocu serve` and the
[local workflow](local-workflow.md).
Opening `index.html` directly through `file://` is not supported. Use
`toudocu serve` locally or publish the directory over HTTP(S).

## Verifying a deployment

1. Serve the entire output over HTTP.
2. Open the home page and a nested document page.
3. Verify that `portal.css`, `portal.js`, and `data/search-index.json` load.
4. Verify search, theme, and the Mermaid fallback.
5. Repeat the smoke test under a nested URL path.

To verify the landing page and both portals together locally, run `make docs`,
then `make landing-serve`. The first command builds `docs/` into the ignored
`build/project-docs/` and `docs-en/` into `build/project-docs/en/` in the safe
order. The server preserves the same public `/ru/`, `/en/`, `/project-docs/`,
and `/project-docs/en/` routes as the Pages artifact.

## Related documents

- [UC-DOCS-01: Build a static HTTP portal](../use-cases/build-portal.md)
- [Go/frontend boundary](../architecture/frontend-runtime-boundary.md)
- [CLI contract](../contracts/cli.md)
