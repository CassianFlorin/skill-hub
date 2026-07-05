#!/usr/bin/env bash
#
# Build SkillHub.app: a native macOS front-end that bundles the skillhub CLI.
#
# Produces a universal (arm64 + x86_64) app at macos/build/SkillHub.app.
#
# Usage:
#   ./build.sh            # universal release build + assemble .app
#   ./build.sh --arch     # build only for the host architecture (faster)
#
set -euo pipefail

MACOS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${MACOS_DIR}/.." && pwd)"
BUILD_DIR="${MACOS_DIR}/build"
APP="${BUILD_DIR}/SkillHub.app"

HOST_ONLY=0
if [[ "${1:-}" == "--arch" ]]; then
    HOST_ONLY=1
fi

echo "==> Cleaning ${BUILD_DIR}"
rm -rf "${APP}"
mkdir -p "${BUILD_DIR}"

# 1. Build the Go skillhub binary (universal unless --arch).
echo "==> Building skillhub CLI binary"
VERSION="$(cd "${REPO_ROOT}" && git describe --tags --always 2>/dev/null || echo dev)"
LDFLAGS="-s -w -X github.com/CassianFlorin/skill-hub/internal/cli.Version=${VERSION}"
if [[ "${HOST_ONLY}" == "1" ]]; then
    (cd "${REPO_ROOT}" && go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/skillhub" ./cmd/skillhub)
else
    (cd "${REPO_ROOT}" && GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/skillhub-arm64" ./cmd/skillhub)
    (cd "${REPO_ROOT}" && GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/skillhub-amd64" ./cmd/skillhub)
    lipo -create -output "${BUILD_DIR}/skillhub" \
        "${BUILD_DIR}/skillhub-arm64" "${BUILD_DIR}/skillhub-amd64"
    rm -f "${BUILD_DIR}/skillhub-arm64" "${BUILD_DIR}/skillhub-amd64"
fi

# 2. Build the SwiftUI app.
echo "==> Building SwiftUI front-end"
if [[ "${HOST_ONLY}" == "1" ]]; then
    (cd "${MACOS_DIR}" && swift build -c release)
    SWIFT_BIN="$(cd "${MACOS_DIR}" && swift build -c release --show-bin-path)/SkillHub"
else
    (cd "${MACOS_DIR}" && swift build -c release --arch arm64 --arch x86_64)
    SWIFT_BIN="$(cd "${MACOS_DIR}" && swift build -c release --arch arm64 --arch x86_64 --show-bin-path)/SkillHub"
fi

# 3. Assemble the .app bundle.
echo "==> Assembling ${APP}"
mkdir -p "${APP}/Contents/MacOS" "${APP}/Contents/Resources"
cp "${SWIFT_BIN}" "${APP}/Contents/MacOS/SkillHub"
cp "${BUILD_DIR}/skillhub" "${APP}/Contents/Resources/skillhub"
chmod +x "${APP}/Contents/Resources/skillhub"
cp "${MACOS_DIR}/Info.plist" "${APP}/Contents/Info.plist"

# Optional app icon, if macos/AppIcon.icns is provided.
if [[ -f "${MACOS_DIR}/AppIcon.icns" ]]; then
    cp "${MACOS_DIR}/AppIcon.icns" "${APP}/Contents/Resources/AppIcon.icns"
fi

# 4. Codesign. Defaults to ad-hoc ("-") for local use; CI sets
#    SKILLHUB_SIGN_IDENTITY to a "Developer ID Application: …" identity for a
#    notarizable build (hardened runtime + secure timestamp, signed inner-out).
SIGN_IDENTITY="${SKILLHUB_SIGN_IDENTITY:--}"
if [[ "${SIGN_IDENTITY}" == "-" ]]; then
    echo "==> Ad-hoc codesigning"
    codesign --force --deep --sign - "${APP}" >/dev/null 2>&1 || \
        echo "    (codesign skipped/failed — app still runs locally)"
else
    echo "==> Codesigning with: ${SIGN_IDENTITY}"
    # Sign the embedded CLI first, then the app bundle (inner-out).
    codesign --force --options runtime --timestamp \
        --sign "${SIGN_IDENTITY}" "${APP}/Contents/Resources/skillhub"
    codesign --force --options runtime --timestamp \
        --sign "${SIGN_IDENTITY}" "${APP}"
    codesign --verify --strict --verbose=2 "${APP}"
fi

echo ""
echo "Built ${APP}"
echo "Run it with:  open \"${APP}\""
