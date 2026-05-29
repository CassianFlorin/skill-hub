# @cassianflorin/skillhub

Installs the `skillhub` CLI from GitHub Releases.

```bash
npm install -g @cassianflorin/skillhub
skillhub version
```

The package downloads the matching platform archive during `postinstall` and
verifies it with the release `checksums.txt` before exposing the `skillhub`
binary.
