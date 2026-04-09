const $ = (id) => document.getElementById(id);

function setStatus(kind, text) {
  const el = $("status");
  el.className = "status";
  if (kind) el.classList.add(`status--${kind}`);
  el.textContent = text || "";
}

function setOutput(text) {
  $("output").textContent = text || "";
}

function setLoading(isLoading) {
  document.querySelectorAll("button.btn").forEach((b) => (b.disabled = isLoading));
  if (isLoading) setStatus("loading", "loading…");
}

/** Заголовки как у GREEN-API: X-Instance-Id, X-Api-Token */
function connectHeaders() {
  const id = $("idInstance").value.trim();
  const tok = $("apiTokenInstance").value.trim();
  const h = {};
  if (id) h["X-Instance-Id"] = id;
  if (tok) h["X-Api-Token"] = tok;
  return h;
}

function readSendMessage() {
  return {
    chatId: $("sm_chatId").value.trim(),
    message: $("sm_message").value,
  };
}

function readSendFileByUrl() {
  return {
    chatId: $("sf_chatId").value.trim(),
    fileUrl: $("sf_fileUrl").value.trim(),
    fileName: $("sf_fileName").value.trim(),
    caption: $("sf_caption").value.trim(),
  };
}

async function callAPI(method, path, jsonBody) {
  setLoading(true);
  setOutput("");
  const headers = { ...connectHeaders() };
  let body;
  if (method !== "GET" && jsonBody !== undefined) {
    headers["Content-Type"] = "application/json; charset=utf-8";
    body = JSON.stringify(jsonBody);
  }
  try {
    const res = await fetch(path, { method, headers, body });
    // Читаем тело ровно один раз: после неудачного res.json() поток уже израсходован — второй .text() даёт TypeError.
    const text = await res.text();
    let data;
    try {
      data = text ? JSON.parse(text) : null;
    } catch {
      setStatus("err", "Ошибка: сервер вернул не-JSON");
      setOutput(text);
      return;
    }
    if (data === null || typeof data !== "object") {
      setStatus("err", "Ошибка: неожиданный ответ сервера");
      setOutput(text);
      return;
    }
    if (!res.ok || !data.ok) {
      setStatus("err", data?.error?.message || "Ошибка");
      setOutput(JSON.stringify(data, null, 2));
      return;
    }
    setStatus("ok", "ok");
    setOutput(data.pretty || "");
  } catch (e) {
    setStatus("err", "Ошибка сети");
    setOutput(String(e));
  } finally {
    setLoading(false);
  }
}

document.addEventListener("click", (ev) => {
  const btn = ev.target.closest("button[data-action]");
  if (!btn) return;

  const action = btn.getAttribute("data-action");
  if (action === "getSettings") return callAPI("GET", "/api/v1/get-settings");
  if (action === "getStateInstance") return callAPI("GET", "/api/v1/get-state-instance");
  if (action === "sendMessage") return callAPI("POST", "/api/v1/send-message", readSendMessage());
  if (action === "sendFileByUrl") {
    const fn = $("sf_fileName");
    if (!fn.value.trim()) {
      fn.reportValidity();
      setStatus("err", "Укажите fileName");
      setOutput("");
      return;
    }
    return callAPI("POST", "/api/v1/send-file-by-url", readSendFileByUrl());
  }
});
