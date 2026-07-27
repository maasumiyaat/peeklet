// Unlock screen: 8-box OTP entry with paste + auto-advance.
const OTP_LEN = 8;

export function renderUnlock(root, { onSubmit }) {
  root.innerHTML = `
    <div class="unlock">
      <div class="unlock-card">
        <div class="unlock-brand wordmark"><span class="glyph">◗</span> Peeklet</div>
        <div class="unlock-eyebrow">Private gallery</div>
        <h1 class="unlock-title">Enter your access code</h1>
        <p class="unlock-sub">This link opens a private set of photos and videos. Type the ${OTP_LEN}-character code you were sent to unlock it.</p>
        <div id="unlock-msg"></div>
        <div class="otp" id="otp"></div>
        <button class="btn" id="unlock-btn">Unlock gallery</button>
        <div class="unlock-foot"><span class="chip" id="unlock-chip">Enter your code</span></div>
      </div>
    </div>`;

  const otp = root.querySelector("#otp");
  const btn = root.querySelector("#unlock-btn");
  const msg = root.querySelector("#unlock-msg");

  const inputs = [];
  for (let i = 0; i < OTP_LEN; i++) {
    const el = document.createElement("input");
    el.type = "text";
    el.inputMode = "text";
    el.autocomplete = "one-time-code";
    el.maxLength = 1;
    el.setAttribute("aria-label", `Character ${i + 1}`);
    otp.appendChild(el);
    inputs.push(el);
  }

  const value = () => inputs.map((i) => i.value).join("");
  const sanitize = (c) => c.toUpperCase().replace(/[^A-Z0-9]/g, "");

  inputs.forEach((el, i) => {
    el.addEventListener("input", () => {
      el.value = sanitize(el.value).slice(0, 1);
      el.classList.toggle("filled", !!el.value);
      if (el.value && i < OTP_LEN - 1) inputs[i + 1].focus();
      if (value().length === OTP_LEN) submit();
    });
    el.addEventListener("keydown", (e) => {
      if (e.key === "Backspace" && !el.value && i > 0) inputs[i - 1].focus();
      if (e.key === "Enter") submit();
    });
    el.addEventListener("paste", (e) => {
      e.preventDefault();
      const text = sanitize(e.clipboardData.getData("text")).slice(0, OTP_LEN);
      inputs.forEach((inp, k) => {
        inp.value = text[k] || "";
        inp.classList.toggle("filled", !!inp.value);
      });
      inputs[Math.min(text.length, OTP_LEN - 1)].focus();
      if (text.length === OTP_LEN) submit();
    });
  });

  let busy = false;
  async function submit() {
    if (busy) return;
    const code = value();
    if (code.length !== OTP_LEN) {
      show("error", "Enter all 8 characters.");
      return;
    }
    busy = true;
    btn.disabled = true;
    btn.innerHTML = `<span class="spinner"></span> Unlocking…`;
    msg.innerHTML = "";
    try {
      await onSubmit(code);
    } catch (err) {
      busy = false;
      btn.disabled = false;
      btn.textContent = "Unlock gallery";
      const m =
        err.status === 401 ? "That code isn't right. Check it and try again." :
        err.status === 429 ? "Too many tries. Wait a few minutes and try again." :
        err.status === 410 ? "This link has expired." :
        err.status === 404 ? "This link isn't valid." :
        err.message || "Something went wrong. Try again.";
      show("error", m);
      inputs.forEach((inp) => { inp.value = ""; inp.classList.remove("filled"); });
      inputs[0].focus();
    }
  }

  function show(kind, text) {
    msg.innerHTML = `<div class="msg ${kind}">${text}</div>`;
  }

  btn.addEventListener("click", submit);
  setTimeout(() => inputs[0].focus(), 60);
}