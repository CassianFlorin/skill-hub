# Signing & notarizing SkillHub.app

The [`Release macOS App`](../.github/workflows/release-macos-app.yml) workflow
builds, Developer ID-signs, notarizes, staples, and uploads `SkillHub.app` to
the GitHub release for each `v*` tag.

Until the secrets below are configured, the workflow's gate job **skips**
packaging, so tagged releases stay green. Once the secrets exist, the next tag
(or a manual `workflow_dispatch`) produces a fully notarized app that opens
without Gatekeeper warnings.

## One-time setup

### 1. Apple Developer Program

Enroll at <https://developer.apple.com/programs/> ($99/year). Developer ID
certificates require the **Account Holder** or **Admin** role.

### 2. Developer ID Application certificate

Create a **Developer ID Application** certificate (Xcode → Settings → Accounts →
Manage Certificates → `+`, or developer.apple.com → Certificates). Then export
it from **Keychain Access** as a `.p12` (select the cert *and* its private key →
right-click → Export), choosing an export password.

Find the identity string for `MACOS_SIGN_IDENTITY`:

```bash
security find-identity -v -p codesigning
# -> "Developer ID Application: Your Name (ABCDE12345)"
```

### 3. App Store Connect API key (for notarization)

App Store Connect → **Users and Access → Integrations → App Store Connect API**
→ generate a key with the **Developer** role. Download the `AuthKey_XXXX.p8`
(one-time download). Note the **Key ID** and the team **Issuer ID** shown there.

## Repository secrets

Add these under **Settings → Secrets and variables → Actions** (or with the
`gh` CLI shown below). Base64-encode the binary files first:

```bash
base64 -i DeveloperID.p12   | pbcopy   # -> MACOS_CERTIFICATE
base64 -i AuthKey_XXXX.p8   | pbcopy   # -> NOTARY_KEY
```

| Secret | Value |
| --- | --- |
| `MACOS_CERTIFICATE` | base64 of the Developer ID `.p12` |
| `MACOS_CERTIFICATE_PWD` | the password you set when exporting the `.p12` |
| `MACOS_SIGN_IDENTITY` | e.g. `Developer ID Application: Your Name (ABCDE12345)` |
| `KEYCHAIN_PWD` | any throwaway password for the CI keychain |
| `NOTARY_KEY` | base64 of the App Store Connect `AuthKey_XXXX.p8` |
| `NOTARY_KEY_ID` | the API Key ID (e.g. `ABCDE12345`) |
| `NOTARY_ISSUER_ID` | the Issuer ID UUID |

With the `gh` CLI:

```bash
gh secret set MACOS_CERTIFICATE     < <(base64 -i DeveloperID.p12)
gh secret set MACOS_CERTIFICATE_PWD --body 'your-p12-password'
gh secret set MACOS_SIGN_IDENTITY   --body 'Developer ID Application: Your Name (ABCDE12345)'
gh secret set KEYCHAIN_PWD          --body "$(openssl rand -hex 16)"
gh secret set NOTARY_KEY            < <(base64 -i AuthKey_XXXX.p8)
gh secret set NOTARY_KEY_ID         --body 'ABCDE12345'
gh secret set NOTARY_ISSUER_ID      --body '00000000-0000-0000-0000-000000000000'
```

## Verifying locally

You can do a Developer ID build locally (notarization still needs the API key):

```bash
SKILLHUB_SIGN_IDENTITY="Developer ID Application: Your Name (ABCDE12345)" \
  macos/build.sh
codesign --verify --strict --verbose=2 macos/build/SkillHub.app
```
