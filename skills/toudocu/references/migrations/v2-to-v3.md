# Migrate documentation v2 to v3

Toudocu reports `DOCS_MIGRATION_REQUIRED` with `Migration: v2-to-v3` for a v2
configuration. Change configuration only; do not move or rewrite Markdown.

1. Set `documentationVersion: 3`.
2. Rename `project.locale` to `project.defaultLocale`.
3. Create `locales.<default-locale>` with the former canonical documentation
   root and `project.sections` map.
4. Move every `translations.<locale>` profile to `locales.<locale>` unchanged.
5. Remove `translations` and `project.sections`, then run `toudocu check` for
   each locale root that the migration is meant to validate.

```yaml
documentationVersion: 3
project:
  defaultLocale: en
locales:
  en:
    root: docs
    sections: # complete built-in section map
  ru:
    root: docs-ru
    sections: # complete built-in section map
```

All profiles are peers. Existing translation manifests do not configure v3
runtime behavior and do not make one locale the source of truth.
