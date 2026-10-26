# KoduMedia translations

The web interface loads translation catalogs from `apps/web/src/i18n/locales/*.json`.
Adding a language does not require changing React components or registering the locale in code.

## Add a translation

1. Copy `en-US.json` to a file named with a valid BCP 47 locale, for example `fi-FI.json`.
2. Update `_meta.locale`, `_meta.name`, `_meta.nativeName`, `_meta.direction`, and `_meta.authors`.
3. Translate values in `messages`. Do not modify message keys or placeholders such as `{{count}}`.
4. Keep every key from `et-EE.json`. The catalog parity test rejects missing or extra keys.
5. Run `npm test` and `npm run build` in `apps/web`.

`et-EE.json` defines the typed source keys. `en-US.json` is the first-run and
runtime fallback catalog; a saved browser-local choice overrides that default.

Catalog files are UTF-8 JSON and are intentionally independent from component logic, so community translations can be reviewed as ordinary text changes on GitHub.
