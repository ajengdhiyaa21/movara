(function () {
  var widget = document.querySelector('[data-pm-chat-widget]');
  if (!widget) return;
  var panel = widget.querySelector('[data-pm-chat-panel]');
  var toggle = widget.querySelector('[data-pm-chat-toggle]');
  var close = widget.querySelector('[data-pm-chat-close]');
  var messages = widget.querySelector('[data-pm-chat-messages]');
  var empty = widget.querySelector('[data-pm-chat-empty]');
  var form = widget.querySelector('[data-pm-chat-form]');
  var body = form.querySelector('[name="body"]');
  var submit = form.querySelector('button[type="submit"]');
  var status = widget.querySelector('[data-pm-chat-status]');
  var errorBox = widget.querySelector('[data-pm-chat-error]');
  var conversationID = 0, csrfToken = '', lastMessageID = 0, socket = null;
  var seen = {}, reconnectTimer = null, delay = 1000, closing = false, loading = false;

  function showError(message) { errorBox.textContent = message || 'Chat tidak dapat dimuat.'; errorBox.classList.remove('hidden'); }
  function clearError() { errorBox.textContent = ''; errorBox.classList.add('hidden'); }
  function escapeHTML(value) { var e = document.createElement('div'); e.textContent = value == null ? '' : value; return e.innerHTML; }
  function time(value) { var d = new Date(value); return isNaN(d) ? '' : d.toLocaleTimeString('id-ID', {hour: '2-digit', minute: '2-digit'}); }
  function setReady(ready) { body.disabled = !ready; submit.disabled = !ready; }
  function render(items, replace) {
    if (replace) seen = {};
    (items || []).forEach(function (item) { if (item && item.id) seen[String(item.id)] = item; });
    var rows = Object.keys(seen).map(function (id) { return seen[id]; }).sort(function (a, b) { return Number(a.id) - Number(b.id); });
    if (!rows.length) { if (empty) empty.classList.remove('hidden'); return; }
    if (empty) empty.classList.add('hidden');
    lastMessageID = Number(rows[rows.length - 1].id) || lastMessageID;
    messages.innerHTML = rows.map(function (item) {
      return '<div class="flex justify-start"><div class="max-w-[88%] rounded-2xl rounded-bl-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700"><p class="whitespace-pre-wrap break-words">' + escapeHTML(item.body) + '</p><span class="mt-1 block text-[10px] text-slate-400">' + time(item.created_at) + '</span></div></div>';
    }).join('');
    messages.scrollTop = messages.scrollHeight;
  }
  function markRead(items) {
    if (!items || !items.length || !csrfToken || !conversationID) return;
    fetch('/chat/read', {method:'POST', headers:{'Content-Type':'application/json','X-CSRF-Token':csrfToken}, body:JSON.stringify({conversation_id:conversationID, message_id:items[items.length - 1].id})}).catch(function () {});
  }
  function sync(replace) {
    var url = '/chat' + (replace || !lastMessageID ? '' : '?conversation_id=' + encodeURIComponent(conversationID) + '&after_id=' + encodeURIComponent(lastMessageID));
    return fetch(url, {headers:{Accept:'application/json'}}).then(function (r) { return r.json().then(function (d) { if (!r.ok) throw new Error(d.error || 'Chat tidak dapat dimuat.'); return d; }); }).then(function (d) {
      conversationID = Number(d.conversation_id) || conversationID; csrfToken = d.csrf_token || csrfToken; render(d.messages, !!replace); markRead(d.messages); setReady(Boolean(conversationID && csrfToken));
    });
  }
  function reconnect() { if (closing || reconnectTimer) return; status.textContent = 'Menyambungkan ulang...'; reconnectTimer = setTimeout(function () { reconnectTimer = null; connect(); }, delay); delay = Math.min(delay * 2, 30000); }
  function connect() {
    if (closing || socket || !window.WebSocket) return;
    socket = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/chat/ws');
    socket.onopen = function () { delay = 1000; status.textContent = 'Terhubung'; sync(false).catch(function (e) { showError(e.message); }); };
    socket.onmessage = function (event) { var d; try { d = JSON.parse(event.data); } catch (_) { return; } if (!d || d.type !== 'chat.message.created' || Number(d.conversation_id) !== conversationID || Number(d.message_id) <= lastMessageID) return; sync(false).catch(function (e) { showError(e.message); }); };
    socket.onerror = function () { if (socket) socket.close(); };
    socket.onclose = function () { socket = null; if (!closing) reconnect(); };
  }
  function open() { panel.classList.remove('hidden'); toggle.classList.add('hidden'); if (!loading && !conversationID) { loading = true; clearError(); sync(true).catch(function (e) { showError(e.message); }).finally(function () { loading = false; }); } }
  function hide() { panel.classList.add('hidden'); if (!document.querySelector('a.floating-chat')) toggle.classList.remove('hidden'); }
  toggle.addEventListener('click', open); close.addEventListener('click', hide);
  var legacy = document.querySelector('a.floating-chat');
  if (legacy) legacy.addEventListener('click', function (e) { e.preventDefault(); open(); }); else toggle.classList.remove('hidden');
  form.addEventListener('submit', function (e) { e.preventDefault(); if (!conversationID || !csrfToken || !body.value.trim()) return; var text = body.value; setReady(false); clearError(); var id = window.crypto && crypto.randomUUID ? crypto.randomUUID() : 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) { var r = Math.random()*16|0, v = c === 'x' ? r : (r&3|8); return v.toString(16); }); fetch('/chat/messages', {method:'POST', headers:{'Content-Type':'application/json','X-CSRF-Token':csrfToken}, body:JSON.stringify({conversation_id:conversationID, client_message_id:id, body:text})}).then(function (r) { return r.json().then(function (d) { if (!r.ok) throw new Error(d.error || 'Pesan gagal dikirim.'); return d; }); }).then(function (d) { body.value = ''; if (!d.existing) return sync(false); }).catch(function (err) { showError(err.message); }).finally(function () { setReady(Boolean(conversationID && csrfToken)); }); });
  window.addEventListener('beforeunload', function () { closing = true; if (reconnectTimer) clearTimeout(reconnectTimer); if (socket) socket.close(); });
  connect();
})();
