# Peeklet — web (Cinematic SPA)

The viewer + owner console for Peeklet. Plain Vite + vanilla JS, no framework.
Talks to the Go backend (Lambda Function URL) over `fetch`; media opens in a
[Bigger Picture](https://github.com/henrygd/bigger-picture) lightbox.

- **Viewer** (`index.html`): recipient opens `/s/{slug}`, enters the OTP, browses
  folders + media, opens the lightbox. Photos + video.
- **Console** (`admin.html`, served at `/admin`): owner signs in, creates share
  links (URL + one-time code), and revokes them.

## Configure

One setting: the backend base URL (your Lambda Function URL from the SAM stack
output). Copy `.env.example` to `.env` for local dev:

```
VITE_API_BASE=https://xxxx.lambda-url.ap-southeast-1.on.aws
```

## Develop

```bash
npm install
npm run dev        # http://localhost:5173  (viewer)
                   # http://localhost:5173/admin.html  (console)
npm run build      # -> dist/
npm run preview    # serve the production build locally
```

## Deploy to Cloudflare Pages

1. Push the repo; in Cloudflare Pages, **Create project → connect the repo**.
2. Build settings:
   - **Root directory:** `web`
   - **Build command:** `npm run build`
   - **Build output directory:** `dist`
3. **Environment variables** → add `VITE_API_BASE` (Production *and* Preview) =
   your Lambda Function URL.
4. Deploy. You'll get `https://<project>.pages.dev`.

`public/_redirects` handles routing: `/admin` serves the console, and every
other path (including `/s/{slug}`) serves the viewer so deep links work.

## Connect the two sides

After the first Pages deploy, set the backend's `AllowedOrigin` parameter to the
Pages URL and redeploy the stack, so CORS allows the frontend:

```bash
# from the repo root (backend)
sam deploy --parameter-overrides "AllowedOrigin=https://<project>.pages.dev" ...
```

That `AllowedOrigin` is also what the console uses to build share links
(`{AllowedOrigin}/s/{slug}`), so it must be the real viewer URL.

## Notes

- Auth is a Bearer JWT kept in `sessionStorage` (viewer per-slug, admin
  separately); it clears when the tab closes and respects the token's own expiry.
- Media URLs arrive already signed from the API — the frontend never sees AWS
  credentials or the CloudFront domain directly.
- Thumbnails currently load the full signed image; when the backend adds a
  thumbnail step, point `tile img` / lightbox `thumb` at the smaller variant.