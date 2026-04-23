package dashboard

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/zmcp/odata-mcp/internal/models"
)

var dashboardTranslations = map[string]map[string]string{
	"ru": {
		"title":               "sap-odata-mcp-universal",
		"subtitle":            "MCP-шлюз для SAP OData",
		"h_systems":           "Системы SAP",
		"h_add_system":        "Новое подключение",
		"h_edit_system":       "Редактирование",
		"conn_name":           "Имя соединения",
		"system_name":         "Имя системы",
		"service_url":         "URL сервиса",
		"sap_client":          "SAP клиент",
		"login":               "Логин",
		"password":            "Пароль",
		"allow_write":         "Разрешить запись",
		"default_system":      "По умолчанию",
		"btn_connect":         "Подключить",
		"btn_save":            "Сохранить",
		"btn_cancel":          "Отмена",
		"btn_refresh":         "Обновить",
		"btn_edit":            "Изменить",
		"btn_delete":          "Удалить",
		"btn_docs":            "Документация",
		"connected":           "Подключена",
		"disconnected":        "Отключена",
		"rw":                  "Чтение/Запись",
		"ro":                  "Только чтение",
		"no_systems":          "Нет подключённых SAP систем. Добавьте первую.",
		"confirm_delete":      "Удалить соединение",
		"confirm_delete_text": "Вы уверены? Активное SAP OData подключение будет удалено.",
		"fill_fields":         "Заполните обязательные поля: имя, система, URL сервиса, логин, пароль",
		"msg_connected":       "Подключено",
		"msg_disconnected":    "Отключено",
		"msg_saved":           "Сохранено",
		"msg_default_set":     "Установлено по умолчанию",
		"msg_error":           "Ошибка",
	},
	"en": {
		"title":               "sap-odata-mcp-universal",
		"subtitle":            "MCP gateway for SAP OData",
		"h_systems":           "SAP Systems",
		"h_add_system":        "New Connection",
		"h_edit_system":       "Edit Connection",
		"conn_name":           "Connection Name",
		"system_name":         "System Name",
		"service_url":         "Service URL",
		"sap_client":          "SAP Client",
		"login":               "Login",
		"password":            "Password",
		"allow_write":         "Allow writes",
		"default_system":      "Default",
		"btn_connect":         "Connect",
		"btn_save":            "Save",
		"btn_cancel":          "Cancel",
		"btn_refresh":         "Refresh",
		"btn_edit":            "Edit",
		"btn_delete":          "Delete",
		"btn_docs":            "Documentation",
		"connected":           "Connected",
		"disconnected":        "Disconnected",
		"rw":                  "Read/Write",
		"ro":                  "Read-only",
		"no_systems":          "No SAP systems connected yet. Add your first one.",
		"confirm_delete":      "Delete connection",
		"confirm_delete_text": "Are you sure? The active SAP OData connection will be removed.",
		"fill_fields":         "Fill required fields: name, system, service URL, login, password",
		"msg_connected":       "Connected",
		"msg_disconnected":    "Disconnected",
		"msg_saved":           "Saved",
		"msg_default_set":     "Set as default",
		"msg_error":           "Error",
	},
}

func renderDashboard(lang string) (string, error) {
	lang = normalizeLang(lang)
	t := dashboardTranslations[lang]
	tJSON, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>sap-odata-mcp-universal</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;background:#0f172a;color:#f8fafc;height:100vh;display:flex;flex-direction:column;overflow:hidden}
.content{flex:1;overflow-y:auto}
.header{background:#1e293b;border-bottom:1px solid #334155;padding:8px 20px;display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:8px;flex-shrink:0}
.header-left{display:flex;align-items:center;gap:10px}
.header h1{font-size:1.05rem;color:#f8fafc;font-weight:700}
.header .sub{color:#94a3b8;font-size:.75rem}
.header-right{display:flex;align-items:center;gap:6px;flex-wrap:wrap}
.lang-sw{display:flex;border:1px solid #475569;border-radius:5px;overflow:hidden}
.lang-sw a{padding:3px 8px;font-size:.7rem;color:#94a3b8;display:block;text-decoration:none}
.lang-sw a.on{background:#334155;color:#f8fafc}
.btn{display:inline-flex;align-items:center;gap:4px;padding:5px 12px;border-radius:5px;font-size:.78rem;cursor:pointer;border:1px solid #475569;background:#1e293b;color:#94a3b8;text-decoration:none;transition:.15s}
.btn:hover{background:#334155;color:#f8fafc}
.btn-d{color:#ef4444;border-color:rgba(239,68,68,.25)}
.btn-d:hover{background:rgba(239,68,68,.1);color:#ef4444;border-color:#ef4444}
.btn-ds{background:#991b1b;border-color:#991b1b;color:#fff}
.btn-ds:hover{background:#b91c1c}
.content{padding:20px}
.card{background:#1e293b;border-radius:8px;padding:12px;border:1px solid #334155;overflow:hidden;margin-bottom:14px}
.card h2{font-size:.65rem;color:#94a3b8;text-transform:uppercase;letter-spacing:.06em;margin-bottom:8px;font-weight:600}
.db-item{background:#0f172a;border:1px solid #334155;border-radius:6px;padding:10px 12px;margin-bottom:8px;transition:border-color .15s}
.db-item:last-child{margin-bottom:0}
.db-item:hover{border-color:#475569}
.db-row{display:flex;align-items:center;gap:10px}
.dot{width:8px;height:8px;border-radius:50%%;flex-shrink:0}
.dot.ok{background:#22c55e}
.dot.err{background:#ef4444}
.db-info{flex:1;min-width:0}
.db-name{font-weight:600;font-size:.88rem}
.db-details{color:#94a3b8;font-size:.75rem;font-family:'SF Mono','Cascadia Code',monospace;margin-top:1px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.db-badges{display:flex;gap:5px;margin-top:4px;flex-wrap:wrap}
.db-actions{display:flex;gap:4px;flex-shrink:0}
.badge{display:inline-flex;align-items:center;padding:1px 6px;border-radius:3px;font-size:.62rem;font-weight:600}
.badge-g{background:rgba(34,197,94,.12);color:#22c55e}
.badge-r{background:rgba(239,68,68,.12);color:#ef4444}
.badge-b{background:rgba(59,130,246,.12);color:#3b82f6}
.badge-c{background:#164e63;color:#22d3ee}
.rd{display:flex;align-items:center;gap:5px;cursor:pointer;font-size:.72rem;color:#94a3b8;background:none;border:0;padding:0}
.rd:hover{color:#cbd5e1}
.rb{width:14px;height:14px;border-radius:50%%;border:2px solid #475569;display:flex;align-items:center;justify-content:center;transition:.15s;flex-shrink:0}
.rb.on{border-color:#22d3ee}
.rb.on::after{content:'';width:7px;height:7px;border-radius:50%%;background:#22d3ee}
.rd:hover .rb{border-color:#22d3ee}
.toggle{display:flex;align-items:center;gap:8px;cursor:pointer;font-size:.78rem;color:#cbd5e1;user-select:none}
.toggle input{display:none}
.toggle-track{width:34px;height:18px;border-radius:9px;background:#475569;position:relative;transition:.2s;flex-shrink:0}
.toggle-track::after{content:'';width:14px;height:14px;border-radius:50%%;background:#94a3b8;position:absolute;top:2px;left:2px;transition:.2s}
.toggle input:checked+.toggle-track{background:#22c55e}
.toggle input:checked+.toggle-track::after{left:18px;background:#fff}
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:8px}
.form-group{display:flex;flex-direction:column;gap:3px}
.form-group.full{grid-column:1/-1}
.form-group label{font-size:.65rem;color:#94a3b8;text-transform:uppercase;letter-spacing:.04em;font-weight:600}
input{padding:5px 8px;border-radius:4px;border:1px solid #475569;background:#0f172a;color:#e2e8f0;font-size:.8rem;transition:border .15s;width:100%%;-moz-appearance:textfield}
input::-webkit-outer-spin-button,input::-webkit-inner-spin-button{-webkit-appearance:none;margin:0}
input:focus{outline:none;border-color:#38bdf8}
.btn:focus-visible,.lang-sw a:focus-visible,input:focus-visible,.rd:focus-visible{outline:2px solid #38bdf8;outline-offset:2px}
.form-actions{display:flex;gap:6px;justify-content:flex-end;margin-top:10px}
.empty{text-align:center;padding:20px;color:#64748b;font-size:.82rem}
.overlay{position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:100;display:flex;align-items:center;justify-content:center}
.modal{background:#1e293b;border:1px solid #334155;border-radius:10px;padding:20px;width:480px;max-width:92%%}
.modal h3{font-size:.88rem;margin-bottom:14px;color:#f8fafc}
.modal p{color:#cbd5e1}
.modal-actions{display:flex;gap:6px;justify-content:flex-end;margin-top:14px}
.toast-msg{position:fixed;top:50%%;left:50%%;transform:translate(-50%%,-50%%);background:#164e63;color:#22d3ee;padding:14px 24px;border-radius:8px;font-size:.9rem;z-index:999;max-width:500px;text-align:center;box-shadow:0 4px 20px rgba(0,0,0,.5);pointer-events:none}
.toast-err{background:#7f1d1d;color:#fca5a5}
.content::-webkit-scrollbar{width:8px}
.content::-webkit-scrollbar-track{background:#0f172a}
.content::-webkit-scrollbar-thumb{background:#334155;border-radius:4px}
.content::-webkit-scrollbar-thumb:hover{background:#475569}
.content{scrollbar-width:thin;scrollbar-color:#334155 #0f172a}
.cols{display:grid;grid-template-columns:1fr 1fr;gap:14px;align-items:start}
.footer{padding:8px 20px;text-align:center;color:#475569;font-size:.68rem;border-top:1px solid #1e293b;flex-shrink:0}
.footer a{color:#64748b;text-decoration:none}.footer a:hover{color:#94a3b8}
@media(max-width:900px){.cols{grid-template-columns:1fr}}
@media(max-width:600px){.content{padding:10px}.form-grid{grid-template-columns:1fr}.db-row{flex-wrap:wrap}.db-actions{width:100%%;justify-content:flex-end}}
</style>
</head>
<body>
<div class="header">
  <div class="header-left">
    <div><h1>sap-odata-mcp-universal</h1><span class="sub">%s</span></div>
  </div>
  <div class="header-right">
    <div class="lang-sw">
      <a href="?lang=ru" class="%s">RU</a>
      <a href="?lang=en" class="%s">EN</a>
    </div>
    <a class="btn" href="/dashboard/docs?lang=%s" target="_blank">%s</a>
    <button class="btn" onclick="loadSystems()">%s</button>
  </div>
</div>
<div class="content">
<div class="cols">
  <div class="card">
    <h2>%s</h2>
    <div id="db-list" aria-live="polite"></div>
  </div>
  <div class="card">
    <h2>%s</h2>
    <div class="form-grid">
      <div class="form-group">
        <label>%s *</label>
        <input id="f-name">
      </div>
      <div class="form-group">
        <label>%s *</label>
        <input id="f-system">
      </div>
      <div class="form-group full">
        <label>%s *</label>
        <input id="f-url" placeholder="https://host/sap/opu/odata/sap/SERVICE_SRV/">
      </div>
      <div class="form-group">
        <label>%s</label>
        <input id="f-client" placeholder="100">
      </div>
      <div class="form-group">
        <label>%s *</label>
        <input id="f-user" autocomplete="off" autocapitalize="off" spellcheck="false">
      </div>
      <div class="form-group full">
        <label>%s *</label>
        <input id="f-pass" type="password" autocomplete="new-password">
      </div>
      <div class="form-group full" style="margin-top:2px">
        <label class="toggle">
          <input type="checkbox" id="f-write">
          <span class="toggle-track"></span>
          %s
        </label>
      </div>
    </div>
    <div class="form-actions">
      <button class="btn" onclick="connectSystem()">&#8594; %s</button>
    </div>
  </div>
</div>
</div>
<script>
const T = %s;

function toast(msg, isErr) {
  var d = document.createElement('div');
  d.className = 'toast-msg' + (isErr ? ' toast-err' : '');
  d.textContent = msg;
  document.body.appendChild(d);
  setTimeout(function(){ d.remove() }, 3000);
}

async function api(url, opts) {
  try {
    const r = await fetch(url, opts || {});
    return await r.json();
  } catch(e) {
    toast(T.msg_error + ': ' + e.message, true);
    return null;
  }
}

function escHtml(s) {
  var d = document.createElement('div');
  d.appendChild(document.createTextNode(s || ''));
  return d.innerHTML;
}

function escAttr(s) {
  return escHtml(s).replace(/'/g, '&#39;').replace(/"/g, '&quot;');
}

function accessModeLabel(mode) {
  return mode === 'unrestricted' ? T.rw : T.ro;
}

function detailsLine(item) {
  const parts = [item.system_name || '', item.safe_service_url || item.service_url || ''];
  if (item.client) parts.push('client=' + item.client);
  if (item.username) parts.push(item.username);
  return parts.filter(Boolean).join(' | ');
}

async function loadSystems() {
  const [items, status] = await Promise.all([api('/api/databases'), api('/api/status')]);
  const list = document.getElementById('db-list');
  const activeName = status?.active_default || '';

  if (!items || items.length === 0) {
    list.innerHTML = '<div class="empty">' + T.no_systems + '</div>';
    return;
  }

  list.innerHTML = items.map(function(item) {
    const isDefault = item.name === activeName;
    const eName = escHtml(item.name);
    const eNameAttr = escAttr(item.name);
    return '<div class="db-item">' +
      '<div class="db-row">' +
        '<div class="dot ' + (item.connected ? 'ok' : 'err') + '"></div>' +
        '<div class="db-info">' +
          '<div class="db-name">' + eName + '</div>' +
          '<div class="db-details">' + escHtml(detailsLine(item)) + '</div>' +
          '<div class="db-badges">' +
            '<span class="badge ' + (item.connected ? 'badge-g' : 'badge-r') + '">' +
              (item.connected ? T.connected : T.disconnected) + '</span>' +
            '<span class="badge ' + (item.access_mode === 'unrestricted' ? 'badge-b' : 'badge-c') + '">' +
              accessModeLabel(item.access_mode) + '</span>' +
          '</div>' +
        '</div>' +
        '<div style="display:flex;flex-direction:column;align-items:flex-end;gap:6px">' +
          '<button type="button" class="rd" aria-pressed="' + (isDefault ? 'true' : 'false') + '" onclick="setDefault(\'' + eNameAttr + '\')">' +
            '<div class="rb ' + (isDefault ? 'on' : '') + '"></div>' +
            '<span>' + T.default_system + '</span>' +
          '</button>' +
          '<div class="db-actions">' +
            '<button class="btn" onclick="editSystem(\'' + eNameAttr + '\',\'' + escAttr(item.system_name) + '\',\'' + escAttr(item.service_url) + '\',\'' + escAttr(item.client || '') + '\',\'' + escAttr(item.username || '') + '\',\'' + escAttr(item.access_mode || '') + '\')">' + T.btn_edit + '</button>' +
            '<button class="btn btn-d" onclick="confirmDelete(\'' + eNameAttr + '\')">' + T.btn_delete + '</button>' +
          '</div>' +
        '</div>' +
      '</div>' +
    '</div>';
  }).join('');
}

async function connectSystem() {
  const payload = {
    name: document.getElementById('f-name').value.trim(),
    system_name: document.getElementById('f-system').value.trim(),
    service_url: document.getElementById('f-url').value.trim(),
    client: document.getElementById('f-client').value.trim(),
    username: document.getElementById('f-user').value.trim(),
    password: document.getElementById('f-pass').value,
    access_mode: document.getElementById('f-write').checked ? 'unrestricted' : 'restricted'
  };

  if (!payload.name || !payload.system_name || !payload.service_url || !payload.username || !payload.password) {
    toast(T.fill_fields, true);
    return;
  }

  const r = await api('/api/connect', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(payload)
  });

  if (!r || r.error) {
    toast(T.msg_error + ': ' + (r?.error || 'unknown'), true);
    return;
  }

  toast(T.msg_connected + ': ' + payload.name);
  document.getElementById('f-name').value = '';
  document.getElementById('f-system').value = '';
  document.getElementById('f-url').value = '';
  document.getElementById('f-client').value = '';
  document.getElementById('f-user').value = '';
  document.getElementById('f-pass').value = '';
  document.getElementById('f-write').checked = false;
  loadSystems();
}

async function setDefault(name) {
  const r = await api('/api/switch', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({ name: name })
  });
  if (!r || r.error) {
    toast(T.msg_error + ': ' + (r?.error || 'unknown'), true);
    return;
  }
  toast(T.msg_default_set + ': ' + name);
  loadSystems();
}

function editSystem(name, systemName, serviceURL, client, username, accessMode) {
  var ov = document.createElement('div');
  ov.className = 'overlay';
  ov.innerHTML =
    '<div class="modal" role="dialog" aria-modal="true" aria-labelledby="edit-dialog-title">' +
      '<h3 id="edit-dialog-title">' + T.h_edit_system + '</h3>' +
      '<div class="form-grid">' +
        '<div class="form-group"><label>' + T.conn_name + '</label><input id="e-name" value="' + escAttr(name) + '"></div>' +
        '<div class="form-group"><label>' + T.system_name + '</label><input id="e-system" value="' + escAttr(systemName) + '"></div>' +
        '<div class="form-group full"><label>' + T.service_url + '</label><input id="e-url" value="' + escAttr(serviceURL) + '"></div>' +
        '<div class="form-group"><label>' + T.sap_client + '</label><input id="e-client" value="' + escAttr(client) + '"></div>' +
        '<div class="form-group"><label>' + T.login + '</label><input id="e-user" value="' + escAttr(username) + '" autocomplete="off" autocapitalize="off" spellcheck="false"></div>' +
        '<div class="form-group full"><label>' + T.password + '</label><input id="e-pass" type="password" placeholder="••••••••" autocomplete="new-password"></div>' +
        '<div class="form-group full" style="margin-top:2px">' +
          '<label class="toggle">' +
            '<input type="checkbox" id="e-write" ' + (accessMode === 'unrestricted' ? 'checked' : '') + '>' +
            '<span class="toggle-track"></span>' +
            T.allow_write +
          '</label>' +
        '</div>' +
      '</div>' +
      '<div class="modal-actions">' +
        '<button class="btn" onclick="this.closest(\'.overlay\').remove()">' + T.btn_cancel + '</button>' +
        '<button class="btn" onclick="saveEdit(\'' + escAttr(name) + '\')">' + T.btn_save + '</button>' +
      '</div>' +
    '</div>';
  document.body.appendChild(ov);
  ov.addEventListener('click', function(e) { if (e.target === ov) ov.remove(); });
}

async function saveEdit(oldName) {
  const payload = {
    old_name: oldName,
    name: document.getElementById('e-name').value.trim(),
    system_name: document.getElementById('e-system').value.trim(),
    service_url: document.getElementById('e-url').value.trim(),
    client: document.getElementById('e-client').value.trim(),
    username: document.getElementById('e-user').value.trim(),
    password: document.getElementById('e-pass').value,
    access_mode: document.getElementById('e-write').checked ? 'unrestricted' : 'restricted'
  };

  if (!payload.name || !payload.system_name || !payload.service_url || !payload.username) {
    toast(T.fill_fields, true);
    return;
  }

  const r = await api('/api/edit', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(payload)
  });
  if (!r || r.error) {
    toast(T.msg_error + ': ' + (r?.error || 'unknown'), true);
    return;
  }

  document.querySelector('.overlay').remove();
  toast(T.msg_saved + ': ' + payload.name);
  loadSystems();
}

function confirmDelete(name) {
  var ov = document.createElement('div');
  ov.className = 'overlay';
  ov.innerHTML =
    '<div class="modal" role="dialog" aria-modal="true" aria-labelledby="delete-dialog-title" aria-describedby="delete-dialog-text" style="width:360px;text-align:center">' +
      '<h3 id="delete-dialog-title">' + T.confirm_delete + ' "' + escHtml(name) + '"?</h3>' +
      '<p id="delete-dialog-text" style="font-size:.82rem;margin-bottom:14px">' + T.confirm_delete_text + '</p>' +
      '<div style="display:flex;gap:6px;justify-content:center">' +
        '<button class="btn" onclick="this.closest(\'.overlay\').remove()">' + T.btn_cancel + '</button>' +
        '<button class="btn btn-ds" onclick="doDelete(\'' + escAttr(name) + '\')">' + T.btn_delete + '</button>' +
      '</div>' +
    '</div>';
  document.body.appendChild(ov);
  ov.addEventListener('click', function(e) { if (e.target === ov) ov.remove(); });
}

async function doDelete(name) {
  document.querySelector('.overlay').remove();
  const r = await api('/api/disconnect', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({ name: name })
  });
  if (!r || r.error) {
    toast(T.msg_error + ': ' + (r?.error || 'unknown'), true);
    return;
  }
  toast(T.msg_disconnected + ': ' + name);
  loadSystems();
}

loadSystems();
</script>
<div class="footer">
sap-odata-mcp-universal &mdash;
<a href="https://github.com/AlekseiSeleznev/sap-odata-mcp-universal">GitHub</a> &mdash;
<a href="https://github.com/AlekseiSeleznev/sap-odata-mcp-universal/blob/main/LICENSE">MIT License</a>
</div>
</body>
</html>`,
		lang,
		html.EscapeString(t["subtitle"]),
		onClass(lang == "ru"),
		onClass(lang == "en"),
		lang,
		html.EscapeString(t["btn_docs"]),
		html.EscapeString(t["btn_refresh"]),
		html.EscapeString(t["h_systems"]),
		html.EscapeString(t["h_add_system"]),
		html.EscapeString(t["conn_name"]),
		html.EscapeString(t["system_name"]),
		html.EscapeString(t["service_url"]),
		html.EscapeString(t["sap_client"]),
		html.EscapeString(t["login"]),
		html.EscapeString(t["password"]),
		html.EscapeString(t["allow_write"]),
		html.EscapeString(t["btn_connect"]),
		string(tJSON),
	), nil
}

func renderDocs(lang string, ctx *models.DashboardDocumentationContext) string {
	lang = normalizeLang(lang)

	if lang == "en" {
		return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>sap-odata-mcp-universal — Documentation</title><style>%s</style></head><body>
<a class="back" href="/dashboard?lang=en">&larr; Dashboard</a>
<h1>sap-odata-mcp-universal</h1>
<div class="sub">MCP gateway for SAP OData with a bilingual connection dashboard</div>
<h2>Contents</h2>
<ul>
<li><a href="#e1">1. Overview</a></li>
<li><a href="#e2">2. Startup modes</a></li>
<li><a href="#e3">3. Dashboard workflow</a></li>
<li><a href="#e4">4. Connection parameters</a></li>
<li><a href="#e5">5. Connect Codex and other MCP clients</a></li>
<li><a href="#e6">6. Persistence and active system switching</a></li>
<li><a href="#e7">7. HTTP endpoints</a></li>
<li><a href="#e8">8. Security notes</a></li>
<li><a href="#e9">9. Troubleshooting</a></li>
</ul>
<h2 id="e1">1. Overview</h2>
<p>This dashboard is designed to manage multiple SAP systems exposed through OData and authenticated with login/password. It replaces the old status-centric page with a PostgreSQL-style connection manager UI.</p>
<ul>
<li>Exact two-column dashboard layout modeled after <code>postgres-mcp-universal</code></li>
<li>Saved SAP OData connection profiles with persistent state on disk</li>
<li>Switchable active default system without restarting the HTTP bridge</li>
<li>Bilingual UI and documentation: Russian and English</li>
<li>Per-connection access mode: <code>unrestricted</code> or <code>restricted</code></li>
</ul>
<h2 id="e2">2. Startup modes</h2>
<p>The HTTP dashboard can start even when no initial <code>--service</code> value is provided. This is the intended mode for operating multiple SAP systems through the dashboard.</p>
<ul>
<li>Transport: <code>%s</code></li>
<li>HTTP address: <code>%s</code></li>
<li>MCP endpoint: <code>%s</code></li>
<li>Health endpoint: <code>%s</code></li>
<li>Supports empty startup: <code>%t</code></li>
</ul>
<p>For <code>stdio</code> usage, you still normally launch the binary with a concrete OData service URL because there is no dashboard in that mode.</p>
<h2 id="e3">3. Dashboard workflow</h2>
<ol>
<li>Open <code>%s</code>.</li>
<li>Create a new SAP OData connection by specifying a connection name, SAP system name, service URL, optional SAP client, login, and password.</li>
<li>Choose whether the MCP bridge should expose write-capable tools for that profile.</li>
<li>Press <code>Connect</code>. The bridge fetches metadata immediately and rebuilds its MCP tools for the chosen SAP system.</li>
<li>Use the radio-style <code>Default</code> selector on the left to switch the active system later.</li>
<li>Use <code>Edit</code> to update a saved profile and <code>Delete</code> to remove it.</li>
</ol>
<h2 id="e4">4. Connection parameters</h2>
<table><tr><th>Field</th><th>Description</th></tr>
<tr><td><code>Connection Name</code></td><td>User-facing dashboard label for the saved profile.</td></tr>
<tr><td><code>System Name</code></td><td>Business name of the SAP landscape or environment, for example <code>S4HANA DEV</code>.</td></tr>
<tr><td><code>Service URL</code></td><td>Root URL of the SAP OData service, for example <code>https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/</code>.</td></tr>
<tr><td><code>SAP Client</code></td><td>Optional <code>sap-client</code> value. When provided, the dashboard appends it to the service URL query string.</td></tr>
<tr><td><code>Login</code></td><td>Basic-auth username used by the bridge for metadata loading and tool execution.</td></tr>
<tr><td><code>Password</code></td><td>Basic-auth password. During editing, leaving it blank keeps the previously stored password.</td></tr>
<tr><td><code>Allow writes</code></td><td>If enabled, the bridge exposes modifying tools when the metadata supports them. If disabled, the profile is forced into read-only mode.</td></tr>
</table>
<h2 id="e5">5. Connect Codex and other MCP clients</h2>
<p>Once an SAP system is active in the dashboard, MCP clients use the regular HTTP transport. The current route for this server is <code>%s</code>.</p>
<pre><code>codex mcp add sap-odata-universal --url http://localhost:3000%s</code></pre>
<p>If you prefer <code>stdio</code> mode for Codex, keep using a dedicated binary configuration for a single target system. The dashboard is primarily for HTTP operation and system switching.</p>
<h2 id="e6">6. Persistence and active system switching</h2>
<p>Saved dashboard profiles are persisted in <code>%s</code>. The state file stores:</p>
<ul>
<li>active default connection name</li>
<li>saved SAP OData profiles</li>
<li>connection-level access mode</li>
<li>credential fields required for reconnecting on restart</li>
</ul>
<p>On startup, the dashboard attempts to restore the active profile and rebuild the bridge automatically. Current active connection: <code>%s</code>. Total saved connections: <code>%d</code>.</p>
<h2 id="e7">7. HTTP endpoints</h2>
<table><tr><th>Endpoint</th><th>Method</th><th>Description</th></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Main bilingual dashboard UI.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Built-in detailed documentation page.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Active connection summary used by the dashboard JS.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>List saved SAP OData connections.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Create and immediately activate a connection.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Remove a saved connection; if it was active, the bridge switches to the next saved one or clears itself.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Update a saved connection and re-activate it if needed.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Switch the active default SAP system.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Transport health probe.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>MCP transport endpoint.</td></tr>
</table>
<h2 id="e8">8. Security notes</h2>
<ul>
<li>The dashboard stores credentials so it can reconnect after restart. Protect the host and the state file permissions.</li>
<li>The HTTP transport should still be protected with token/TLS when exposed beyond localhost.</li>
<li>The dashboard only hides passwords from the UI list; it does not encrypt them in the persisted state file.</li>
<li>Use restricted mode for landscapes where writes must never be exposed through MCP.</li>
</ul>
<h2 id="e9">9. Troubleshooting</h2>
<ul>
<li>If a connection fails during <code>Connect</code>, verify the OData root URL and credentials first.</li>
<li>If metadata loads but write tools are missing, the SAP service metadata may mark the entity sets as non-creatable/non-updatable/non-deletable.</li>
<li>If the wrong client data appears, set the <code>SAP Client</code> field explicitly so the dashboard adds <code>sap-client</code>.</li>
<li>If the bridge starts with no active tools, check whether the saved active profile can still authenticate.</li>
</ul>
</body></html>`,
			docStyle,
			html.EscapeString(ctx.Transport),
			html.EscapeString(ctx.HTTPAddr),
			html.EscapeString(ctx.MCPPath),
			html.EscapeString(ctx.HealthPath),
			ctx.SupportsEmptyStartup,
			html.EscapeString(ctx.DashboardPath),
			html.EscapeString(ctx.HTTPAddr),
			html.EscapeString(ctx.MCPPath),
			html.EscapeString(ctx.StateFile),
			html.EscapeString(emptyFallback(ctx.ActiveConnection)),
			ctx.TotalConnections,
			html.EscapeString(ctx.DashboardPath),
			html.EscapeString(ctx.DocumentationPath),
			html.EscapeString(ctx.StatusPath),
			html.EscapeString(ctx.ListPath),
			html.EscapeString(ctx.ConnectPath),
			html.EscapeString(ctx.DisconnectPath),
			html.EscapeString(ctx.EditPath),
			html.EscapeString(ctx.SwitchPath),
			html.EscapeString(ctx.HealthPath),
			html.EscapeString(ctx.MCPPath),
		)
	}

	return fmt.Sprintf(`<!DOCTYPE html><html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>sap-odata-mcp-universal — Документация</title><style>%s</style></head><body>
<a class="back" href="/dashboard?lang=ru">&larr; Dashboard</a>
<h1>sap-odata-mcp-universal</h1>
<div class="sub">MCP-шлюз для SAP OData с двуязычным dashboard для управления подключениями</div>
<h2>Оглавление</h2>
<ul>
<li><a href="#r1">1. Обзор</a></li>
<li><a href="#r2">2. Режимы запуска</a></li>
<li><a href="#r3">3. Сценарий работы через dashboard</a></li>
<li><a href="#r4">4. Параметры подключения</a></li>
<li><a href="#r5">5. Подключение Codex и других MCP-клиентов</a></li>
<li><a href="#r6">6. Хранение состояния и переключение активной системы</a></li>
<li><a href="#r7">7. HTTP endpoints</a></li>
<li><a href="#r8">8. Замечания по безопасности</a></li>
<li><a href="#r9">9. Диагностика проблем</a></li>
</ul>
<h2 id="r1">1. Обзор</h2>
<p>Этот dashboard предназначен для управления несколькими SAP системами, доступными по OData и аутентифицируемыми по логину и паролю. Он заменяет прежнюю страницу со статусами и метаданными на connection-manager в стиле <code>postgres-mcp-universal</code>.</p>
<ul>
<li>Та же двухколоночная компоновка dashboard, что и в postgres-референсе</li>
<li>Постоянный реестр SAP OData профилей на диске</li>
<li>Переключение активной системы без перезапуска HTTP bridge</li>
<li>Двуязычный интерфейс и документация: русский и английский</li>
<li>Режим доступа на уровне подключения: <code>unrestricted</code> или <code>restricted</code></li>
</ul>
<h2 id="r2">2. Режимы запуска</h2>
<p>HTTP dashboard может стартовать даже без начального <code>--service</code>. Это целевой режим для сценария, где один bridge обслуживает несколько SAP систем через web UI.</p>
<ul>
<li>Transport: <code>%s</code></li>
<li>HTTP address: <code>%s</code></li>
<li>MCP endpoint: <code>%s</code></li>
<li>Health endpoint: <code>%s</code></li>
<li>Поддержка пустого старта: <code>%t</code></li>
</ul>
<p>Для <code>stdio</code> режима по-прежнему обычно нужен конкретный <code>service URL</code>, потому что dashboard в этом транспортe не используется.</p>
<h2 id="r3">3. Сценарий работы через dashboard</h2>
<ol>
<li>Откройте <code>%s</code>.</li>
<li>Создайте новое SAP OData подключение: имя соединения, имя системы, URL сервиса, необязательный SAP client, логин и пароль.</li>
<li>Выберите, должен ли bridge отдавать write-capable инструменты для этого профиля.</li>
<li>Нажмите <code>Подключить</code>. Bridge немедленно загрузит metadata и перестроит MCP tools под выбранную SAP систему.</li>
<li>Позже переключайте активную систему через radio-style селектор <code>По умолчанию</code> в левом списке.</li>
<li>Используйте <code>Изменить</code> для обновления сохранённого профиля и <code>Удалить</code> для его удаления.</li>
</ol>
<h2 id="r4">4. Параметры подключения</h2>
<table><tr><th>Поле</th><th>Описание</th></tr>
<tr><td><code>Имя соединения</code></td><td>Пользовательская метка профиля в dashboard.</td></tr>
<tr><td><code>Имя системы</code></td><td>Бизнес-имя SAP-ландшафта, например <code>S4HANA DEV</code>.</td></tr>
<tr><td><code>URL сервиса</code></td><td>Корневой URL SAP OData сервиса, например <code>https://host/sap/opu/odata/sap/API_SALES_ORDER_SRV/</code>.</td></tr>
<tr><td><code>SAP клиент</code></td><td>Необязательное значение <code>sap-client</code>. Если оно задано, dashboard автоматически добавляет его в query string URL.</td></tr>
<tr><td><code>Логин</code></td><td>Имя пользователя basic auth для загрузки metadata и выполнения инструментов.</td></tr>
<tr><td><code>Пароль</code></td><td>Пароль basic auth. При редактировании пустое значение сохраняет ранее сохранённый пароль.</td></tr>
<tr><td><code>Разрешить запись</code></td><td>Если включено, bridge отдаёт модифицирующие инструменты там, где это позволяет metadata. Если выключено, профиль принудительно переводится в read-only режим.</td></tr>
</table>
<h2 id="r5">5. Подключение Codex и других MCP-клиентов</h2>
<p>Когда активная SAP система выбрана в dashboard, MCP-клиенты используют обычный HTTP transport. Текущий маршрут сервера: <code>%s</code>.</p>
<pre><code>codex mcp add sap-odata-universal --url http://localhost:3000%s</code></pre>
<p>Если вы предпочитаете <code>stdio</code> для Codex, используйте отдельную бинарную конфигурацию под одну целевую систему. Dashboard в первую очередь предназначен для HTTP-режима и быстрого переключения систем.</p>
<h2 id="r6">6. Хранение состояния и переключение активной системы</h2>
<p>Сохранённые dashboard-профили лежат в <code>%s</code>. В state file сохраняются:</p>
<ul>
<li>имя активного подключения по умолчанию</li>
<li>сохранённые SAP OData профили</li>
<li>режим доступа для каждого подключения</li>
<li>credential-поля, необходимые для автопереподключения после рестарта</li>
</ul>
<p>На старте dashboard пытается восстановить активный профиль и автоматически перестроить bridge. Текущее активное подключение: <code>%s</code>. Всего сохранённых подключений: <code>%d</code>.</p>
<h2 id="r7">7. HTTP endpoints</h2>
<table><tr><th>Endpoint</th><th>Метод</th><th>Описание</th></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Главный двуязычный dashboard.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Встроенная подробная документация.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Сводка по активному подключению для dashboard JS.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Список сохранённых SAP OData подключений.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать и сразу активировать подключение.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить сохранённое подключение; если оно было активным, bridge переключится на следующее или очистится.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Обновить сохранённый профиль и при необходимости переактивировать его.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Переключить активную SAP систему.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Проверка здоровья transport-слоя.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>MCP transport endpoint.</td></tr>
</table>
<h2 id="r8">8. Замечания по безопасности</h2>
<ul>
<li>Dashboard хранит учётные данные, чтобы переподключаться после рестарта. Защищайте host и права доступа к state file.</li>
<li>HTTP transport по-прежнему нужно закрывать token/TLS, если вы публикуете его вне localhost.</li>
<li>Dashboard скрывает пароль в списке UI, но не шифрует его в сохранённом state file.</li>
<li>Для ландшафтов, где запись недопустима, используйте restricted режим.</li>
</ul>
<h2 id="r9">9. Диагностика проблем</h2>
<ul>
<li>Если подключение падает на <code>Подключить</code>, сначала проверьте корневой OData URL и учётные данные.</li>
<li>Если metadata загружается, но write-tools нет, SAP service metadata может помечать entity sets как non-creatable/non-updatable/non-deletable.</li>
<li>Если приходят данные не из того клиента, явно задайте поле <code>SAP клиент</code>, чтобы dashboard добавил <code>sap-client</code>.</li>
<li>Если bridge стартует без активных tools, проверьте, что сохранённый активный профиль всё ещё проходит аутентификацию.</li>
</ul>
</body></html>`,
		docStyle,
		html.EscapeString(ctx.Transport),
		html.EscapeString(ctx.HTTPAddr),
		html.EscapeString(ctx.MCPPath),
		html.EscapeString(ctx.HealthPath),
		ctx.SupportsEmptyStartup,
		html.EscapeString(ctx.DashboardPath),
		html.EscapeString(ctx.HTTPAddr),
		html.EscapeString(ctx.MCPPath),
		html.EscapeString(ctx.StateFile),
		html.EscapeString(emptyFallback(ctx.ActiveConnection)),
		ctx.TotalConnections,
		html.EscapeString(ctx.DashboardPath),
		html.EscapeString(ctx.DocumentationPath),
		html.EscapeString(ctx.StatusPath),
		html.EscapeString(ctx.ListPath),
		html.EscapeString(ctx.ConnectPath),
		html.EscapeString(ctx.DisconnectPath),
		html.EscapeString(ctx.EditPath),
		html.EscapeString(ctx.SwitchPath),
		html.EscapeString(ctx.HealthPath),
		html.EscapeString(ctx.MCPPath),
	)
}

const docStyle = `body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;background:#0f172a;color:#e2e8f0;padding:20px;max-width:900px;margin:0 auto;line-height:1.6;font-size:.88rem}
a{color:#38bdf8;text-decoration:none}a:hover{text-decoration:underline}
h1{font-size:1.3rem;margin-bottom:4px}h2{font-size:1rem;margin-top:24px;margin-bottom:8px;color:#f8fafc;border-bottom:1px solid #334155;padding-bottom:4px}
h3{font-size:.88rem;margin-top:16px;margin-bottom:4px;color:#cbd5e1}
.sub{color:#64748b;font-size:.78rem}
code{background:#1e293b;padding:1px 5px;border-radius:3px;font-size:.82rem;color:#38bdf8}
pre{background:#1e293b;padding:12px;border-radius:6px;overflow-x:auto;font-size:.8rem;border:1px solid #334155;margin:8px 0}
pre code{background:none;padding:0;color:#e2e8f0}
table{width:100%;border-collapse:collapse;margin:8px 0;font-size:.82rem}
th{text-align:left;padding:6px 8px;border-bottom:1px solid #334155;color:#94a3b8;font-size:.72rem;text-transform:uppercase}
td{padding:6px 8px;border-bottom:1px solid rgba(51,65,85,.4)}
td code{font-size:.78rem}
.back{display:inline-block;margin-bottom:16px;font-size:.82rem}
ul,ol{margin:4px 0 4px 20px}li{margin:2px 0}`

func normalizeLang(lang string) string {
	if strings.EqualFold(strings.TrimSpace(lang), "en") {
		return "en"
	}
	return "ru"
}

func onClass(on bool) string {
	if on {
		return "on"
	}
	return ""
}

func emptyFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}
