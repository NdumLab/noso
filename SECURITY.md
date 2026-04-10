# Security Policy

## Supported versions

Only the latest release is supported with security fixes.

## Reporting a vulnerability

Please do **not** open a public GitHub issue for security vulnerabilities.

Email the maintainers directly with:

- A description of the vulnerability
- Steps to reproduce it
- The version or commit where you found it
- Any suggested fix, if you have one

You will receive a response within 5 business days.  If the issue is confirmed,
a fix will be prepared and released before public disclosure.

## Security design notes

- The audit log is written with `0600` permissions and its directory with `0700`.
  Only the owning user can read session history.
- noso never auto-executes commands.  All output is read-only guidance unless
  the user explicitly runs a suggested command themselves.
- Evidence collection uses direct `exec.Command` calls — not `bash -c` with
  user-supplied strings — to avoid shell injection.
- Input size is capped at 64 KB for queries and 512 KB for interpreted output.
