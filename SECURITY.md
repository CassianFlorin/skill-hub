# Security Policy

## Supported Versions

Only the latest release line (`v1.3.x`) receives security fixes.

## Reporting a Vulnerability

Please report vulnerabilities privately through [GitHub Security Advisories](https://github.com/CassianFlorin/skill-hub/security/advisories/new). Do not open a public issue for security problems.

Include:

- A description of the issue and its impact.
- Steps to reproduce or a proof of concept.
- The `skillhub version` output and platform you tested on.

You can expect an acknowledgement within 7 days. Once a fix is released, the advisory will be published with credit unless you prefer to stay anonymous.

## Scope Notes

Skills are packages of instructions and scripts executed by AI agents. skill-hub verifies checksums for its own release artifacts and records install provenance in `skillhub.lock`, but it does not sandbox Skill content. Malicious Skill content in a third-party registry is a registry curation issue — report catalog concerns to [skill-hub-registry](https://github.com/CassianFlorin/skill-hub-registry/issues); report anything that lets a Skill or registry escape skill-hub's documented behavior (path traversal on install/deploy, checksum bypass, arbitrary write outside managed directories) here as a vulnerability.
