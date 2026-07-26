#!/usr/bin/env bash
# Generates the RSA keypair CloudFront uses for signed URLs, and prints both
# PEMs ready to paste as stack parameters (CFPublicKeyPem / CFPrivateKeyPem).
#
# CloudFront requires SHA-256 / RSA-2048 keys in PEM (PKCS#1/SPKI) form.
set -euo pipefail

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

openssl genrsa -out "$tmp/private.pem" 2048 2>/dev/null
openssl rsa -pubout -in "$tmp/private.pem" -out "$tmp/public.pem" 2>/dev/null

echo "===== CFPublicKeyPem (safe to share) ====="
cat "$tmp/public.pem"
echo
echo "===== CFPrivateKeyPem (SECRET - handle carefully) ====="
cat "$tmp/private.pem"
echo
echo "Tip: pass these via a params file rather than the shell to avoid history/quoting issues:"
echo '  sam deploy --parameter-overrides "CFPublicKeyPem=$(cat public.pem)" ...'