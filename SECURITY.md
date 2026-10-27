# Security policy

## Supported version

Security fixes are applied to the latest KoduMedia release. Older development
packages are not supported.

## Important deployment boundary

KoduMedia 0.8.58 is intended for a trusted home network. It does not yet have
user accounts, TLS termination or secure internet-facing remote access.

- Do not expose port `8096` to the public internet.
- Allow Windows Firewall access on Private networks only.
- Use a separate, trusted reverse proxy and authentication layer only if you
  understand and accept the additional security responsibility.
- Treat every configured media path and stored credential as private.

The detailed current threat model is documented in
[`docs/SECURITY.md`](docs/SECURITY.md).

## Reporting a vulnerability

Do not disclose an exploitable vulnerability, credential or private media path
in a public issue.

Use GitHub's **Report a vulnerability** option in the repository's Security tab
when it is available. If private reporting is unavailable, open a minimal issue
that contains no exploit details or secrets and ask the maintainer to establish
a private contact channel.

Include privately:

- the affected version;
- reproduction steps or a proof of concept;
- the expected and observed impact;
- relevant logs with credentials and personal paths removed;
- a suggested fix, if available.

You should receive an acknowledgement within seven days. Public disclosure
should wait until a fix or mitigation is available.

## Secrets

KoduMedia does not contain shared TMDb, AcoustID or OpenSubtitles credentials.
Users provide their own optional credentials at runtime. `.env`, `data/`, key
files and common credential exports are excluded by `.gitignore`.

If a real secret is ever committed, removing it from the latest file is not
enough. Revoke or rotate it immediately and remove it from Git history before
publishing the repository.
