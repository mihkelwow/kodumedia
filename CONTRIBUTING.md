# Contributing to KoduMedia

Thank you for helping improve KoduMedia.

## Before opening an issue

- Search existing issues and the roadmap.
- Use a bug report for reproducible failures and a feature request for new
  behavior.
- Do not post API keys, passwords, private media paths, cookies, tokens or
  personal media samples publicly.
- Follow `SECURITY.md` for vulnerabilities.

## Development setup

KoduMedia's official build environment is Windows 10/11 x64 with:

- Go 1.23+;
- Node.js 22 LTS or another version accepted by `apps/web/package.json`;
- npm 10.8.2+;
- PowerShell 5.1+.

Clone the repository and run from its root:

```powershell
.\test.ps1
```

The script installs locked npm dependencies, validates the UI and translations,
runs frontend tests and build, synchronizes embedded assets, and runs Go tests
and vetting.

## Making changes

1. Create a focused branch from `main`.
2. Keep changes small and explain the user-visible outcome.
3. Add or update tests for changed behavior.
4. Run `gofmt` on modified Go files.
5. Run `.\test.ps1` before opening a pull request.
6. Do not commit generated releases, dependencies, local data or secrets.

### UI and translations

KoduMedia supports Estonian and English. Every user-facing UI string must be
available in both locales:

```text
apps/web/src/i18n/locales/et-EE.json
apps/web/src/i18n/locales/en-US.json
```

Run `npm run check` in `apps/web`; the i18n check must pass without missing or
extra keys. Reuse the existing design system and shared components.

### Backend boundaries

- Keep domain types independent of adapters and transport code.
- Put use-case decisions in `internal/service`.
- Keep FFmpeg command construction inside the transcode adapter.
- Do not return stored secrets from read APIs or logs.
- Preserve Windows path, UNC path and local-network behavior.

## Pull requests

Include:

- a concise summary of the problem and solution;
- test evidence;
- screenshots or a short recording for visible UI changes;
- compatibility or migration notes when applicable;
- documentation and translation updates.

By contributing, you agree that your contribution is licensed under
GPL-3.0-only, the same license as KoduMedia.
