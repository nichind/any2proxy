# security

subscription urls are secrets. windows uses dpapi. linux and macos use a 0600 file.

the fetch cache stores a url hash, body hash, etag, last-modified and size. not the url.

private, loopback, link-local and metadata addresses are blocked. `--allow-private` turns
that off. redirect targets are checked again.
