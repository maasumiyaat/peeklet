import "./styles.css";
import { verify } from "./api.js";
import { saveViewer, loadViewer, clearViewer } from "./auth.js";
import { renderUnlock } from "./ui/unlock.js";
import { renderGallery } from "./ui/gallery.js";

const app = document.getElementById("app");

// Short links are /s/{slug}. Also accept ?s={slug} as a fallback.
function getSlug() {
  const m = location.pathname.match(/^\/s\/([A-Za-z0-9_-]+)/);
  if (m) return m[1];
  const q = new URLSearchParams(location.search).get("s");
  return q || null;
}

function landing() {
  app.innerHTML = `
    <div class="center-card"><div class="center-inner">
      <div class="wordmark" style="justify-content:center"><span class="glyph">◗</span> Peeklet</div>
      <h1>Open a share link to view a gallery</h1>
      <p>This page shows a private set of photos and videos when you open the link you were sent. There's nothing to see here on its own.</p>
    </div></div>`;
}

function showGallery(slug, session) {
  renderGallery(app, {
    slug,
    session,
    onExpired: () => {
      clearViewer(slug);
      showUnlock(slug, "Your session ended. Enter your code again to continue.");
    },
  });
}

function showUnlock(slug, note) {
  renderUnlock(app, {
    onSubmit: async (code) => {
      const res = await verify(slug, code);
      const session = { token: res.token, root: res.root, expiresAt: res.expiresAt };
      saveViewer(slug, session);
      showGallery(slug, session);
    },
  });
  if (note) {
    const msg = app.querySelector("#unlock-msg");
    if (msg) msg.innerHTML = `<div class="msg ok">${note}</div>`;
  }
}

function boot() {
  const slug = getSlug();
  if (!slug) return landing();

  const existing = loadViewer(slug);
  if (existing) showGallery(slug, existing);
  else showUnlock(slug);
}

boot();