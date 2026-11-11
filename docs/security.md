# Security

- Subscription URLs: DPAPI-encrypted on Windows, mode 0600 on Linux and macOS.
- Fetch cache: URL hash, body hash, ETag, Last-Modified, size. No URL.
- Private, loopback, link-local and metadata addresses: blocked. `--allow-private` opts out.
- Redirect targets: checked again.
- Reports: GitHub Security Advisories. No live URLs, tokens or credentials.
