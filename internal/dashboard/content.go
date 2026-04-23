package dashboard

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/AlekseiSeleznev/sap-odata-mcp-universal/internal/models"
)

var dashboardTranslations = map[string]map[string]string{
	"ru": {
		"title":            "sap-odata-mcp-universal",
		"subtitle":         "MCP-шлюз для SAP OData",
		"h_tree":           "Системы и сущности",
		"h_system":         "Система SAP",
		"h_services":       "Каталог сервисов",
		"h_entity":         "Сущность",
		"h_operation":      "Операция",
		"btn_docs":         "Документация",
		"btn_refresh":      "Обновить",
		"btn_new_system":   "Новая система",
		"btn_new_entity":   "Новая сущность",
		"btn_new_op":       "Новая операция",
		"btn_new_service":  "Новый сервис",
		"btn_save":         "Сохранить",
		"btn_delete":       "Удалить",
		"btn_activate":     "Сделать активной",
		"btn_discover":     "Обновить метаданные",
		"system_name":      "Имя системы",
		"base_url":         "Базовый URL SAP",
		"sap_client":       "Мандант / SAP клиент",
		"login":            "Логин",
		"password":         "Пароль",
		"allow_write":      "Разрешить запись",
		"service_name":     "Имя сервиса",
		"service_url":      "URL сервиса",
		"entity_label":     "Имя сущности",
		"entity_desc":      "Описание",
		"op_verb":          "Метод",
		"op_service":       "Сервис",
		"op_entityset":     "Entity Set",
		"active":           "Активна",
		"connected":        "Подключена",
		"disconnected":     "Неактивна",
		"rw":               "Чтение/Запись",
		"ro":               "Только чтение",
		"empty_tree":       "Систем пока нет. Создайте первую систему SAP.",
		"empty_services":   "Сервисы ещё не добавлены.",
		"empty_entities":   "Сущности ещё не добавлены.",
		"empty_ops":        "Операции ещё не добавлены.",
		"select_system":    "Сначала выберите систему.",
		"select_entity":    "Сначала выберите сущность.",
		"select_service":   "Выберите сервис",
		"select_entityset": "Выберите entity set",
		"msg_saved":        "Сохранено",
		"msg_deleted":      "Удалено",
		"msg_activated":    "Система активирована",
		"msg_error":        "Ошибка",
		"msg_discovery":    "Метаданные обновлены",
		"confirm_delete":   "Подтвердите удаление",
		"verb_get":         "GET",
		"verb_list":        "LIST",
		"verb_create":      "POST",
		"verb_update":      "PATCH/PUT",
		"verb_delete":      "DELETE",
		"tool_name":        "MCP tool",
		"service_note":     "Сервис используется в операциях сущностей. Один system-profile может включать много сервисов.",
		"footer":           "sap-odata-mcp-universal — GitHub — MIT License",
		"discovery_sets":   "Доступные Entity Sets",
		"discovery_hint":   "После выбора сервиса UI подтягивает metadata и подсказывает entity sets и capabilities.",
	},
	"en": {
		"title":            "sap-odata-mcp-universal",
		"subtitle":         "MCP gateway for SAP OData",
		"h_tree":           "Systems and entities",
		"h_system":         "SAP system",
		"h_services":       "Service catalog",
		"h_entity":         "Entity",
		"h_operation":      "Operation",
		"btn_docs":         "Documentation",
		"btn_refresh":      "Refresh",
		"btn_new_system":   "New system",
		"btn_new_entity":   "New entity",
		"btn_new_op":       "New operation",
		"btn_new_service":  "New service",
		"btn_save":         "Save",
		"btn_delete":       "Delete",
		"btn_activate":     "Make active",
		"btn_discover":     "Refresh metadata",
		"system_name":      "System name",
		"base_url":         "SAP base URL",
		"sap_client":       "Client / mandant",
		"login":            "Login",
		"password":         "Password",
		"allow_write":      "Allow writes",
		"service_name":     "Service name",
		"service_url":      "Service URL",
		"entity_label":     "Entity label",
		"entity_desc":      "Description",
		"op_verb":          "Method",
		"op_service":       "Service",
		"op_entityset":     "Entity set",
		"active":           "Active",
		"connected":        "Connected",
		"disconnected":     "Inactive",
		"rw":               "Read/Write",
		"ro":               "Read-only",
		"empty_tree":       "No systems configured yet. Create the first SAP system.",
		"empty_services":   "No services added yet.",
		"empty_entities":   "No entities added yet.",
		"empty_ops":        "No operations added yet.",
		"select_system":    "Select a system first.",
		"select_entity":    "Select an entity first.",
		"select_service":   "Select a service",
		"select_entityset": "Select an entity set",
		"msg_saved":        "Saved",
		"msg_deleted":      "Deleted",
		"msg_activated":    "System activated",
		"msg_error":        "Error",
		"msg_discovery":    "Metadata refreshed",
		"confirm_delete":   "Confirm deletion",
		"verb_get":         "GET",
		"verb_list":        "LIST",
		"verb_create":      "POST",
		"verb_update":      "PATCH/PUT",
		"verb_delete":      "DELETE",
		"tool_name":        "MCP tool",
		"service_note":     "Services are reused by entity operations. One system profile can aggregate multiple SAP OData services.",
		"footer":           "sap-odata-mcp-universal — GitHub — MIT License",
		"discovery_sets":   "Available entity sets",
		"discovery_hint":   "When you choose a service, the UI loads metadata and suggests entity sets and capabilities.",
	},
}

func renderDashboard(lang string) (string, error) {
	lang = normalizeLang(lang)
	t := dashboardTranslations[lang]
	tJSON, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	body := applyTemplate(dashboardHTML, map[string]string{
		"lang":        lang,
		"subtitle":    t["subtitle"],
		"ru_on":       onClass(lang == "ru"),
		"en_on":       onClass(lang == "en"),
		"btn_docs":    t["btn_docs"],
		"btn_refresh": t["btn_refresh"],
		"h_tree":      t["h_tree"],
		"h_system":    t["h_system"],
		"h_services":  t["h_services"],
		"h_entity":    t["h_entity"],
		"h_operation": t["h_operation"],
		"t_json":      string(tJSON),
		"footer":      t["footer"],
	})
	return body, nil
}

func renderDocs(lang string, ctx *models.DashboardDocumentationContext) string {
	lang = normalizeLang(lang)
	if lang == "en" {
		return renderDocsEN(ctx)
	}
	return renderDocsRU(ctx)
}

func renderDocsRU(ctx *models.DashboardDocumentationContext) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>sap-odata-mcp-universal docs</title><style>%s</style></head><body>
<h1>sap-odata-mcp-universal</h1>
<p>Иерархический dashboard для SAP OData. Здесь конфигурация строится не вокруг одного OData root, а вокруг бизнес-модели <code>System → Entity → Operation</code>.</p>

<h2>1. Что именно настраивается</h2>
<ul>
<li><strong>System</strong>: одна SAP система, её логин/пароль, client/mandant и базовые ограничения на запись.</li>
<li><strong>Service Catalog</strong>: набор OData сервисов, доступных внутри этой системы.</li>
<li><strong>Entity</strong>: бизнес-сущность, например <code>Materials</code> или <code>BusinessPartners</code>.</li>
<li><strong>Operation</strong>: конкретный binding вида <code>GET</code>, <code>LIST</code>, <code>POST</code>, <code>PATCH/PUT</code>, <code>DELETE</code>, который привязывается к выбранному сервису и entity set.</li>
</ul>

<h2>2. Зачем нужна такая модель</h2>
<p>В SAP одна и та же бизнес-сущность часто разбита по разным OData сервисам. Например:</p>
<pre><code>Materials.GET   -> MMIM_MATERIAL_DATA_SRV / MaterialHeaders
Materials.POST  -> API_PRODUCT_SRV / A_Product</code></pre>
<p>Старый flat-подход «одно подключение = один сервис» плохо описывает такой сценарий. Новый dashboard хранит общие credentials на уровне системы и даёт связать разные методы одной сущности с разными сервисами.</p>

<h2>3. Базовый workflow</h2>
<ol>
<li>Создайте систему SAP через <code>%s</code>.</li>
<li>Добавьте в неё OData сервисы через <code>%s</code>.</li>
<li>Создайте сущность через <code>%s</code>.</li>
<li>Для сущности добавьте операции через <code>%s</code>, выбирая сервис и entity set.</li>
<li>Сделайте систему активной через <code>%s</code>. После этого runtime перестроит MCP tools под bindings активной системы.</li>
</ol>

<h2>4. Как runtime это исполняет</h2>
<ul>
<li>При активации системы backend очищает старые business-tools и строит новые tools на основе configured operations.</li>
<li>Для каждого задействованного сервиса metadata кэшируются отдельно.</li>
<li>Каждый MCP tool роутится напрямую в нужный OData service root через <code>client.ODataClient</code>.</li>
<li>CSRF handling, basic auth и CRUD уже выполняются существующим OData client слоем автоматически.</li>
</ul>

<h2>5. HTTP API dashboard</h2>
<table>
<tr><th>Path</th><th>Method</th><th>Назначение</th></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Runtime status активной системы.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Полное дерево систем, сервисов, сущностей и операций.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать или изменить system profile.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить систему.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Сделать систему активной.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать или изменить сервис.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить сервис.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Загрузить metadata выбранного сервиса и список entity sets.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать или изменить сущность.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить сущность.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать или изменить operation binding.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить operation binding.</td></tr>
</table>

<h2>6. MCP endpoint</h2>
<p>Сам MCP endpoint остаётся прежним: <code>%s</code>. Dashboard only управляет тем, какие tools публикуются на этом endpoint для активной системы.</p>

<h2>7. Что попадает в tool names</h2>
<p>Tool name строится из сущности, метода и system id. Пример:</p>
<pre><code>materials_get_for_s4d-100
materials_create_for_s4d-100</code></pre>
<p>Если имя занято, backend добавляет числовой suffix.</p>

<h2>8. Безопасность и хранение state</h2>
<ul>
<li>State file: <code>%s</code></li>
<li>Пароли сохраняются для восстановления активной системы после перезапуска. На диске они не шифруются.</li>
<li>Если HTTP transport открыт наружу, используйте MCP token, TLS и нормальные сетевые ограничения.</li>
<li>В режиме <code>restricted</code> runtime не публикует mutating operations.</li>
</ul>

<h2>9. Практический пример</h2>
<pre><code>System: S4D / client 100
Services:
  materials-read  -> .../MMIM_MATERIAL_DATA_SRV/
  products-write  -> .../API_PRODUCT_SRV/
Entity:
  Materials
Operations:
  GET    -> materials-read / MaterialHeaders
  LIST   -> materials-read / MaterialHeaders
  POST   -> products-write / A_Product
  PATCH  -> products-write / A_Product</code></pre>

<h2>10. Что делать, если operation не сохраняется</h2>
<ul>
<li>Проверьте credentials системы.</li>
<li>Проверьте, что сервис добавлен в Service Catalog той же системы.</li>
<li>Используйте refresh metadata, чтобы убедиться, что entity set реально существует.</li>
<li>Если система <code>restricted</code>, mutating operations сохранятся, но публиковаться не будут до переключения в <code>unrestricted</code>.</li>
</ul>

</body></html>`,
		docStyle,
		html.EscapeString(ctx.SaveSystemPath),
		html.EscapeString(ctx.SaveServicePath),
		html.EscapeString(ctx.SaveEntityPath),
		html.EscapeString(ctx.SaveOperationPath),
		html.EscapeString(ctx.ActivateSystemPath),
		html.EscapeString(ctx.StatusPath),
		html.EscapeString(ctx.SystemsPath),
		html.EscapeString(ctx.SaveSystemPath),
		html.EscapeString(ctx.DeleteSystemPath),
		html.EscapeString(ctx.ActivateSystemPath),
		html.EscapeString(ctx.SaveServicePath),
		html.EscapeString(ctx.DeleteServicePath),
		html.EscapeString(ctx.DiscoveryPath),
		html.EscapeString(ctx.SaveEntityPath),
		html.EscapeString(ctx.DeleteEntityPath),
		html.EscapeString(ctx.SaveOperationPath),
		html.EscapeString(ctx.DeleteOperationPath),
		html.EscapeString(ctx.MCPPath),
		html.EscapeString(ctx.StateFile),
	)
}

func renderDocsEN(ctx *models.DashboardDocumentationContext) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>sap-odata-mcp-universal docs</title><style>%s</style></head><body>
<h1>sap-odata-mcp-universal</h1>
<p>A hierarchical SAP OData dashboard. Configuration is centered around <code>System → Entity → Operation</code> instead of a single flat OData service profile.</p>

<h2>1. Configuration model</h2>
<ul>
<li><strong>System</strong>: SAP landscape, credentials, client/mandant, and default write policy.</li>
<li><strong>Service Catalog</strong>: all OData service roots available inside the selected system.</li>
<li><strong>Entity</strong>: a business concept such as <code>Materials</code>.</li>
<li><strong>Operation</strong>: a concrete business method such as <code>GET</code>, <code>LIST</code>, <code>POST</code>, <code>PATCH/PUT</code>, or <code>DELETE</code> mapped to a service and entity set.</li>
</ul>

<h2>2. Why this exists</h2>
<p>In real SAP landscapes, one business object is frequently split across several OData services. The dashboard keeps credentials at system level and lets one business entity route different methods to different services.</p>

<h2>3. Typical workflow</h2>
<ol>
<li>Create a system profile via <code>%s</code>.</li>
<li>Add one or more services via <code>%s</code>.</li>
<li>Create a business entity via <code>%s</code>.</li>
<li>Add operation bindings via <code>%s</code>.</li>
<li>Activate the system with <code>%s</code>. MCP tools are rebuilt for that active system only.</li>
</ol>

<h2>4. Runtime behavior</h2>
<ul>
<li>The backend clears previous business tools and rebuilds them from configured bindings.</li>
<li>Metadata are cached per service root.</li>
<li>Each tool routes directly to the corresponding OData service through <code>client.ODataClient</code>.</li>
<li>CSRF handling, auth, and CRUD behavior are reused from the existing OData client implementation.</li>
</ul>

<h2>5. Dashboard HTTP API</h2>
<table>
<tr><th>Path</th><th>Method</th><th>Purpose</th></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Runtime status for the active system.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Full hierarchy of systems, services, entities, and operations.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Create or update a system.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Delete a system.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Activate a system.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Create or update a service.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Delete a service.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Load service metadata and entity set suggestions.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Create or update an entity.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Delete an entity.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Create or update an operation binding.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Delete an operation binding.</td></tr>
</table>

<h2>6. MCP endpoint</h2>
<p>The MCP endpoint remains <code>%s</code>. The dashboard only changes which tools are published there for the active SAP system.</p>

<h2>7. Tool naming</h2>
<pre><code>materials_get_for_s4d-100
materials_create_for_s4d-100</code></pre>

<h2>8. State and security</h2>
<ul>
<li>State file: <code>%s</code></li>
<li>Passwords are persisted for restart recovery and are not encrypted on disk.</li>
<li>For remote HTTP exposure, use MCP token, TLS, and proper network boundaries.</li>
<li>In <code>restricted</code> mode, mutating operations are not published as MCP tools.</li>
</ul>

<h2>9. Example</h2>
<pre><code>System: S4D / client 100
Services:
  materials-read  -> .../MMIM_MATERIAL_DATA_SRV/
  products-write  -> .../API_PRODUCT_SRV/
Entity:
  Materials
Operations:
  GET    -> materials-read / MaterialHeaders
  LIST   -> materials-read / MaterialHeaders
  POST   -> products-write / A_Product
  PATCH  -> products-write / A_Product</code></pre>

</body></html>`,
		docStyle,
		html.EscapeString(ctx.SaveSystemPath),
		html.EscapeString(ctx.SaveServicePath),
		html.EscapeString(ctx.SaveEntityPath),
		html.EscapeString(ctx.SaveOperationPath),
		html.EscapeString(ctx.ActivateSystemPath),
		html.EscapeString(ctx.StatusPath),
		html.EscapeString(ctx.SystemsPath),
		html.EscapeString(ctx.SaveSystemPath),
		html.EscapeString(ctx.DeleteSystemPath),
		html.EscapeString(ctx.ActivateSystemPath),
		html.EscapeString(ctx.SaveServicePath),
		html.EscapeString(ctx.DeleteServicePath),
		html.EscapeString(ctx.DiscoveryPath),
		html.EscapeString(ctx.SaveEntityPath),
		html.EscapeString(ctx.DeleteEntityPath),
		html.EscapeString(ctx.SaveOperationPath),
		html.EscapeString(ctx.DeleteOperationPath),
		html.EscapeString(ctx.MCPPath),
		html.EscapeString(ctx.StateFile),
	)
}

func applyTemplate(input string, values map[string]string) string {
	replacements := make([]string, 0, len(values)*2)
	for key, value := range values {
		replacements = append(replacements, "{{"+key+"}}", value)
	}
	return strings.NewReplacer(replacements...).Replace(input)
}

func onClass(enabled bool) string {
	if enabled {
		return "on"
	}
	return ""
}

func normalizeLang(lang string) string {
	if strings.EqualFold(strings.TrimSpace(lang), "en") {
		return "en"
	}
	return "ru"
}

const docStyle = `body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;background:#0f172a;color:#e2e8f0;padding:20px;max-width:980px;margin:0 auto;line-height:1.6;font-size:.92rem}
h1,h2{color:#f8fafc}h1{margin-bottom:14px}h2{margin-top:24px;margin-bottom:10px}
p,ul,ol,table,pre{margin-bottom:14px}ul,ol{padding-left:22px}
code,pre{font-family:'SF Mono','Cascadia Code',monospace}
pre{background:#111827;border:1px solid #334155;border-radius:8px;padding:14px;overflow:auto}
table{width:100%%;border-collapse:collapse}
th,td{border:1px solid #334155;padding:8px;vertical-align:top}
th{background:#1e293b;text-align:left}
a{color:#38bdf8;text-decoration:none}a:hover{text-decoration:underline}`

const dashboardHTML = `<!DOCTYPE html>
<html lang="{{lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>sap-odata-mcp-universal</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;background:#0f172a;color:#f8fafc;height:100vh;display:flex;flex-direction:column;overflow:hidden}
.content{flex:1;overflow-y:auto;padding:20px}
.header{background:#1e293b;border-bottom:1px solid #334155;padding:8px 20px;display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:8px;flex-shrink:0}
.header-left{display:flex;align-items:center;gap:10px}
.header h1{font-size:1.05rem;color:#f8fafc;font-weight:700}
.header .sub{color:#94a3b8;font-size:.75rem}
.header-right{display:flex;align-items:center;gap:6px;flex-wrap:wrap}
.lang-sw{display:flex;border:1px solid #475569;border-radius:5px;overflow:hidden}
.lang-sw a{padding:3px 8px;font-size:.7rem;color:#94a3b8;display:block;text-decoration:none}
.lang-sw a.on{background:#334155;color:#f8fafc}
.btn{display:inline-flex;align-items:center;gap:4px;padding:5px 12px;border-radius:5px;font-size:.78rem;cursor:pointer;border:1px solid #475569;background:#1e293b;color:#94a3b8;text-decoration:none;transition:.15s}
.btn:hover{background:#334155;color:#f8fafc}.btn-d{color:#ef4444;border-color:rgba(239,68,68,.25)}.btn-d:hover{background:rgba(239,68,68,.1);color:#ef4444;border-color:#ef4444}
.card{background:#1e293b;border-radius:8px;padding:12px;border:1px solid #334155;overflow:hidden;margin-bottom:14px}
.card h2{font-size:.65rem;color:#94a3b8;text-transform:uppercase;letter-spacing:.06em;margin-bottom:8px;font-weight:600}
.cols{display:grid;grid-template-columns:1.05fr 1.2fr;gap:14px;align-items:start}
.stack{display:grid;gap:14px}
.tree-item{background:#0f172a;border:1px solid #334155;border-radius:6px;padding:10px 12px;margin-bottom:8px}
.tree-item:last-child,.svc-item:last-child,.op-mini:last-child{margin-bottom:0}
.tree-row{display:flex;align-items:flex-start;gap:10px}
.tree-main{flex:1}
.tree-name{font-weight:600;font-size:.88rem}
.tree-details{color:#94a3b8;font-size:.75rem;font-family:'SF Mono','Cascadia Code',monospace;margin-top:2px}
.entity-node{margin-top:10px;padding-top:10px;border-top:1px solid #1e293b}
.entity-head{display:flex;align-items:center;justify-content:space-between;gap:8px}
.entity-title{font-size:.81rem;font-weight:600;color:#e2e8f0}
.entity-desc{font-size:.72rem;color:#94a3b8;margin-top:2px}
.op-mini{margin-top:6px;padding:7px 8px;border:1px solid #334155;border-radius:6px;background:#111827}
.op-row{display:flex;align-items:center;justify-content:space-between;gap:8px}
.badge{display:inline-flex;align-items:center;padding:1px 6px;border-radius:3px;font-size:.62rem;font-weight:600}
.badge-g{background:rgba(34,197,94,.12);color:#22c55e}.badge-r{background:rgba(239,68,68,.12);color:#ef4444}.badge-b{background:rgba(59,130,246,.12);color:#3b82f6}.badge-c{background:#164e63;color:#22d3ee}
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:8px}.form-group{display:flex;flex-direction:column;gap:3px}.form-group.full{grid-column:1/-1}
.form-group label{font-size:.65rem;color:#94a3b8;text-transform:uppercase;letter-spacing:.04em;font-weight:600}
input,select,textarea{padding:5px 8px;border-radius:4px;border:1px solid #475569;background:#0f172a;color:#e2e8f0;font-size:.8rem;transition:border .15s;width:100%%}
textarea{min-height:76px;resize:vertical}
input:focus,select:focus,textarea:focus{outline:none;border-color:#38bdf8}
.form-actions{display:flex;gap:6px;justify-content:flex-end;margin-top:10px;flex-wrap:wrap}
.toolbar{display:flex;gap:6px;flex-wrap:wrap;margin-bottom:8px}
.svc-item{display:flex;align-items:flex-start;justify-content:space-between;gap:8px;padding:8px 10px;border:1px solid #334155;border-radius:6px;background:#0f172a;margin-bottom:8px}
.svc-main{flex:1;min-width:0}
.empty{text-align:center;padding:20px;color:#64748b;font-size:.82rem}
.toggle{display:flex;align-items:center;gap:8px;cursor:pointer;font-size:.78rem;color:#cbd5e1;user-select:none}.toggle input{display:none}.toggle-track{width:34px;height:18px;border-radius:9px;background:#475569;position:relative;transition:.2s;flex-shrink:0}.toggle-track::after{content:'';width:14px;height:14px;border-radius:50%%;background:#94a3b8;position:absolute;top:2px;left:2px;transition:.2s}.toggle input:checked+.toggle-track{background:#22c55e}.toggle input:checked+.toggle-track::after{left:18px;background:#fff}
.footer{padding:8px 20px;text-align:center;color:#475569;font-size:.68rem;border-top:1px solid #1e293b;flex-shrink:0}.footer a{color:#64748b;text-decoration:none}.footer a:hover{color:#94a3b8}
.toast-msg{position:fixed;top:50%%;left:50%%;transform:translate(-50%%,-50%%);background:#164e63;color:#22d3ee;padding:14px 24px;border-radius:8px;font-size:.9rem;z-index:999;max-width:500px;text-align:center;box-shadow:0 4px 20px rgba(0,0,0,.5);pointer-events:none}.toast-err{background:#7f1d1d;color:#fca5a5}
.hint{font-size:.72rem;color:#94a3b8;margin-top:6px}
@media(max-width:1080px){.cols{grid-template-columns:1fr}.content{padding:12px}}
@media(max-width:600px){.form-grid{grid-template-columns:1fr}}
</style>
</head>
<body>
<div class="header">
  <div class="header-left">
    <div><h1>sap-odata-mcp-universal</h1><span class="sub">{{subtitle}}</span></div>
  </div>
  <div class="header-right">
    <div class="lang-sw">
      <a href="?lang=ru" class="{{ru_on}}">RU</a>
      <a href="?lang=en" class="{{en_on}}">EN</a>
    </div>
    <a class="btn" href="/dashboard/docs?lang={{lang}}" target="_blank">{{btn_docs}}</a>
    <button class="btn" onclick="loadAll()">{{btn_refresh}}</button>
  </div>
</div>
<div class="content">
  <div class="cols">
    <div class="card">
      <h2>{{h_tree}}</h2>
      <div class="toolbar">
        <button class="btn" onclick="newSystem()">{{btn_new_system}}</button>
        <button class="btn" onclick="newEntity()">{{btn_new_entity}}</button>
        <button class="btn" onclick="newOperation()">{{btn_new_op}}</button>
      </div>
      <div id="tree"></div>
    </div>
    <div class="stack">
      <div class="card">
        <h2>{{h_system}}</h2>
        <div class="form-grid">
          <div class="form-group"><label>{{system_name}}</label><input id="sys-name"></div>
          <div class="form-group"><label>{{sap_client}}</label><input id="sys-client" placeholder="100"></div>
          <div class="form-group full"><label>{{base_url}}</label><input id="sys-base" placeholder="http://s4d.msgplaut.com:8000"></div>
          <div class="form-group"><label>{{login}}</label><input id="sys-user" autocomplete="off"></div>
          <div class="form-group"><label>{{password}}</label><input id="sys-pass" type="password" autocomplete="new-password"></div>
          <div class="form-group full"><label class="toggle"><input type="checkbox" id="sys-write"><span class="toggle-track"></span>{{allow_write}}</label></div>
        </div>
        <div class="form-actions">
          <button class="btn" onclick="saveSystem()">{{btn_save}}</button>
          <button class="btn" onclick="activateSystem()">{{btn_activate}}</button>
          <button class="btn btn-d" onclick="deleteSystem()">{{btn_delete}}</button>
        </div>
      </div>

      <div class="card">
        <h2>{{h_services}}</h2>
        <div id="services"></div>
        <div class="hint">{{service_note}}</div>
        <div class="form-grid" style="margin-top:10px">
          <div class="form-group"><label>{{service_name}}</label><input id="svc-name"></div>
          <div class="form-group full"><label>{{service_url}}</label><input id="svc-url" placeholder="https://host/sap/opu/odata/sap/SERVICE_SRV/"></div>
        </div>
        <div class="form-actions">
          <button class="btn" onclick="saveService()">{{btn_save}}</button>
          <button class="btn" onclick="discoverSelectedService()">{{btn_discover}}</button>
          <button class="btn btn-d" onclick="deleteService()">{{btn_delete}}</button>
        </div>
      </div>

      <div class="card">
        <h2>{{h_entity}}</h2>
        <div class="form-grid">
          <div class="form-group"><label>{{entity_label}}</label><input id="ent-label"></div>
          <div class="form-group full"><label>{{entity_desc}}</label><textarea id="ent-desc"></textarea></div>
        </div>
        <div class="form-actions">
          <button class="btn" onclick="saveEntity()">{{btn_save}}</button>
          <button class="btn btn-d" onclick="deleteEntity()">{{btn_delete}}</button>
        </div>
      </div>

      <div class="card">
        <h2>{{h_operation}}</h2>
        <div class="form-grid">
          <div class="form-group"><label>{{op_verb}}</label><select id="op-verb"></select></div>
          <div class="form-group"><label>{{op_service}}</label><select id="op-service"></select></div>
          <div class="form-group full"><label>{{op_entityset}}</label><select id="op-entityset"></select></div>
        </div>
        <div id="discovery" class="hint">{{discovery_hint}}</div>
        <div class="form-actions">
          <button class="btn" onclick="saveOperation()">{{btn_save}}</button>
          <button class="btn" onclick="refreshDiscovery()">{{btn_discover}}</button>
          <button class="btn btn-d" onclick="deleteOperation()">{{btn_delete}}</button>
        </div>
      </div>
    </div>
  </div>
</div>
<div class="footer"><a href="https://github.com/AlekseiSeleznev/sap-odata-mcp-universal" target="_blank">{{footer}}</a></div>
<script>
const T = {{t_json}};
const VERBS = [
  {value:'get', label:T.verb_get},
  {value:'list', label:T.verb_list},
  {value:'create', label:T.verb_create},
  {value:'update', label:T.verb_update},
  {value:'delete', label:T.verb_delete},
];
const state = {systems:[], status:null, selectedSystemId:'', selectedEntityId:'', selectedOperationId:'', editingServiceId:'', discovery:{}};

function toast(msg, isErr) {
  const d = document.createElement('div');
  d.className = 'toast-msg' + (isErr ? ' toast-err' : '');
  d.textContent = msg;
  document.body.appendChild(d);
  setTimeout(() => d.remove(), 3000);
}
async function api(url, opts) {
  const r = await fetch(url, opts || {});
  return r.json();
}
function esc(s) {
  const d = document.createElement('div');
  d.appendChild(document.createTextNode(s || ''));
  return d.innerHTML;
}
function activeSystem() { return state.systems.find(x => x.id === state.selectedSystemId) || null; }
function activeEntity() {
  const sys = activeSystem();
  return sys ? sys.entities.find(x => x.id === state.selectedEntityId) || null : null;
}
function activeOperation() {
  const ent = activeEntity();
  return ent ? ent.operations.find(x => x.id === state.selectedOperationId) || null : null;
}
function syncSelection() {
  if (!state.systems.length) {
    state.selectedSystemId = ''; state.selectedEntityId = ''; state.selectedOperationId = ''; state.editingServiceId = ''; return;
  }
  if (!state.systems.some(x => x.id === state.selectedSystemId)) {
    const active = state.systems.find(x => x.active) || state.systems[0];
    state.selectedSystemId = active.id;
  }
  const sys = activeSystem();
  if (!sys.entities.some(x => x.id === state.selectedEntityId)) {
    state.selectedEntityId = sys.entities[0] ? sys.entities[0].id : '';
  }
  const ent = activeEntity();
  if (!ent || !ent.operations.some(x => x.id === state.selectedOperationId)) {
    state.selectedOperationId = ent && ent.operations[0] ? ent.operations[0].id : '';
  }
  if (state.editingServiceId && !sys.services.some(x => x.id === state.editingServiceId)) state.editingServiceId = '';
}
function modeLabel(mode) { return mode === 'unrestricted' ? T.rw : T.ro; }
function opLabel(verb) {
  const found = VERBS.find(x => x.value === verb);
  return found ? found.label : verb.toUpperCase();
}
function systemDetails(sys) {
  const parts = [];
  if (sys.base_url) parts.push(sys.base_url);
  if (sys.client) parts.push('client=' + sys.client);
  if (sys.username) parts.push(sys.username);
  return parts.join(' | ');
}
async function loadAll() {
  const [systems, status] = await Promise.all([api('/api/systems'), api('/api/status')]);
  state.systems = Array.isArray(systems) ? systems : [];
  state.status = status || null;
  syncSelection();
  renderAll();
}
function renderAll() {
  renderTree(); renderSystemForm(); renderServices(); renderEntityForm(); renderOperationForm();
}
function renderTree() {
  const root = document.getElementById('tree');
  if (!state.systems.length) { root.innerHTML = '<div class="empty">' + T.empty_tree + '</div>'; return; }
  root.innerHTML = state.systems.map(sys => {
    const sysBadges = [
      '<span class="badge ' + (sys.connected ? 'badge-g' : 'badge-r') + '">' + (sys.connected ? T.connected : T.disconnected) + '</span>',
      '<span class="badge ' + (sys.access_mode === 'unrestricted' ? 'badge-b' : 'badge-c') + '">' + modeLabel(sys.access_mode) + '</span>'
    ].join(' ');
    const entities = sys.entities.length ? sys.entities.map(ent => {
      const ops = ent.operations.length ? ent.operations.map(op => '<div class="op-mini" onclick="selectOperation(\'' + esc(op.id) + '\')"><div class="op-row"><div><strong>' + opLabel(op.verb) + '</strong> <span style="color:#94a3b8">' + esc(op.entity_set) + '</span></div><div>' + (op.tool_name ? '<span class="badge badge-c">' + esc(op.tool_name) + '</span>' : '') + '</div></div></div>').join('') : '<div class="hint">' + T.empty_ops + '</div>';
      return '<div class="entity-node">' +
        '<div class="entity-head"><div onclick="selectEntity(\'' + esc(ent.id) + '\')"><div class="entity-title">' + esc(ent.label) + '</div><div class="entity-desc">' + esc(ent.description || '') + '</div></div></div>' +
        ops + '</div>';
    }).join('') : '<div class="hint">' + T.empty_entities + '</div>';
    return '<div class="tree-item">' +
      '<div class="tree-row"><div class="tree-main" onclick="selectSystem(\'' + esc(sys.id) + '\')"><div class="tree-name">' + esc(sys.name) + '</div><div class="tree-details">' + esc(systemDetails(sys)) + '</div><div style="margin-top:4px;display:flex;gap:5px;flex-wrap:wrap">' + sysBadges + (sys.active ? '<span class="badge badge-c">' + T.active + '</span>' : '') + '</div></div></div>' +
      entities + '</div>';
  }).join('');
}
function renderSystemForm() {
  const sys = activeSystem();
  document.getElementById('sys-name').value = sys ? sys.name : '';
  document.getElementById('sys-base').value = sys ? (sys.base_url || '') : '';
  document.getElementById('sys-client').value = sys ? (sys.client || '') : '';
  document.getElementById('sys-user').value = sys ? (sys.username || '') : '';
  document.getElementById('sys-pass').value = '';
  document.getElementById('sys-write').checked = !!(sys && sys.access_mode === 'unrestricted');
}
function renderServices() {
  const sys = activeSystem();
  const root = document.getElementById('services');
  if (!sys) { root.innerHTML = '<div class="empty">' + T.select_system + '</div>'; return; }
  if (!sys.services.length) root.innerHTML = '<div class="empty">' + T.empty_services + '</div>';
  else root.innerHTML = sys.services.map(svc => '<div class="svc-item"><div class="svc-main" onclick="editService(\'' + esc(svc.id) + '\')"><div class="tree-name">' + esc(svc.name) + '</div><div class="tree-details">' + esc(svc.safe_service_url || svc.service_url) + '</div></div><div style="display:flex;gap:6px"><button class="btn" onclick="editService(\'' + esc(svc.id) + '\')">' + T.btn_save + '</button><button class="btn btn-d" onclick="deleteService(\'' + esc(svc.id) + '\')">' + T.btn_delete + '</button></div></div>').join('');
  const editing = sys.services.find(x => x.id === state.editingServiceId) || null;
  document.getElementById('svc-name').value = editing ? editing.name : '';
  document.getElementById('svc-url').value = editing ? editing.service_url : '';
}
function renderEntityForm() {
  const sys = activeSystem(); const ent = activeEntity();
  document.getElementById('ent-label').value = ent ? ent.label : '';
  document.getElementById('ent-desc').value = ent ? (ent.description || '') : '';
  if (!sys) document.getElementById('ent-label').placeholder = T.select_system;
}
function renderOperationForm() {
  const sys = activeSystem(); const ent = activeEntity(); const op = activeOperation();
  const verbSel = document.getElementById('op-verb');
  verbSel.innerHTML = VERBS.map(v => '<option value="' + v.value + '">' + v.label + '</option>').join('');
  const serviceSel = document.getElementById('op-service');
  if (!sys) {
    serviceSel.innerHTML = '<option value="">' + T.select_system + '</option>';
    document.getElementById('op-entityset').innerHTML = '<option value="">' + T.select_service + '</option>';
    return;
  }
  serviceSel.innerHTML = '<option value="">' + T.select_service + '</option>' + sys.services.map(s => '<option value="' + esc(s.id) + '">' + esc(s.name) + '</option>').join('');
  if (op) {
    verbSel.value = op.verb;
    serviceSel.value = op.service_id;
  } else if (sys.services[0]) {
    serviceSel.value = sys.services[0].id;
  }
  if (serviceSel.value && !state.discovery[state.selectedSystemId + '::' + serviceSel.value]) {
    discoverService(state.selectedSystemId, serviceSel.value).then(() => renderOperationForm());
  }
  populateEntitySetOptions(serviceSel.value, op ? op.entity_set : '');
}
function populateEntitySetOptions(serviceId, selected) {
  const sel = document.getElementById('op-entityset');
  const key = state.selectedSystemId + '::' + serviceId;
  const discovery = state.discovery[key];
  const options = discovery && discovery.entity_sets ? discovery.entity_sets : [];
  if (!serviceId) { sel.innerHTML = '<option value="">' + T.select_service + '</option>'; return; }
  sel.innerHTML = '<option value="">' + T.select_entityset + '</option>' + options.map(x => '<option value="' + esc(x.name) + '">' + esc(x.name) + '</option>').join('');
  if (selected) sel.value = selected;
  document.getElementById('discovery').innerHTML = options.length ? '<strong>' + T.discovery_sets + ':</strong> ' + options.map(x => esc(x.name)).join(', ') : T.discovery_hint;
}
function selectSystem(id) { state.selectedSystemId = id; state.selectedEntityId = ''; state.selectedOperationId = ''; state.editingServiceId = ''; syncSelection(); renderAll(); }
function selectEntity(id) { state.selectedEntityId = id; state.selectedOperationId = ''; syncSelection(); renderAll(); }
function selectOperation(id) { state.selectedOperationId = id; syncSelection(); renderAll(); }
function newSystem() { state.selectedSystemId = ''; state.selectedEntityId = ''; state.selectedOperationId = ''; state.editingServiceId = ''; renderAll(); }
function newEntity() { if (!activeSystem()) { toast(T.select_system, true); return; } state.selectedEntityId = ''; state.selectedOperationId = ''; renderAll(); }
function newOperation() { if (!activeEntity()) { toast(T.select_entity, true); return; } state.selectedOperationId = ''; renderAll(); }
function editService(id) { state.editingServiceId = id; renderServices(); }
async function saveSystem() {
  const payload = { old_id: activeSystem() ? activeSystem().id : '', name: document.getElementById('sys-name').value.trim(), base_url: document.getElementById('sys-base').value.trim(), client: document.getElementById('sys-client').value.trim(), username: document.getElementById('sys-user').value.trim(), password: document.getElementById('sys-pass').value, access_mode: document.getElementById('sys-write').checked ? 'unrestricted' : 'restricted' };
  const r = await api('/api/system/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_saved); await loadAll();
}
async function activateSystem() {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const r = await api('/api/system/activate', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_activated); await loadAll();
}
async function deleteSystem() {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  if (!confirm(T.confirm_delete + ': ' + sys.name)) return;
  const r = await api('/api/system/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({id: sys.id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_deleted); state.selectedSystemId=''; await loadAll();
}
async function saveService() {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const payload = {system_id: sys.id, old_id: state.editingServiceId || '', name: document.getElementById('svc-name').value.trim(), service_url: document.getElementById('svc-url').value.trim()};
  const r = await api('/api/service/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_saved); state.editingServiceId=''; await loadAll();
}
async function deleteService(id) {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const target = id || state.editingServiceId; if (!target) return;
  if (!confirm(T.confirm_delete)) return;
  const r = await api('/api/service/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id, service_id: target})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_deleted); state.editingServiceId=''; await loadAll();
}
async function discoverSelectedService() {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const id = state.editingServiceId || document.getElementById('op-service').value;
  if (!id) return toast(T.select_service, true);
  await discoverService(sys.id, id); toast(T.msg_discovery); renderOperationForm();
}
async function refreshDiscovery() {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const serviceId = document.getElementById('op-service').value;
  if (!serviceId) return toast(T.select_service, true);
  await discoverService(sys.id, serviceId); renderOperationForm(); toast(T.msg_discovery);
}
async function discoverService(systemId, serviceId) {
  const data = await api('/api/service/discover?system_id=' + encodeURIComponent(systemId) + '&service_id=' + encodeURIComponent(serviceId));
  if (data && !data.error) state.discovery[systemId + '::' + serviceId] = data;
  else toast(T.msg_error + ': ' + ((data && data.error) || 'unknown'), true);
}
document.getElementById('op-service').addEventListener('change', async function() {
  const sys = activeSystem(); if (!sys || !this.value) return populateEntitySetOptions('', '');
  if (!state.discovery[sys.id + '::' + this.value]) await discoverService(sys.id, this.value);
  populateEntitySetOptions(this.value, '');
});
async function saveEntity() {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const ent = activeEntity();
  const payload = {system_id: sys.id, old_id: ent ? ent.id : '', label: document.getElementById('ent-label').value.trim(), description: document.getElementById('ent-desc').value.trim()};
  const r = await api('/api/entity/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_saved); await loadAll();
}
async function deleteEntity() {
  const sys = activeSystem(); const ent = activeEntity(); if (!sys || !ent) return toast(T.select_entity, true);
  if (!confirm(T.confirm_delete + ': ' + ent.label)) return;
  const r = await api('/api/entity/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id, entity_id: ent.id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_deleted); state.selectedEntityId=''; await loadAll();
}
async function saveOperation() {
  const sys = activeSystem(); const ent = activeEntity(); if (!sys || !ent) return toast(T.select_entity, true);
  const op = activeOperation();
  const payload = {system_id: sys.id, entity_id: ent.id, old_id: op ? op.id : '', verb: document.getElementById('op-verb').value, service_id: document.getElementById('op-service').value, entity_set: document.getElementById('op-entityset').value, mode: 'generated', enabled: true};
  const r = await api('/api/operation/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_saved); await loadAll();
}
async function deleteOperation() {
  const sys = activeSystem(); const ent = activeEntity(); const op = activeOperation();
  if (!sys || !ent || !op) return toast(T.empty_ops, true);
  if (!confirm(T.confirm_delete + ': ' + opLabel(op.verb))) return;
  const r = await api('/api/operation/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id, entity_id: ent.id, operation_id: op.id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_deleted); state.selectedOperationId=''; await loadAll();
}
loadAll();
</script>
</body></html>`
