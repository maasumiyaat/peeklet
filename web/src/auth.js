// Session storage for tokens. sessionStorage clears when the tab closes, which
// suits short-lived viewer access; tokens also carry their own JWT expiry.

const vKey = (slug) => `peeklet:v:${slug}`;
const A_KEY = "peeklet:admin";

export function saveViewer(slug, session) {
  sessionStorage.setItem(vKey(slug), JSON.stringify(session));
}

export function loadViewer(slug) {
  try {
    const s = JSON.parse(sessionStorage.getItem(vKey(slug)));
    if (s && s.token && (!s.expiresAt || s.expiresAt * 1000 > Date.now())) return s;
  } catch { /* ignore */ }
  return null;
}

export function clearViewer(slug) {
  sessionStorage.removeItem(vKey(slug));
}

export function saveAdmin(token) {
  sessionStorage.setItem(A_KEY, token);
}

export function loadAdmin() {
  return sessionStorage.getItem(A_KEY) || null;
}

export function clearAdmin() {
  sessionStorage.removeItem(A_KEY);
}