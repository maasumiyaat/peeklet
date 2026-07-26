# Peeklet — backend (chunk a: infra + config + store)

An OTP-gated viewer for private S3 photo/video folders. You generate a short
URL + OTP; a recipient enters the OTP and browses that folder (and its
subfolders, never its parent) through a fast CloudFront CDN. Links and OTPs
expire automatically (default 15 days).

## Architecture

- **Go Lambda** (Function URL) — the API. Holds no static AWS keys; uses its
  IAM role. Validates OTPs, enforces the folder boundary, signs media URLs.
- **DynamoDB** — one row per share, with native TTL auto-expiry. The app also
  checks `expiresAt` on every read, so TTL lag never widens access.
- **S3** (private) + **CloudFront** — media origin + CDN, cached 1 day.
- **Signed URLs** (not cookies) gate the media; the app session is a **Bearer
  JWT** (not a cookie). This keeps the frontend hostable on any free static
  host with zero cross-site-cookie friction, and is strictly per-object: a
  viewer can only load URLs the app signed for keys under their granted prefix.

## Prerequisites

- Go 1.22+, AWS SAM CLI, AWS credentials configured, `openssl`.
- Docker only if you want `sam local`.

## Deploy

```bash
# 1. Fetch dependencies
make tidy

# 2. Generate the CloudFront signing keypair (copy both PEMs)
make keys

# 3. Hash your owner/admin password
make hashpw PW='your-strong-password'

# 4. Build + first deploy (interactive; pass the values from steps 2-3)
make build
sam deploy --guided \
  --parameter-overrides \
    "MediaBucketName=your-private-media-bucket" \
    "AllowedOrigin=https://your-app.pages.dev" \
    "OwnerPasswordHash=<from step 3>" \
    "JWTSecret=$(openssl rand -hex 32)" \
    "CFPublicKeyPem=<PEM from step 2>" \
    "CFPrivateKeyPem=<PEM from step 2>"
```

Subsequent deploys: just `make deploy` (or `sam deploy`), which reuses the
saved `samconfig.toml`.

### Verify

The stack outputs `FunctionUrl` and `CdnDomain`. Smoke-test:

```bash
curl "$FUNCTION_URL/health"      # -> {"status":"ok"}
```

Other routes currently return `501` — they're built in chunk (b).

## Config (all tunable, then redeploy)

Set these as stack parameters (defaults shown). Durations use Go syntax
(`360h`, `15m`, `2h`).

| Parameter | Default | Meaning |
|-----------|---------|---------|
| `LinkTTL` | `360h` (15d) | Default expiry stamped on **new** shares only |
| `SessionTTL` | `2h` | Viewer browsing-session length |
| `MediaURLTTL` | `2h` | Signed-URL lifetime (must be ≤ `SessionTTL`) |
| `CDNCacheTTL` | `86400` | CloudFront cache seconds ("1 day max") |
| `OTPLength` | `8` | OTP length |
| `OTPAlphabet` | `A–Z2–9` (no 0/O/1/I/L) | OTP character set |
| `OTPMaxAttempts` | `5` | Failed tries before lockout |
| `OTPLockout` | `15m` | Lockout duration |
| `AllowedExt` | jpg,jpeg,png,webp,gif,mp4,webm,mov | Viewable extensions |
| `ListPageSize` | `100` | Folder pagination size |
| `AllowedOrigin` | `*` | Frontend origin for CORS |

Changing `LinkTTL` affects only shares created afterward; existing shares keep
their original expiry. (If you'd rather have config retroactively re-expire
live shares, say so — it's a small change to the read path.)

## Project layout

Module path is `peeklet` (bare). Clone into a folder named `peeklet/`.

```
peeklet/
├── template.yaml              SAM: DynamoDB, Lambda+FunctionURL, S3, CloudFront, keys, IAM
├── Makefile                   build-ApiFunction (SAM), tidy/deploy/keys/hashpw helpers
├── go.mod                     module peeklet
├── samconfig.toml.example     copy to samconfig.toml (git-ignored) and edit
├── README.md
├── .gitignore
├── events/
│   └── health.json            sample event for `sam local invoke`
├── scripts/
│   └── gen-keys.sh            RSA keypair generator for CloudFront signed URLs
├── cmd/
│   ├── lambda/main.go         tiny entrypoint: load config -> api.NewServer -> Start
│   └── hashpw/main.go         bcrypt hash helper for the owner password
└── internal/
    ├── config/config.go       env -> Config, validation, media-ext check      [chunk a]
    ├── store/store.go         DynamoDB share store (CRUD + lockout counters)   [chunk a]
    ├── api/server.go          Server struct, AWS wiring, router, JSON helpers  [chunk a/b]
    ├── api/helpers.go         slug gen, prefix normalize, media-type           [chunk b]
    ├── api/admin.go           /admin/* handlers                                [chunk b]
    ├── api/viewer.go          /api/{slug}/verify + /list handlers              [chunk b]
    ├── otp/otp.go             OTP generate + PBKDF2 hash + verify              [chunk b]
    ├── session/session.go     JWT issue/verify                                 [chunk b]
    ├── cfsign/cfsign.go       CloudFront URL signing (key from Secrets Mgr)    [chunk b]
    └── s3list/s3list.go       ListObjectsV2 + normalized prefix guard          [chunk b]
```

## Status

- **Chunk (a) — done:** infra, config, data store, deployable skeleton.
- **Chunk (b) — done:** OTP hashing + verify with lockout, JWT sessions,
  CloudFront URL signing, S3 listing with the prefix guard, and all handlers.
- **Chunk (c) — next:** the SPA + Bigger Picture viewer (separate frontend).

## API reference

All bodies are JSON. Admin routes need `Authorization: Bearer <admin token>`;
the two `/api/{slug}/...` routes are the viewer flow.

```
GET    /health                      -> {"status":"ok"}

POST   /admin/login                 {password}            -> {token}
POST   /admin/shares                {prefix,label?,ttlOverride?}
                                      -> {slug,url,otp,expiresAt}   (otp shown once)
GET    /admin/shares                -> [{slug,prefix,label,createdAt,expiresAt}]
DELETE /admin/shares/{slug}         -> {deleted}

POST   /api/{slug}/verify           {otp}                 -> {token,root,expiresAt}
GET    /api/{slug}/list?path=&token= (Bearer viewer token)
        -> {path,parent,folders:[{name,path}],files:[{name,url,type}],nextToken}
```

### Quick manual test (after deploy)

```bash
BASE="$FUNCTION_URL"           # from stack outputs

# 1. admin login
TOKEN=$(curl -s "$BASE/admin/login" -d '{"password":"YOUR_PW"}' | jq -r .token)

# 2. create a share for an existing S3 folder
curl -s "$BASE/admin/shares" -H "authorization: Bearer $TOKEN" \
     -d '{"prefix":"clients/wedding-jan/","label":"Jan wedding"}' | jq
# -> note the slug + otp

# 3. as a viewer: verify the OTP
VTOK=$(curl -s "$BASE/api/<slug>/verify" -d '{"otp":"<OTP>"}' | jq -r .token)

# 4. list the root folder (signed media URLs come back in .files[].url)
curl -s "$BASE/api/<slug>/list" -H "authorization: Bearer $VTOK" | jq
```

Notes: `prefix` must be a real, `/`-delimited S3 prefix (a trailing slash is
added if missing); a viewer can descend into subfolders returned in `folders`
but `Resolve` rejects anything outside the granted prefix; each `files[].url`
is a CloudFront signed URL valid for `MediaURLTTL`.