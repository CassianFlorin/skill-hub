# @cassianflorin/skillhub

Installs the `skillhub` CLI from GitHub Releases.

```bash
npm install -g @cassianflorin/skillhub
npm update -g @cassianflorin/skillhub
skillhub version
```

The package downloads the matching platform archive during `postinstall` and
verifies it with the release `checksums.txt` before exposing the `skillhub`
binary.

To pin or mirror a specific GitHub Release tarball:

```bash
VERSION=1.4.0
npm install -g "https://github.com/CassianFlorin/skill-hub/releases/download/v${VERSION}/cassianflorin-skillhub-${VERSION}.tgz"
```
