import { list } from "../api.js";
import { openLightbox } from "./lightbox.js";

const FOLDER_SVG = `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h6a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>`;
const PLAY_SVG = `<svg viewBox="0 0 24 24" fill="#fff"><circle cx="12" cy="12" r="12" fill="rgba(0,0,0,.45)"/><path d="M9.5 8.2v7.6L16 12z"/></svg>`;

function daysLeft(expiresAt) {
  if (!expiresAt) return null;
  const d = Math.ceil((expiresAt * 1000 - Date.now()) / 86400000);
  return d > 0 ? d : 0;
}

function segmentsAfterRoot(root, current) {
  const rest = current.startsWith(root) ? current.slice(root.length) : "";
  const parts = rest.split("/").filter(Boolean);
  const crumbs = [];
  let acc = root;
  for (const p of parts) {
    acc += p + "/";
    crumbs.push({ name: p, path: acc });
  }
  return crumbs;
}

export function renderGallery(root, { slug, session, onExpired }) {
  const rootPrefix = session.root;
  const rootName = rootPrefix.replace(/\/$/, "").split("/").pop() || "Gallery";
  let current = rootPrefix;
  let files = []; // current folder's viewable files, for the lightbox

  root.innerHTML = `
    <div class="topbar">
      <div class="wordmark"><span class="glyph">◗</span> Peeklet</div>
      <div class="divider"></div>
      <div class="crumbs" id="crumbs"></div>
      <div class="topbar-right">
        <span class="count" id="count"></span>
        <span class="chip" id="expiry"></span>
      </div>
    </div>
    <div class="gallery-body" id="body"></div>`;

  const crumbsEl = root.querySelector("#crumbs");
  const countEl = root.querySelector("#count");
  const expiryEl = root.querySelector("#expiry");
  const bodyEl = root.querySelector("#body");

  const dl = daysLeft(session.expiresAt);
  expiryEl.textContent = dl === null ? "" : dl === 0 ? "expires today" : `expires in ${dl}d`;
  if (dl === null) expiryEl.style.display = "none";

  function renderCrumbs() {
    const crumbs = segmentsAfterRoot(rootPrefix, current);
    let html = `<button data-path="${rootPrefix}" ${crumbs.length ? "" : 'class="here"'}>${rootName}</button>`;
    crumbs.forEach((c, i) => {
      const isLast = i === crumbs.length - 1;
      html += `<span class="slash">/</span><button data-path="${c.path}" ${isLast ? 'class="here"' : ""}>${c.name}</button>`;
    });
    crumbsEl.innerHTML = html;
    crumbsEl.querySelectorAll("button").forEach((b) => {
      b.addEventListener("click", () => navigate(b.dataset.path));
    });
  }

  async function navigate(path) {
    current = path;
    renderCrumbs();
    bodyEl.innerHTML = `<div class="empty"><span class="spinner" style="border-color:#333;border-top-color:var(--accent)"></span></div>`;
    try {
      const data = await list(slug, session.token, path === rootPrefix ? "" : path);
      render(data);
    } catch (err) {
      if (err.status === 401 || err.status === 410) { onExpired(err); return; }
      bodyEl.innerHTML = `<div class="empty">Couldn't load this folder.<br/><button class="btn ghost small" style="margin-top:14px" id="retry">Try again</button></div>`;
      bodyEl.querySelector("#retry")?.addEventListener("click", () => navigate(path));
    }
  }

  function render(data) {
    files = data.files || [];
    const folders = data.folders || [];
    countEl.textContent = `${folders.length} folder${folders.length === 1 ? "" : "s"} · ${files.length} item${files.length === 1 ? "" : "s"}`;

    let html = "";
    if (folders.length) {
      html += `<div class="section-label">Folders</div><div class="folders">`;
      html += folders
        .map(
          (f) => `<button class="folder" data-path="${f.path}">
            <span class="ic">${FOLDER_SVG}</span>
            <span class="meta"><b>${escapeHtml(f.name)}</b><span>Folder</span></span>
          </button>`
        )
        .join("");
      html += `</div>`;
    }

    if (files.length) {
      html += `<div class="section-label">Photos &amp; videos</div><div class="masonry" id="masonry">`;
      html += files
        .map(
          (f, i) => `<button class="tile" data-i="${i}" aria-label="${escapeHtml(f.name)}">
            <img src="${f.url}" alt="${escapeHtml(f.name)}" loading="lazy" />
            ${f.type === "video" ? `<span class="vbadge">VIDEO</span><span class="play">${PLAY_SVG}</span>` : ""}
            <span class="cap">${escapeHtml(f.name)}</span>
          </button>`
        )
        .join("");
      html += `</div>`;
    }

    if (!folders.length && !files.length) {
      html = `<div class="empty">This folder is empty.</div>`;
    }

    bodyEl.innerHTML = html;

    bodyEl.querySelectorAll(".folder").forEach((b) => {
      b.addEventListener("click", () => navigate(b.dataset.path));
    });
    bodyEl.querySelectorAll(".tile").forEach((b) => {
      b.addEventListener("click", () => openLightbox(files, Number(b.dataset.i)));
    });
  }

  renderCrumbs();
  navigate(rootPrefix);
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c])
  );
}