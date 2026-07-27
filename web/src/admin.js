import "./styles.css";
import { adminLogin, createShare, listShares, deleteShare } from "./api.js";
import { saveAdmin, loadAdmin, clearAdmin } from "./auth.js";

const app = document.getElementById("app");

function boot() {
  const token = loadAdmin();
  if (token) renderConsole(token);
  else renderLogin();
}

// ---------- login ----------
function renderLogin(note) {
  app.innerHTML = `
    <div class="unlock">
      <div class="unlock-card">
        <div class="unlock-brand wordmark" style="justify-content:center"><span class="glyph">◗</span> Peeklet</div>
        <div class="unlock-eyebrow">Owner console</div>
        <h1 class="unlock-title">Sign in</h1>
        <p class="unlock-sub">Enter the owner password to create and manage share links.</p>
        <div id="login-msg">${note ? `<div class="msg error">${note}</div>` : ""}</div>
        <div class="field"><input class="input" type="password" id="pw" placeholder="Owner password" autocomplete="current-password" /></div>
        <button class="btn" id="login-btn" style="width:100%">Sign in</button>
      </div>
    </div>`;

  const pw = app.querySelector("#pw");
  const btn = app.querySelector("#login-btn");
  const msg = app.querySelector("#login-msg");

  async function submit() {
    btn.disabled = true;
    btn.innerHTML = `<span class="spinner"></span> Signing in…`;
    try {
      const res = await adminLogin(pw.value);
      saveAdmin(res.token);
      renderConsole(res.token);
    } catch (err) {
      btn.disabled = false;
      btn.textContent = "Sign in";
      msg.innerHTML = `<div class="msg error">${err.status === 401 ? "Wrong password." : err.message}</div>`;
    }
  }
  btn.addEventListener("click", submit);
  pw.addEventListener("keydown", (e) => { if (e.key === "Enter") submit(); });
  pw.focus();
}

// ---------- console ----------
function renderConsole(token) {
  app.innerHTML = `
    <div class="admin-wrap">
      <div class="admin-head">
        <div class="wordmark"><span class="glyph">◗</span> Peeklet</div>
        <span class="chip">Owner console</span>
        <button class="btn ghost small" id="logout">Sign out</button>
      </div>

      <div class="panel">
        <h2>Create a share</h2>
        <p class="hint">Point a link at an S3 folder. The recipient gets the URL plus a one-time code.</p>
        <div id="create-msg"></div>
        <div class="field">
          <label>Folder prefix (e.g. clients/wedding-jan/)</label>
          <input class="input" id="prefix" placeholder="clients/wedding-jan/" />
        </div>
        <div class="row">
          <div class="field"><label>Label (optional)</label><input class="input" id="label" placeholder="Jan wedding" /></div>
          <div class="field"><label>Expiry (optional, e.g. 72h)</label><input class="input" id="ttl" placeholder="default 15 days" /></div>
        </div>
        <button class="btn" id="create-btn" style="margin-top:6px">Create share</button>
        <div id="create-result"></div>
      </div>

      <div class="panel">
        <h2>Active shares</h2>
        <p class="hint">Codes aren't shown again after creation. Revoke to end access immediately.</p>
        <div id="shares"></div>
      </div>
    </div>`;

  app.querySelector("#logout").addEventListener("click", () => { clearAdmin(); renderLogin(); });

  const createBtn = app.querySelector("#create-btn");
  const createMsg = app.querySelector("#create-msg");
  const resultEl = app.querySelector("#create-result");

  createBtn.addEventListener("click", async () => {
    const prefix = app.querySelector("#prefix").value.trim();
    const label = app.querySelector("#label").value.trim();
    const ttl = app.querySelector("#ttl").value.trim();
    createMsg.innerHTML = "";
    if (!prefix) { createMsg.innerHTML = `<div class="msg error">Enter a folder prefix.</div>`; return; }

    createBtn.disabled = true;
    createBtn.innerHTML = `<span class="spinner"></span> Creating…`;
    try {
      const payload = { prefix };
      if (label) payload.label = label;
      if (ttl) payload.ttlOverride = ttl;
      const res = await createShare(token, payload);
      resultEl.innerHTML = renderResult(res);
      wireCopy(resultEl);
      app.querySelector("#prefix").value = "";
      app.querySelector("#label").value = "";
      app.querySelector("#ttl").value = "";
      loadShares(token);
    } catch (err) {
      if (err.status === 401) { clearAdmin(); return renderLogin("Session ended. Sign in again."); }
      createMsg.innerHTML = `<div class="msg error">${err.message}</div>`;
    } finally {
      createBtn.disabled = false;
      createBtn.textContent = "Create share";
    }
  });

  loadShares(token);
}

function renderResult(res) {
  return `
    <div class="result">
      <div class="kv"><label>Link</label><div class="val" data-copy="${res.url}">${res.url}</div><button class="btn ghost small" data-copybtn>Copy</button></div>
      <div class="kv"><label>Code</label><div class="val otpval" data-copy="${res.otp}">${res.otp}</div><button class="btn ghost small" data-copybtn>Copy</button></div>
      <p class="hint" style="margin:6px 0 0">Send the link and code separately. The code won't be shown again.</p>
    </div>`;
}

function wireCopy(scope) {
  scope.querySelectorAll("[data-copybtn]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const val = btn.previousElementSibling?.dataset.copy || "";
      navigator.clipboard?.writeText(val);
      const t = btn.textContent;
      btn.textContent = "Copied";
      setTimeout(() => (btn.textContent = t), 1200);
    });
  });
}

async function loadShares(token) {
  const el = app.querySelector("#shares");
  el.innerHTML = `<div class="empty" style="padding:20px"><span class="spinner" style="border-color:#333;border-top-color:var(--accent)"></span></div>`;
  try {
    const shares = await listShares(token);
    if (!shares.length) { el.innerHTML = `<div class="empty" style="padding:20px">No active shares yet.</div>`; return; }
    shares.sort((a, b) => b.createdAt - a.createdAt);
    el.innerHTML = `<ul class="share-list">${shares
      .map((s) => {
        const dl = Math.ceil((s.expiresAt * 1000 - Date.now()) / 86400000);
        return `<li class="share-item">
          <div class="smeta">
            <b>${escapeHtml(s.label || s.prefix)}</b>
            <span>${escapeHtml(s.prefix)} · /s/${s.slug} · ${dl > 0 ? `expires in ${dl}d` : "expired"}</span>
          </div>
          <button class="btn danger small" data-del="${s.slug}">Revoke</button>
        </li>`;
      })
      .join("")}</ul>`;
    el.querySelectorAll("[data-del]").forEach((btn) => {
      btn.addEventListener("click", async () => {
        btn.disabled = true;
        try { await deleteShare(token, btn.dataset.del); loadShares(token); }
        catch { btn.disabled = false; }
      });
    });
  } catch (err) {
    if (err.status === 401) { clearAdmin(); return renderLogin("Session ended. Sign in again."); }
    el.innerHTML = `<div class="empty" style="padding:20px">Couldn't load shares.</div>`;
  }
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c])
  );
}

boot();