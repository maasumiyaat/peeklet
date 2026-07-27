// Thin fetch wrapper around the Peeklet backend.
// VITE_API_BASE is the Lambda Function URL, injected at build time.
const API_BASE = (import.meta.env.VITE_API_BASE || "").replace(/\/$/, "");

export class ApiError extends Error {
  constructor(status, message, data) {
    super(message);
    this.status = status;
    this.data = data;
  }
}

async function request(path, { method = "GET", token, body } = {}) {
  if (!API_BASE) {
    throw new ApiError(0, "API base URL is not configured (set VITE_API_BASE).");
  }
  const headers = {};
  if (token) headers["authorization"] = `Bearer ${token}`;
  if (body !== undefined) headers["content-type"] = "application/json";

  let res;
  try {
    res = await fetch(API_BASE + path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch (e) {
    throw new ApiError(0, "Can't reach the server. Check your connection.");
  }

  let data = null;
  try { data = await res.json(); } catch { /* empty body */ }

  if (!res.ok) {
    throw new ApiError(res.status, data?.error || `Request failed (${res.status})`, data);
  }
  return data;
}

// ---- viewer ----
export const verify = (slug, otp) =>
  request(`/api/${encodeURIComponent(slug)}/verify`, { method: "POST", body: { otp } });

export const list = (slug, token, path = "", pageToken = "") => {
  const q = new URLSearchParams();
  if (path) q.set("path", path);
  if (pageToken) q.set("token", pageToken);
  const qs = q.toString();
  return request(`/api/${encodeURIComponent(slug)}/list${qs ? "?" + qs : ""}`, { token });
};

// ---- admin ----
export const adminLogin = (password) =>
  request(`/admin/login`, { method: "POST", body: { password } });

export const createShare = (token, payload) =>
  request(`/admin/shares`, { method: "POST", token, body: payload });

export const listShares = (token) =>
  request(`/admin/shares`, { token });

export const deleteShare = (token, slug) =>
  request(`/admin/shares/${encodeURIComponent(slug)}`, { method: "DELETE", token });