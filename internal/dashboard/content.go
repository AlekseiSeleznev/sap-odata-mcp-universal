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
		"title":             "sap-odata-mcp-universal",
		"subtitle":          "MCP-шлюз для SAP OData",
		"h_tree":            "Системы и объекты",
		"h_system":          "Система SAP",
		"h_services":        "Каталог сервисов",
		"h_entity":          "Объект",
		"h_operation":       "Операция",
		"h_edit_system":     "Редактирование системы",
		"h_edit_service":    "Редактирование сервиса",
		"h_edit_entity":     "Редактирование объекта",
		"h_edit_operation":  "Редактирование операции",
		"h_editor":          "Редактор",
		"btn_docs":          "Документация",
		"btn_refresh":       "Обновить",
		"btn_new_system":    "Новая система",
		"btn_new_entity":    "Новый объект",
		"btn_new_op":        "Новая операция",
		"btn_new_service":   "Новый сервис",
		"btn_services":      "Сервисы",
		"btn_test":          "Проверить",
		"btn_edit":          "Изменить",
		"btn_cancel":        "Отмена",
		"btn_save":          "Сохранить",
		"btn_delete":        "Удалить",
		"btn_activate":      "Сделать активной",
		"btn_discover":      "Обновить метаданные",
		"system_name":       "Имя системы",
		"base_url":          "Базовый URL SAP",
		"sap_client":        "Мандант / SAP клиент",
		"login":             "Логин",
		"password":          "Пароль",
		"allow_write":       "Разрешить запись",
		"service_name":      "Имя сервиса",
		"service_url":       "URL сервиса",
		"entity_label":      "Имя объекта",
		"entity_desc":       "Описание",
		"op_name":           "Имя операции",
		"op_verb":           "Тип операции",
		"op_service":        "Сервис",
		"op_entityset":      "Entity Set",
		"op_query_expand":   "По умолчанию $expand",
		"op_query_select":   "По умолчанию $select",
		"op_query_filter":   "По умолчанию $filter",
		"op_query_orderby":  "По умолчанию $orderby",
		"op_query_top":      "По умолчанию $top",
		"active":            "Активна",
		"connected":         "Подключена",
		"disconnected":      "Неактивна",
		"rw":                "Чтение/Запись",
		"ro":                "Только чтение",
		"empty_tree":        "Систем пока нет. Создайте первую систему SAP.",
		"empty_services":    "Сервисы ещё не добавлены.",
		"empty_entities":    "Объекты ещё не добавлены.",
		"empty_ops":         "Операции ещё не добавлены.",
		"select_system":     "Сначала выберите систему.",
		"select_entity":     "Сначала выберите объект.",
		"select_service":    "Выберите сервис",
		"select_entityset":  "Выберите entity set",
		"editor_empty":      "Правая панель используется для добавления. Для редактирования, проверки и управления сервисами используйте кнопки слева.",
		"msg_saved":         "Сохранено",
		"msg_deleted":       "Удалено",
		"msg_activated":     "Система активирована",
		"msg_test_ok":       "Подключение проверено",
		"msg_test_fail":     "Проверка подключения не удалась",
		"msg_error":         "Ошибка",
		"msg_auth_required": "Нужен MCP token для доступа к dashboard API",
		"auth_title":        "MCP token",
		"auth_hint":         "Введите токен, с которым запущен sap-odata-mcp-universal. Токен хранится только в sessionStorage браузера.",
		"auth_token":        "Токен",
		"btn_unlock":        "Подключить",
		"btn_forget_token":  "Сбросить токен",
		"msg_discovery":     "Метаданные обновлены",
		"confirm_delete":    "Подтвердите удаление",
		"verb_get":          "GET запись",
		"verb_list":         "GET список",
		"verb_create":       "POST",
		"verb_update":       "PATCH/PUT",
		"verb_delete":       "DELETE",
		"tool_name":         "MCP tool",
		"service_note":      "Сервис используется в операциях объектов. Один system-profile может включать много сервисов.",
		"h_manage_services": "Сервисы системы",
		"footer_title":      "sap-odata-mcp-universal",
		"footer_github":     "GitHub",
		"footer_license":    "MIT License",
		"discovery_sets":    "Доступные Entity Sets",
		"discovery_hint":    "После выбора сервиса UI подтягивает metadata и подсказывает entity sets и capabilities.",
	},
	"en": {
		"title":             "sap-odata-mcp-universal",
		"subtitle":          "MCP gateway for SAP OData",
		"h_tree":            "Systems and objects",
		"h_system":          "SAP system",
		"h_services":        "Service catalog",
		"h_entity":          "Object",
		"h_operation":       "Operation",
		"h_edit_system":     "Edit system",
		"h_edit_service":    "Edit service",
		"h_edit_entity":     "Edit object",
		"h_edit_operation":  "Edit operation",
		"h_editor":          "Editor",
		"btn_docs":          "Documentation",
		"btn_refresh":       "Refresh",
		"btn_new_system":    "New system",
		"btn_new_entity":    "New object",
		"btn_new_op":        "New operation",
		"btn_new_service":   "New service",
		"btn_services":      "Services",
		"btn_test":          "Test",
		"btn_edit":          "Edit",
		"btn_cancel":        "Cancel",
		"btn_save":          "Save",
		"btn_delete":        "Delete",
		"btn_activate":      "Make active",
		"btn_discover":      "Refresh metadata",
		"system_name":       "System name",
		"base_url":          "SAP base URL",
		"sap_client":        "Client / mandant",
		"login":             "Login",
		"password":          "Password",
		"allow_write":       "Allow writes",
		"service_name":      "Service name",
		"service_url":       "Service URL",
		"entity_label":      "Object name",
		"entity_desc":       "Description",
		"op_name":           "Operation name",
		"op_verb":           "Operation type",
		"op_service":        "Service",
		"op_entityset":      "Entity set",
		"op_query_expand":   "Default $expand",
		"op_query_select":   "Default $select",
		"op_query_filter":   "Default $filter",
		"op_query_orderby":  "Default $orderby",
		"op_query_top":      "Default $top",
		"active":            "Active",
		"connected":         "Connected",
		"disconnected":      "Inactive",
		"rw":                "Read/Write",
		"ro":                "Read-only",
		"empty_tree":        "No systems configured yet. Create the first SAP system.",
		"empty_services":    "No services added yet.",
		"empty_entities":    "No objects added yet.",
		"empty_ops":         "No operations added yet.",
		"select_system":     "Select a system first.",
		"select_entity":     "Select an object first.",
		"select_service":    "Select a service",
		"select_entityset":  "Select an entity set",
		"editor_empty":      "The right pane is used for creation only. Use the buttons on the left for editing, testing, and service management.",
		"msg_saved":         "Saved",
		"msg_deleted":       "Deleted",
		"msg_activated":     "System activated",
		"msg_test_ok":       "Connection verified",
		"msg_test_fail":     "Connection test failed",
		"msg_error":         "Error",
		"msg_auth_required": "MCP token is required for dashboard API access",
		"auth_title":        "MCP token",
		"auth_hint":         "Enter the token used to start sap-odata-mcp-universal. It is stored only in browser sessionStorage.",
		"auth_token":        "Token",
		"btn_unlock":        "Unlock",
		"btn_forget_token":  "Forget token",
		"msg_discovery":     "Metadata refreshed",
		"confirm_delete":    "Confirm deletion",
		"verb_get":          "GET single",
		"verb_list":         "GET list",
		"verb_create":       "POST",
		"verb_update":       "PATCH/PUT",
		"verb_delete":       "DELETE",
		"tool_name":         "MCP tool",
		"service_note":      "Services are reused by object operations. One system profile can aggregate multiple SAP OData services.",
		"h_manage_services": "System services",
		"footer_title":      "sap-odata-mcp-universal",
		"footer_github":     "GitHub",
		"footer_license":    "MIT License",
		"discovery_sets":    "Available entity sets",
		"discovery_hint":    "When you choose a service, the UI loads metadata and suggests entity sets and capabilities.",
	},
}

func renderDashboard(lang string) (string, error) {
	lang = normalizeLang(lang)
	t := dashboardTranslations[lang]
	tJSON, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	values := make(map[string]string, len(t)+5)
	for key, value := range t {
		values[key] = value
	}
	values["lang"] = lang
	values["ru_on"] = onClass(lang == "ru")
	values["en_on"] = onClass(lang == "en")
	values["t_json"] = string(tJSON)

	body := applyTemplate(dashboardHTML, values)
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
<p>Иерархический dashboard для SAP OData. Здесь конфигурация строится не вокруг одного OData root, а вокруг бизнес-модели <code>System → Object → Operation</code>.</p>

<h2>1. Что именно настраивается</h2>
<ul>
<li><strong>System</strong>: одна SAP система, её логин/пароль, client/mandant и базовые ограничения на запись.</li>
<li><strong>Service Catalog</strong>: набор OData сервисов, доступных внутри этой системы.</li>
<li><strong>Object</strong>: бизнес-объект, например <code>Materials</code> или <code>BusinessPartners</code>.</li>
<li><strong>Operation</strong>: конкретный binding вида <code>GET</code>, <code>LIST</code>, <code>POST</code>, <code>PATCH/PUT</code>, <code>DELETE</code>, который привязывается к выбранному сервису и entity set.</li>
</ul>

<h2>2. Зачем нужна такая модель</h2>
<p>В SAP один и тот же бизнес-объект часто разбит по разным OData сервисам. Например:</p>
<pre><code>Materials.GET   -> MMIM_MATERIAL_DATA_SRV / MaterialHeaders
Materials.POST  -> API_PRODUCT_SRV / A_Product</code></pre>
<p>Старый flat-подход «одно подключение = один сервис» плохо описывает такой сценарий. Новый dashboard хранит общие credentials на уровне системы и даёт связать разные методы одного объекта с разными сервисами.</p>

<h2>3. Базовый workflow</h2>
<ol>
<li>Создайте систему SAP через <code>%s</code>.</li>
<li>Добавьте в неё OData сервисы через <code>%s</code>.</li>
<li>Создайте объект через <code>%s</code>.</li>
<li>Для объекта добавьте операции через <code>%s</code>, выбирая сервис и entity set.</li>
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
<tr><td><code>%s</code></td><td>GET</td><td>Полное дерево систем, сервисов, объектов и операций.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать или изменить system profile.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить систему.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Сделать систему активной.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать или изменить сервис.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить сервис.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Загрузить metadata выбранного сервиса и список entity sets.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать или изменить объект.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить объект.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Создать или изменить operation binding.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Удалить operation binding.</td></tr>
</table>

<h2>6. MCP endpoint</h2>
<p>Сам MCP endpoint остаётся прежним: <code>%s</code>. Dashboard управляет тем, какие tools публикуются на этом endpoint для активной системы.</p>

<h2>7. Что попадает в tool names</h2>
<p>Tool name строится из объекта, метода и system id. Пример:</p>
<pre><code>materials_get_for_s4d-100
materials_create_for_s4d-100</code></pre>
<p>Если имя занято, backend добавляет числовой suffix.</p>

<h2>8. Безопасность и хранение state</h2>
<ul>
<li>State file: <code>%s</code></li>
<li>Пароли сохраняются для восстановления активной системы после перезапуска. На диске они не шифруются.</li>
<li>Если сервер запущен с <code>--mcp-token</code>, MCP маршруты <code>/mcp</code>, <code>/rpc</code> и <code>/sse</code> требуют token на каждом запросе.</li>
<li>Dashboard API <code>/api/*</code> доступен без token только с той же машины через loopback (<code>localhost</code>, <code>127.0.0.1</code>, <code>::1</code>), чтобы локальный браузер не требовал ручного ввода token.</li>
<li>При удалённом доступе dashboard принимает token через модальное окно или query-параметр <code>?token=...</code>, хранит его только в browser <code>sessionStorage</code> и отправляет в <code>Authorization: Bearer ...</code>.</li>
<li>Если HTTP transport открыт наружу, используйте MCP token, TLS и нормальные сетевые ограничения.</li>
<li>В режиме <code>restricted</code> runtime оставляет tools доступными, но возвращает ошибку при попытке выполнить mutating operation.</li>
</ul>

<h2>9. Практический пример</h2>
<pre><code>System: S4D / client 100
Services:
  materials-read  -> .../MMIM_MATERIAL_DATA_SRV/
  products-write  -> .../API_PRODUCT_SRV/
Object:
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
<li>Если система <code>restricted</code>, mutating operations сохранятся, но выполнение записи будет отклонено до переключения в <code>unrestricted</code>.</li>
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
<p>A hierarchical SAP OData dashboard. Configuration is centered around <code>System → Object → Operation</code> instead of a single flat OData service profile.</p>

<h2>1. Configuration model</h2>
<ul>
<li><strong>System</strong>: SAP landscape, credentials, client/mandant, and default write policy.</li>
<li><strong>Service Catalog</strong>: all OData service roots available inside the selected system.</li>
<li><strong>Object</strong>: a business concept such as <code>Materials</code>.</li>
<li><strong>Operation</strong>: a concrete business method such as <code>GET</code>, <code>LIST</code>, <code>POST</code>, <code>PATCH/PUT</code>, or <code>DELETE</code> mapped to a service and entity set.</li>
</ul>

<h2>2. Why this exists</h2>
<p>In real SAP landscapes, one business object is frequently split across several OData services. The dashboard keeps credentials at system level and lets one business object route different methods to different services.</p>

<h2>3. Typical workflow</h2>
<ol>
<li>Create a system profile via <code>%s</code>.</li>
<li>Add one or more services via <code>%s</code>.</li>
<li>Create a business object via <code>%s</code>.</li>
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
<tr><td><code>%s</code></td><td>GET</td><td>Full hierarchy of systems, services, objects, and operations.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Create or update a system.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Delete a system.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Activate a system.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Create or update a service.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Delete a service.</td></tr>
<tr><td><code>%s</code></td><td>GET</td><td>Load service metadata and entity set suggestions.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Create or update an object.</td></tr>
<tr><td><code>%s</code></td><td>POST</td><td>Delete an object.</td></tr>
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
<li>When the server runs with <code>--mcp-token</code>, MCP routes <code>/mcp</code>, <code>/rpc</code>, and <code>/sse</code> require a token on every request.</li>
<li>The dashboard API <code>/api/*</code> is available without a token only from the same machine over loopback (<code>localhost</code>, <code>127.0.0.1</code>, <code>::1</code>) so the local browser does not require manual token entry.</li>
<li>For remote access, the dashboard accepts the token via a modal dialog or <code>?token=...</code>, stores it only in browser <code>sessionStorage</code>, and sends <code>Authorization: Bearer ...</code>.</li>
<li>For remote HTTP exposure, use MCP token, TLS, and proper network boundaries.</li>
<li>In <code>restricted</code> mode, mutating tools remain visible but return an explicit error when a write is attempted.</li>
</ul>

<h2>9. Example</h2>
<pre><code>System: S4D / client 100
Services:
  materials-read  -> .../MMIM_MATERIAL_DATA_SRV/
  products-write  -> .../API_PRODUCT_SRV/
Object:
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
table{width:100%;border-collapse:collapse}
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
.btn-s{padding:3px 8px;font-size:.72rem}
.card{background:#1e293b;border-radius:8px;padding:12px;border:1px solid #334155;overflow:hidden;margin-bottom:14px}
.card h2{font-size:.65rem;color:#94a3b8;text-transform:uppercase;letter-spacing:.06em;margin-bottom:8px;font-weight:600}
.cols{display:grid;grid-template-columns:1.05fr 1.2fr;gap:14px;align-items:start}
.stack{display:grid;gap:14px}
.tree-item{background:#0f172a;border:1px solid #334155;border-radius:6px;padding:10px 12px;margin-bottom:8px}
.tree-item:last-child,.svc-item:last-child,.op-mini:last-child{margin-bottom:0}
.tree-row{display:flex;align-items:flex-start;gap:10px}
.tree-main{flex:1}
.tree-actions{display:flex;gap:6px;flex-wrap:wrap}
.tree-name{font-weight:600;font-size:.88rem}
.tree-details{color:#94a3b8;font-size:.75rem;font-family:'SF Mono','Cascadia Code',monospace;margin-top:2px}
.entity-node{margin-top:10px;padding-top:10px;border-top:1px solid #1e293b}
.entity-head{display:flex;align-items:center;justify-content:space-between;gap:8px}
.entity-title{font-size:.81rem;font-weight:600;color:#e2e8f0}
.entity-desc{font-size:.72rem;color:#94a3b8;margin-top:2px}
.op-mini{margin-top:6px;padding:7px 8px;border:1px solid #334155;border-radius:6px;background:#111827}
.op-row{display:flex;align-items:center;justify-content:space-between;gap:8px}
.op-query{margin-top:5px;display:flex;gap:5px;flex-wrap:wrap}
.badge{display:inline-flex;align-items:center;padding:1px 6px;border-radius:3px;font-size:.62rem;font-weight:600}
.badge-g{background:rgba(34,197,94,.12);color:#22c55e}.badge-r{background:rgba(239,68,68,.12);color:#ef4444}.badge-b{background:rgba(59,130,246,.12);color:#3b82f6}.badge-c{background:#164e63;color:#22d3ee}
.badge-q{background:rgba(14,165,233,.12);color:#7dd3fc;border:1px solid rgba(125,211,252,.16);font-family:'SF Mono','Cascadia Code',monospace}
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:8px}.form-group{display:flex;flex-direction:column;gap:3px}.form-group.full{grid-column:1/-1}
.form-group label{font-size:.65rem;color:#94a3b8;text-transform:uppercase;letter-spacing:.04em;font-weight:600}
input,select,textarea{padding:5px 8px;border-radius:4px;border:1px solid #475569;background:#0f172a;color:#e2e8f0;font-size:.8rem;transition:border .15s;width:100%}
textarea{min-height:76px;resize:vertical}
input:focus,select:focus,textarea:focus{outline:none;border-color:#38bdf8}
.form-actions{display:flex;gap:6px;justify-content:flex-end;margin-top:10px;flex-wrap:wrap}
.toolbar{display:flex;gap:6px;flex-wrap:wrap;margin-bottom:8px}
.svc-item{display:flex;align-items:flex-start;justify-content:space-between;gap:8px;padding:8px 10px;border:1px solid #334155;border-radius:6px;background:#0f172a;margin-bottom:8px}
.svc-main{flex:1;min-width:0}
.empty{text-align:center;padding:20px;color:#64748b;font-size:.82rem}
.toggle{display:flex;align-items:center;gap:8px;cursor:pointer;font-size:.78rem;color:#cbd5e1;user-select:none}
.form-group label.toggle{font-size:.78rem;color:#cbd5e1;text-transform:uppercase;letter-spacing:.04em;font-weight:600}
.toggle input{display:none}
.toggle-track{width:34px;height:18px;border-radius:9px;background:#475569;position:relative;transition:.2s;flex-shrink:0}
.toggle-track::after{content:'';width:14px;height:14px;border-radius:50%;background:#fff;position:absolute;top:2px;left:2px;transition:.2s}
.toggle input:checked+.toggle-track{background:#22c55e}
.toggle input:checked+.toggle-track::after{left:18px;background:#fff}
.overlay{position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:100;display:flex;align-items:center;justify-content:center}
.modal{background:#1e293b;border:1px solid #334155;border-radius:10px;padding:20px;width:520px;max-width:92%}
.modal h3{font-size:.88rem;margin-bottom:14px;color:#f8fafc}
.modal-actions{display:flex;gap:6px;justify-content:flex-end;margin-top:14px}
.token-modal{width:440px}
.footer{padding:8px 20px;text-align:center;color:#475569;font-size:.68rem;border-top:1px solid #1e293b;flex-shrink:0}.footer a{color:#64748b;text-decoration:none}.footer a:hover{color:#94a3b8}
.toast-msg{position:fixed;top:50%;left:50%;transform:translate(-50%,-50%);background:#164e63;color:#22d3ee;padding:14px 24px;border-radius:8px;font-size:.9rem;z-index:999;max-width:500px;text-align:center;box-shadow:0 4px 20px rgba(0,0,0,.5);pointer-events:none}.toast-err{background:#7f1d1d;color:#fca5a5}
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
    <button class="btn" onclick="forgetToken()">{{btn_forget_token}}</button>
    <button class="btn" onclick="loadAll()">{{btn_refresh}}</button>
  </div>
</div>
<div class="content">
  <div class="cols">
    <div class="card">
      <h2>{{h_tree}}</h2>
      <div class="toolbar">
        <button class="btn" onclick="newSystem()">{{btn_new_system}}</button>
      </div>
      <div id="tree"></div>
    </div>
    <div class="stack">
      <div id="editor-pane" class="card"></div>
    </div>
  </div>
</div>
<div class="footer">{{footer_title}} — <a href="https://github.com/AlekseiSeleznev/sap-odata-mcp-universal" target="_blank">{{footer_github}}</a> — <a href="https://github.com/AlekseiSeleznev/sap-odata-mcp-universal/blob/main/LICENSE" target="_blank">{{footer_license}}</a></div>
<script>
const T = {{t_json}};
const AUTH_TOKEN_KEY = 'sap-odata-mcp-universal-token';
const VERBS = [
  {value:'get', label:T.verb_get},
  {value:'list', label:T.verb_list},
  {value:'create', label:T.verb_create},
  {value:'update', label:T.verb_update},
  {value:'delete', label:T.verb_delete},
];
const state = {systems:[], status:null, selectedSystemId:'', selectedEntityId:'', selectedOperationId:'', editingServiceId:'', discovery:{}, editor:{kind:'', mode:'create'}};
const initialToken = new URLSearchParams(window.location.search).get('token') || '';
if (initialToken) {
  sessionStorage.setItem(AUTH_TOKEN_KEY, initialToken);
  const cleanURL = new URL(window.location.href);
  cleanURL.searchParams.delete('token');
  window.history.replaceState({}, '', cleanURL.toString());
}

function toast(msg, isErr) {
  const d = document.createElement('div');
  d.className = 'toast-msg' + (isErr ? ' toast-err' : '');
  d.textContent = msg;
  document.body.appendChild(d);
  setTimeout(() => d.remove(), 3000);
}
async function api(url, opts) {
  const options = opts ? {...opts} : {};
  const headers = new Headers(options.headers || {});
  const token = sessionStorage.getItem(AUTH_TOKEN_KEY) || '';
  if (token && !headers.has('Authorization')) headers.set('Authorization', 'Bearer ' + token);
  options.headers = headers;
  try {
    const r = await fetch(url, options);
    const contentType = r.headers.get('content-type') || '';
    const payload = contentType.includes('application/json') ? await r.json() : {error: await r.text()};
    if (r.status === 401) {
      openTokenDialog();
      return {...payload, status: r.status, error: payload.error || T.msg_auth_required};
    }
    if (!r.ok) return {...payload, status: r.status, error: payload.error || r.statusText};
    return payload;
  } catch (err) {
    return {error: err && err.message ? err.message : String(err)};
  }
}
function openTokenDialog() {
  if (document.getElementById('token-modal')) return;
  openOverlay(
    '<div class="modal token-modal" id="token-modal">' +
      '<h3>' + T.auth_title + '</h3>' +
      '<p class="hint" style="margin-bottom:12px">' + T.auth_hint + '</p>' +
      '<div class="form-group"><label>' + T.auth_token + '</label><input id="auth-token" type="password" autocomplete="off"></div>' +
      '<div class="modal-actions">' +
        '<button class="btn" onclick="closeOverlay()">' + T.btn_cancel + '</button>' +
        '<button class="btn" onclick="saveTokenAndReload()">' + T.btn_unlock + '</button>' +
      '</div>' +
    '</div>'
  );
}
function saveTokenAndReload() {
  const input = document.getElementById('auth-token');
  const token = input ? input.value.trim() : '';
  if (!token) return toast(T.msg_auth_required, true);
  sessionStorage.setItem(AUTH_TOKEN_KEY, token);
  closeOverlay();
  loadAll();
}
function forgetToken() {
  sessionStorage.removeItem(AUTH_TOKEN_KEY);
  openTokenDialog();
}
function esc(s) {
  const d = document.createElement('div');
  d.appendChild(document.createTextNode(s || ''));
  return d.innerHTML;
}
function escAttr(s) { return esc(s).replace(/"/g, '&quot;').replace(/'/g, '&#39;'); }
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
    state.selectedSystemId = ''; state.selectedEntityId = ''; state.selectedOperationId = ''; state.editingServiceId = ''; state.editor = {kind:'', mode:'create'}; return;
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
function operationTitle(op) {
  return op && op.name ? op.name : (op ? (opLabel(op.verb) + ' ' + op.entity_set) : '');
}
function suggestOperationName(verb, entitySet) {
  const label = opLabel(verb || 'list');
  return (label + (entitySet ? ' ' + entitySet : '')).trim();
}
function operationQueryBadges(query) {
  query = query || {};
  const keys = ['$expand', '$select', '$filter', '$orderby', '$top', '$skip', '$count'];
  const items = [];
  for (const key of keys) {
    if (query[key]) items.push('<span class="badge badge-q">' + esc(key + '=' + query[key]) + '</span>');
  }
  return items.length ? '<div class="op-query">' + items.join('') + '</div>' : '';
}
function fillOperationQuery(prefix, query) {
  query = query || {};
  const fields = {
    '$expand': 'expand',
    '$select': 'select',
    '$filter': 'filter',
    '$orderby': 'orderby',
    '$top': 'top'
  };
  for (const key in fields) {
    const el = document.getElementById(prefix + fields[key]);
    if (el) el.value = query[key] || '';
  }
}
function collectOperationQuery(prefix) {
  const fields = [
    ['$expand', 'expand'],
    ['$select', 'select'],
    ['$filter', 'filter'],
    ['$orderby', 'orderby'],
    ['$top', 'top']
  ];
  const query = {};
  for (const pair of fields) {
    const el = document.getElementById(prefix + pair[1]);
    const value = el ? el.value.trim() : '';
    if (value) query[pair[0]] = value;
  }
  return query;
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
  if ((systems && systems.status === 401) || (status && status.status === 401)) {
    state.systems = [];
    state.status = null;
    renderAll();
    return;
  }
  state.systems = Array.isArray(systems) ? systems : [];
  state.status = status || null;
  syncSelection();
  renderAll();
}
async function refreshStatusOnly() {
  const status = await api('/api/status');
  state.status = status || null;
  const activeId = status && status.active_system_id ? status.active_system_id : '';
  for (const sys of state.systems) {
    sys.active = !!activeId && sys.id === activeId;
    sys.connected = !!activeId && sys.id === activeId && !!status.connected;
  }
  renderAll();
}
function renderAll() {
  renderTree(); renderEditor();
}
function setEditor(kind, mode) {
  state.editor = {kind: kind, mode: mode || 'edit'};
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
      const ops = ent.operations.length ? ent.operations.map(op => '<div class="op-mini"><div class="op-row"><div onclick="selectOperation(\'' + esc(op.id) + '\')" style="flex:1;cursor:pointer"><strong>' + esc(operationTitle(op)) + '</strong> <span style="color:#94a3b8">' + esc(opLabel(op.verb)) + ' · ' + esc(op.entity_set) + '</span></div><div style="display:flex;gap:6px;align-items:center;flex-wrap:wrap">' + (op.tool_name ? '<span class="badge badge-c">' + esc(op.tool_name) + '</span>' : '') + '<button class="btn btn-s" onclick="openOperationEditModal(\'' + esc(ent.id) + '\', \'' + esc(op.id) + '\')">' + T.btn_edit + '</button><button class="btn btn-s btn-d" onclick="deleteOperationFromTree(\'' + esc(ent.id) + '\', \'' + esc(op.id) + '\')">' + T.btn_delete + '</button></div></div>' + operationQueryBadges(op.query) + '</div>').join('') : '<div class="hint">' + T.empty_ops + '</div>';
      return '<div class="entity-node">' +
        '<div class="entity-head"><div onclick="selectEntity(\'' + esc(ent.id) + '\')" style="flex:1;cursor:pointer"><div class="entity-title">' + esc(ent.label) + '</div><div class="entity-desc">' + esc(ent.description || '') + '</div></div><div style="display:flex;gap:6px;flex-wrap:wrap"><button class="btn btn-s" onclick="newOperationForEntity(\'' + esc(ent.id) + '\')">' + T.btn_new_op + '</button><button class="btn btn-s" onclick="openEntityEditModal(\'' + esc(ent.id) + '\')">' + T.btn_edit + '</button><button class="btn btn-s btn-d" onclick="deleteEntityFromTree(\'' + esc(ent.id) + '\')">' + T.btn_delete + '</button></div></div>' +
        ops + '</div>';
    }).join('') : '<div class="hint">' + T.empty_entities + '</div>';
    return '<div class="tree-item">' +
      '<div class="tree-row"><div class="tree-main" onclick="selectSystem(\'' + esc(sys.id) + '\')" style="cursor:pointer"><div class="tree-name">' + esc(sys.name) + '</div><div class="tree-details">' + esc(systemDetails(sys)) + '</div><div style="margin-top:4px;display:flex;gap:5px;flex-wrap:wrap">' + sysBadges + (sys.active ? '<span class="badge badge-c">' + T.active + '</span>' : '') + '</div></div><div class="tree-actions"><button class="btn btn-s" onclick="newEntityForSystem(\'' + esc(sys.id) + '\')">' + T.btn_new_entity + '</button><button class="btn btn-s" onclick="openSystemServicesModal(\'' + esc(sys.id) + '\')">' + T.btn_services + '</button><button class="btn btn-s" onclick="testSystemFromTree(\'' + esc(sys.id) + '\')">' + T.btn_test + '</button><button class="btn btn-s" onclick="openSystemEditModal(\'' + esc(sys.id) + '\')">' + T.btn_edit + '</button><button class="btn btn-s" onclick="activateSystemFromTree(\'' + esc(sys.id) + '\')">' + T.btn_activate + '</button><button class="btn btn-s btn-d" onclick="deleteSystemFromTree(\'' + esc(sys.id) + '\')">' + T.btn_delete + '</button></div></div>' +
      entities + '</div>';
  }).join('');
}
function renderEditor() {
  const root = document.getElementById('editor-pane');
  if (!root) return;
  if (state.editor.kind === 'system') {
    root.innerHTML = renderSystemEditorTemplate();
    renderSystemForm();
    return;
  }
  if (state.editor.kind === 'object') {
    root.innerHTML = renderObjectEditorTemplate();
    renderEntityForm();
    return;
  }
  if (state.editor.kind === 'operation') {
    root.innerHTML = renderOperationEditorTemplate();
    renderOperationForm();
    return;
  }
  root.innerHTML = '<h2>' + T.h_editor + '</h2><div class="empty">' + T.editor_empty + '</div>';
}
function renderSystemEditorTemplate() {
  const showExisting = state.editor.mode !== 'create' && !!activeSystem();
  return '<h2>' + (state.editor.mode === 'create' ? T.btn_new_system : T.h_system) + '</h2>' +
    '<div class="form-grid">' +
      '<div class="form-group"><label>' + T.system_name + '</label><input id="sys-name"></div>' +
      '<div class="form-group"><label>' + T.sap_client + '</label><input id="sys-client" placeholder="100"></div>' +
      '<div class="form-group full"><label>' + T.base_url + '</label><input id="sys-base" placeholder="http://s4d.msgplaut.com:8000"></div>' +
      '<div class="form-group"><label>' + T.login + '</label><input id="sys-user" autocomplete="off"></div>' +
      '<div class="form-group"><label>' + T.password + '</label><input id="sys-pass" type="password" autocomplete="new-password"></div>' +
      '<div class="form-group full"><label class="toggle"><input type="checkbox" id="sys-write"><span class="toggle-track"></span>' + T.allow_write + '</label></div>' +
    '</div>' +
    '<div class="form-actions">' +
      '<button class="btn" onclick="saveSystem()">' + T.btn_save + '</button>' +
      (showExisting ? '<button class="btn" onclick="activateSystem()">' + T.btn_activate + '</button>' : '') +
      (showExisting ? '<button class="btn btn-d" onclick="deleteSystem()">' + T.btn_delete + '</button>' : '') +
    '</div>' +
    (showExisting ? '<div class="hint">' + T.editor_empty + '</div>' : '');
}
function renderObjectEditorTemplate() {
  return '<h2>' + (state.editor.mode === 'create' ? T.btn_new_entity : T.h_entity) + '</h2>' +
    '<div class="form-grid">' +
      '<div class="form-group"><label>' + T.entity_label + '</label><input id="ent-label"></div>' +
      '<div class="form-group full"><label>' + T.entity_desc + '</label><textarea id="ent-desc"></textarea></div>' +
    '</div>' +
    '<div class="form-actions">' +
      '<button class="btn" onclick="saveEntity()">' + T.btn_save + '</button>' +
      (state.editor.mode !== 'create' ? '<button class="btn btn-d" onclick="deleteEntity()">' + T.btn_delete + '</button>' : '') +
    '</div>';
}
function renderOperationEditorTemplate() {
  return '<h2>' + (state.editor.mode === 'create' ? T.btn_new_op : T.h_operation) + '</h2>' +
    '<div class="form-grid">' +
      '<div class="form-group full"><label>' + T.op_name + '</label><input id="op-name"></div>' +
      '<div class="form-group"><label>' + T.op_verb + '</label><select id="op-verb"></select></div>' +
      '<div class="form-group"><label>' + T.op_service + '</label><select id="op-service"></select></div>' +
      '<div class="form-group full"><label>' + T.op_entityset + '</label><select id="op-entityset"></select></div>' +
      '<div class="form-group"><label>' + T.op_query_expand + '</label><input id="op-query-expand" placeholder="ToDescription"></div>' +
      '<div class="form-group"><label>' + T.op_query_select + '</label><input id="op-query-select" placeholder="MATNR,ToDescription"></div>' +
      '<div class="form-group"><label>' + T.op_query_filter + '</label><input id="op-query-filter"></div>' +
      '<div class="form-group"><label>' + T.op_query_orderby + '</label><input id="op-query-orderby"></div>' +
      '<div class="form-group"><label>' + T.op_query_top + '</label><input id="op-query-top" type="number" min="0"></div>' +
    '</div>' +
    '<div id="discovery" class="hint">' + T.discovery_hint + '</div>' +
    '<div class="form-actions">' +
      '<button class="btn" onclick="saveOperation()">' + T.btn_save + '</button>' +
      '<button class="btn" onclick="refreshDiscovery()">' + T.btn_discover + '</button>' +
      (state.editor.mode !== 'create' ? '<button class="btn btn-d" onclick="deleteOperation()">' + T.btn_delete + '</button>' : '') +
    '</div>';
}
function renderSystemForm() {
  if (!document.getElementById('sys-name')) return;
  const sys = activeSystem();
  document.getElementById('sys-name').value = sys ? sys.name : '';
  document.getElementById('sys-base').value = sys ? (sys.base_url || '') : '';
  document.getElementById('sys-client').value = sys ? (sys.client || '') : '';
  document.getElementById('sys-user').value = sys ? (sys.username || '') : '';
  document.getElementById('sys-pass').value = '';
  document.getElementById('sys-write').checked = !!(sys && sys.access_mode === 'unrestricted');
}
function renderEntityForm() {
  if (!document.getElementById('ent-label')) return;
  const sys = activeSystem(); const ent = activeEntity();
  document.getElementById('ent-label').value = ent ? ent.label : '';
  document.getElementById('ent-desc').value = ent ? (ent.description || '') : '';
  if (!sys) document.getElementById('ent-label').placeholder = T.select_system;
}
function renderOperationForm() {
  if (!document.getElementById('op-verb')) return;
  const sys = activeSystem(); const ent = activeEntity(); const op = activeOperation();
  const verbSel = document.getElementById('op-verb');
  const nameInput = document.getElementById('op-name');
  verbSel.innerHTML = VERBS.map(v => '<option value="' + v.value + '">' + v.label + '</option>').join('');
  const serviceSel = document.getElementById('op-service');
  if (!sys) {
    serviceSel.innerHTML = '<option value="">' + T.select_system + '</option>';
    document.getElementById('op-entityset').innerHTML = '<option value="">' + T.select_service + '</option>';
    return;
  }
  serviceSel.innerHTML = '<option value="">' + T.select_service + '</option>' + sys.services.map(s => '<option value="' + esc(s.id) + '">' + esc(s.name) + '</option>').join('');
  if (op) {
    nameInput.value = op.name || '';
    verbSel.value = op.verb;
    serviceSel.value = op.service_id;
    fillOperationQuery('op-query-', op.query);
  } else if (sys.services[0]) {
    serviceSel.value = sys.services[0].id;
    fillOperationQuery('op-query-', {});
  }
  if (serviceSel.value && !state.discovery[state.selectedSystemId + '::' + serviceSel.value]) {
    discoverService(state.selectedSystemId, serviceSel.value).then(() => renderOperationForm());
  }
  serviceSel.onchange = async function() {
    const currentSys = activeSystem(); if (!currentSys || !this.value) return populateEntitySetOptions('', '');
    if (!state.discovery[currentSys.id + '::' + this.value]) await discoverService(currentSys.id, this.value);
    populateEntitySetOptions(this.value, '');
  };
  verbSel.onchange = function() {
    nameInput.placeholder = suggestOperationName(this.value, document.getElementById('op-entityset').value);
  };
  populateEntitySetOptions(serviceSel.value, op ? op.entity_set : '');
  nameInput.placeholder = suggestOperationName(verbSel.value, document.getElementById('op-entityset').value);
}
function populateEntitySetOptions(serviceId, selected) {
  const sel = document.getElementById('op-entityset');
  const key = state.selectedSystemId + '::' + serviceId;
  const discovery = state.discovery[key];
  const options = discovery && discovery.entity_sets ? discovery.entity_sets : [];
  if (!serviceId) { sel.innerHTML = '<option value="">' + T.select_service + '</option>'; return; }
  sel.innerHTML = '<option value="">' + T.select_entityset + '</option>' + options.map(x => '<option value="' + esc(x.name) + '">' + esc(x.name) + '</option>').join('');
  if (selected) sel.value = selected;
  sel.onchange = function() {
    const nameInput = document.getElementById('op-name');
    const verbSel = document.getElementById('op-verb');
    if (nameInput && verbSel) nameInput.placeholder = suggestOperationName(verbSel.value, this.value);
  };
  document.getElementById('discovery').innerHTML = options.length ? '<strong>' + T.discovery_sets + ':</strong> ' + options.map(x => esc(x.name)).join(', ') : T.discovery_hint;
}
function selectSystem(id) { state.selectedSystemId = id; state.selectedEntityId = ''; state.selectedOperationId = ''; state.editingServiceId = ''; syncSelection(); renderAll(); }
function selectEntity(id) { state.selectedEntityId = id; state.selectedOperationId = ''; syncSelection(); renderAll(); }
function selectOperation(id) { state.selectedOperationId = id; syncSelection(); renderAll(); }
function newSystem() { state.selectedSystemId = ''; state.selectedEntityId = ''; state.selectedOperationId = ''; state.editingServiceId = ''; setEditor('system', 'create'); renderAll(); }
function newEntity() { if (!activeSystem()) { toast(T.select_system, true); return; } state.selectedEntityId = ''; state.selectedOperationId = ''; setEditor('object', 'create'); renderAll(); }
function newOperation() { if (!activeEntity()) { toast(T.select_entity, true); return; } state.selectedOperationId = ''; setEditor('operation', 'create'); renderAll(); }
function newEntityForSystem(id) { selectSystem(id); state.selectedEntityId = ''; state.selectedOperationId = ''; setEditor('object', 'create'); renderAll(); }
function newOperationForEntity(id) { selectEntity(id); state.selectedOperationId = ''; setEditor('operation', 'create'); renderAll(); }
async function activateSystemFromTree(id) { selectSystem(id); return activateSystem(); }
async function testSystemFromTree(id) {
  selectSystem(id);
  const sys = activeSystem();
  if (!sys) return toast(T.select_system, true);
  const result = await api('/api/system/test', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id})});
  if (!result || result.error) return toast(T.msg_test_fail + ': ' + ((result && result.error) || 'unknown'), true);
  sys.connected = !!result.ok;
  renderAll();
  const services = Array.isArray(result.services) ? result.services : [];
  const failed = services.filter(x => !x.ok).map(x => x.service_name || x.service_id);
  const details = failed.length ? ' — ' + failed.join(', ') : '';
  return toast((result.message || (result.ok ? T.msg_test_ok : T.msg_test_fail)) + (result.duration_ms ? ' (' + result.duration_ms + ' ms)' : '') + details, !result.ok);
}
async function discoverServiceFromManager(id) {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  await discoverService(sys.id, id);
  toast(T.msg_discovery);
}
function deleteSystemFromTree(id) {
  selectSystem(id);
  return deleteSystem();
}
function deleteEntityFromTree(id) {
  selectEntity(id);
  return deleteEntity();
}
function deleteOperationFromTree(entityId, operationId) {
  selectEntity(entityId);
  selectOperation(operationId);
  return deleteOperation();
}
function closeOverlay() {
  const ov = document.querySelector('.overlay');
  if (ov) ov.remove();
}
function openOverlay(html) {
  closeOverlay();
  const ov = document.createElement('div');
  ov.className = 'overlay';
  ov.innerHTML = html;
  ov.addEventListener('click', function(e) { if (e.target === ov) ov.remove(); });
  document.body.appendChild(ov);
}
function confirmDeleteDialog(message) {
  return new Promise(function(resolve) {
    openOverlay(
      '<div class="modal" style="width:360px;text-align:center">' +
        '<h3>' + T.confirm_delete + '</h3>' +
        '<p style="color:#94a3b8;font-size:.82rem;margin-bottom:14px">' + esc(message) + '</p>' +
        '<div class="modal-actions" style="justify-content:center">' +
          '<button class="btn" id="confirm-cancel-btn">' + T.btn_cancel + '</button>' +
          '<button class="btn btn-d" id="confirm-delete-btn">' + T.btn_delete + '</button>' +
        '</div>' +
      '</div>'
    );
    document.getElementById('confirm-cancel-btn').onclick = function() { closeOverlay(); resolve(false); };
    document.getElementById('confirm-delete-btn').onclick = function() { closeOverlay(); resolve(true); };
  });
}
function buildServiceOptions(sys, selectedId) {
  return '<option value="">' + T.select_service + '</option>' + sys.services.map(s => '<option value="' + esc(s.id) + '"' + (s.id === selectedId ? ' selected' : '') + '>' + esc(s.name) + '</option>').join('');
}
function renderSystemServicesModalBody(sys) {
  if (!sys.services.length) return '<div class="empty">' + T.empty_services + '</div>';
  return sys.services.map(function(svc) {
    return '<div class="svc-item">' +
      '<div class="svc-main"><div class="tree-name">' + esc(svc.name) + '</div><div class="tree-details">' + esc(svc.safe_service_url || svc.service_url) + '</div></div>' +
      '<div style="display:flex;gap:6px;flex-wrap:wrap">' +
        '<button class="btn btn-s" onclick="openServiceEditModal(\'' + esc(svc.id) + '\', true)">' + T.btn_edit + '</button>' +
        '<button class="btn btn-s" onclick="discoverServiceFromManager(\'' + esc(svc.id) + '\')">' + T.btn_discover + '</button>' +
        '<button class="btn btn-s btn-d" onclick="deleteServiceFromManager(\'' + esc(svc.id) + '\')">' + T.btn_delete + '</button>' +
      '</div>' +
    '</div>';
  }).join('');
}
function openSystemServicesModal(id) {
  selectSystem(id);
  const sys = activeSystem();
  if (!sys) return toast(T.select_system, true);
  openOverlay(
    '<div class="modal" style="width:820px">' +
      '<h3>' + T.h_manage_services + '</h3>' +
      '<div class="hint" style="margin-bottom:12px">' + T.service_note + '</div>' +
      '<div class="toolbar" style="margin-bottom:12px"><button class="btn" onclick="openServiceEditModal(\'\', true)">' + T.btn_new_service + '</button></div>' +
      '<div id="m-services-list">' + renderSystemServicesModalBody(sys) + '</div>' +
      '<div class="modal-actions">' +
        '<button class="btn" onclick="closeOverlay()">' + T.btn_cancel + '</button>' +
      '</div>' +
    '</div>'
  );
}
function openSystemEditModal(id) {
  selectSystem(id);
  const sys = activeSystem();
  if (!sys) return;
  openOverlay(
    '<div class="modal">' +
      '<h3>' + T.h_edit_system + '</h3>' +
      '<div class="form-grid">' +
        '<div class="form-group"><label>' + T.system_name + '</label><input id="m-sys-name" value="' + escAttr(sys.name) + '"></div>' +
        '<div class="form-group"><label>' + T.sap_client + '</label><input id="m-sys-client" value="' + escAttr(sys.client || '') + '" placeholder="100"></div>' +
        '<div class="form-group full"><label>' + T.base_url + '</label><input id="m-sys-base" value="' + escAttr(sys.base_url || '') + '"></div>' +
        '<div class="form-group"><label>' + T.login + '</label><input id="m-sys-user" value="' + escAttr(sys.username || '') + '" autocomplete="off"></div>' +
        '<div class="form-group"><label>' + T.password + '</label><input id="m-sys-pass" type="password" autocomplete="new-password" placeholder="' + (sys.has_password ? '••••••••' : '') + '"></div>' +
        '<div class="form-group full"><label class="toggle"><input type="checkbox" id="m-sys-write" ' + (sys.access_mode === 'unrestricted' ? 'checked' : '') + '><span class="toggle-track"></span>' + T.allow_write + '</label></div>' +
      '</div>' +
      '<div class="modal-actions">' +
        '<button class="btn" onclick="closeOverlay()">' + T.btn_cancel + '</button>' +
        '<button class="btn" onclick="saveSystemModal(\'' + escAttr(sys.id) + '\')">' + T.btn_save + '</button>' +
      '</div>' +
    '</div>'
  );
}
async function saveSystemModal(oldId) {
  const payload = {
    old_id: oldId,
    name: document.getElementById('m-sys-name').value.trim(),
    base_url: document.getElementById('m-sys-base').value.trim(),
    client: document.getElementById('m-sys-client').value.trim(),
    username: document.getElementById('m-sys-user').value.trim(),
    password: document.getElementById('m-sys-pass').value,
    access_mode: document.getElementById('m-sys-write').checked ? 'unrestricted' : 'restricted'
  };
  const r = await api('/api/system/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  closeOverlay();
  toast(T.msg_saved);
  await loadAll();
}
function openServiceEditModal(id, reopenManager) {
  const sys = activeSystem();
  if (!sys) return toast(T.select_system, true);
  const svc = id ? sys.services.find(x => x.id === id) : null;
  if (id && !svc) return toast(T.select_system, true);
  openOverlay(
    '<div class="modal">' +
      '<h3>' + (svc ? T.h_edit_service : T.btn_new_service) + '</h3>' +
      '<div class="form-grid">' +
        '<div class="form-group"><label>' + T.service_name + '</label><input id="m-svc-name" value="' + escAttr(svc ? svc.name : '') + '"></div>' +
        '<div class="form-group full"><label>' + T.service_url + '</label><input id="m-svc-url" value="' + escAttr(svc ? svc.service_url : '') + '"></div>' +
      '</div>' +
      '<div class="modal-actions">' +
        '<button class="btn" onclick="' + (reopenManager ? 'openSystemServicesModal(\'' + escAttr(sys.id) + '\')' : 'closeOverlay()') + '">' + T.btn_cancel + '</button>' +
        '<button class="btn" onclick="saveServiceModal(\'' + escAttr(id) + '\', ' + (reopenManager ? 'true' : 'false') + ')">' + T.btn_save + '</button>' +
      '</div>' +
    '</div>'
  );
}
async function saveServiceModal(oldId, reopenManager) {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const payload = {system_id: sys.id, old_id: oldId, name: document.getElementById('m-svc-name').value.trim(), service_url: document.getElementById('m-svc-url').value.trim()};
  const r = await api('/api/service/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  await loadAll();
  if (reopenManager) openSystemServicesModal(sys.id);
  else closeOverlay();
  toast(T.msg_saved);
}
function openEntityEditModal(id) {
  selectEntity(id);
  const ent = activeEntity();
  if (!ent) return toast(T.select_entity, true);
  openOverlay(
    '<div class="modal">' +
      '<h3>' + T.h_edit_entity + '</h3>' +
      '<div class="form-grid">' +
        '<div class="form-group"><label>' + T.entity_label + '</label><input id="m-ent-label" value="' + escAttr(ent.label) + '"></div>' +
        '<div class="form-group full"><label>' + T.entity_desc + '</label><textarea id="m-ent-desc">' + esc(ent.description || '') + '</textarea></div>' +
      '</div>' +
      '<div class="modal-actions">' +
        '<button class="btn" onclick="closeOverlay()">' + T.btn_cancel + '</button>' +
        '<button class="btn" onclick="saveEntityModal(\'' + escAttr(ent.id) + '\')">' + T.btn_save + '</button>' +
      '</div>' +
    '</div>'
  );
}
async function saveEntityModal(oldId) {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const payload = {system_id: sys.id, old_id: oldId, label: document.getElementById('m-ent-label').value.trim(), description: document.getElementById('m-ent-desc').value.trim()};
  const r = await api('/api/entity/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  closeOverlay();
  toast(T.msg_saved);
  await loadAll();
}
async function openOperationEditModal(entityId, operationId) {
  selectEntity(entityId);
  selectOperation(operationId);
  const sys = activeSystem();
  const op = activeOperation();
  if (!sys || !op) return toast(T.select_entity, true);
  if (op.service_id && !state.discovery[sys.id + '::' + op.service_id]) await discoverService(sys.id, op.service_id);
  openOverlay(
    '<div class="modal">' +
      '<h3>' + T.h_edit_operation + '</h3>' +
      '<div class="form-grid">' +
        '<div class="form-group full"><label>' + T.op_name + '</label><input id="m-op-name" value="' + escAttr(op.name || '') + '"></div>' +
        '<div class="form-group"><label>' + T.op_verb + '</label><select id="m-op-verb">' + VERBS.map(v => '<option value="' + v.value + '"' + (v.value === op.verb ? ' selected' : '') + '>' + v.label + '</option>').join('') + '</select></div>' +
        '<div class="form-group"><label>' + T.op_service + '</label><select id="m-op-service" onchange="changeOperationModalService()">' + buildServiceOptions(sys, op.service_id) + '</select></div>' +
        '<div class="form-group full"><label>' + T.op_entityset + '</label><select id="m-op-entityset"></select></div>' +
        '<div class="form-group"><label>' + T.op_query_expand + '</label><input id="m-op-query-expand" placeholder="ToDescription"></div>' +
        '<div class="form-group"><label>' + T.op_query_select + '</label><input id="m-op-query-select"></div>' +
        '<div class="form-group"><label>' + T.op_query_filter + '</label><input id="m-op-query-filter"></div>' +
        '<div class="form-group"><label>' + T.op_query_orderby + '</label><input id="m-op-query-orderby"></div>' +
        '<div class="form-group"><label>' + T.op_query_top + '</label><input id="m-op-query-top" type="number" min="0"></div>' +
      '</div>' +
      '<div id="m-op-discovery" class="hint">' + T.discovery_hint + '</div>' +
      '<div class="modal-actions">' +
        '<button class="btn" onclick="closeOverlay()">' + T.btn_cancel + '</button>' +
        '<button class="btn" onclick="saveOperationModal(\'' + escAttr(operationId) + '\')">' + T.btn_save + '</button>' +
      '</div>' +
    '</div>'
  );
  populateOperationModalEntitySets(sys.id, op.service_id, op.entity_set);
  fillOperationQuery('m-op-query-', op.query);
}
async function changeOperationModalService() {
  const sys = activeSystem(); if (!sys) return;
  const serviceId = document.getElementById('m-op-service').value;
  if (serviceId && !state.discovery[sys.id + '::' + serviceId]) await discoverService(sys.id, serviceId);
  populateOperationModalEntitySets(sys.id, serviceId, '');
}
function populateOperationModalEntitySets(systemId, serviceId, selected) {
  const sel = document.getElementById('m-op-entityset');
  const hint = document.getElementById('m-op-discovery');
  if (!sel || !hint) return;
  const key = systemId + '::' + serviceId;
  const discovery = state.discovery[key];
  const options = discovery && discovery.entity_sets ? discovery.entity_sets : [];
  if (!serviceId) { sel.innerHTML = '<option value="">' + T.select_service + '</option>'; hint.innerHTML = T.discovery_hint; return; }
  sel.innerHTML = '<option value="">' + T.select_entityset + '</option>' + options.map(x => '<option value="' + esc(x.name) + '">' + esc(x.name) + '</option>').join('');
  if (selected) sel.value = selected;
  hint.innerHTML = options.length ? '<strong>' + T.discovery_sets + ':</strong> ' + options.map(x => esc(x.name)).join(', ') : T.discovery_hint;
}
async function saveOperationModal(oldId) {
  const sys = activeSystem(); const ent = activeEntity(); if (!sys || !ent) return toast(T.select_entity, true);
  const payload = {system_id: sys.id, entity_id: ent.id, old_id: oldId, id: oldId, name: document.getElementById('m-op-name').value.trim(), verb: document.getElementById('m-op-verb').value, service_id: document.getElementById('m-op-service').value, entity_set: document.getElementById('m-op-entityset').value, query: collectOperationQuery('m-op-query-'), mode: 'generated', enabled: true};
  const r = await api('/api/operation/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  closeOverlay();
  toast(T.msg_saved);
  await loadAll();
}
async function saveSystem() {
  const current = activeSystem();
  const payload = { id: current ? current.id : '', old_id: current ? current.id : '', name: document.getElementById('sys-name').value.trim(), base_url: document.getElementById('sys-base').value.trim(), client: document.getElementById('sys-client').value.trim(), username: document.getElementById('sys-user').value.trim(), password: document.getElementById('sys-pass').value, access_mode: document.getElementById('sys-write').checked ? 'unrestricted' : 'restricted' };
  const r = await api('/api/system/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  if (current) {
    current.name = payload.name;
    current.base_url = payload.base_url;
    current.client = payload.client;
    current.username = payload.username;
    current.access_mode = payload.access_mode;
    renderAll();
    refreshStatusOnly();
  } else {
    await loadAll();
  }
  toast(T.msg_saved);
}
async function activateSystem() {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  const r = await api('/api/system/activate', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_activated); await loadAll();
}
async function deleteSystem() {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  if (!await confirmDeleteDialog(sys.name)) return;
  const r = await api('/api/system/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({id: sys.id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_deleted); state.selectedSystemId=''; await loadAll();
}
async function deleteServiceFromManager(id) {
  const sys = activeSystem(); if (!sys) return toast(T.select_system, true);
  if (!await confirmDeleteDialog(id)) return;
  const r = await api('/api/service/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id, service_id: id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  await loadAll();
  openSystemServicesModal(sys.id);
  return toast(T.msg_deleted);
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
  if (!await confirmDeleteDialog(ent.label)) return;
  const r = await api('/api/entity/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id, entity_id: ent.id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_deleted); state.selectedEntityId=''; await loadAll();
}
async function saveOperation() {
  const sys = activeSystem(); const ent = activeEntity(); if (!sys || !ent) return toast(T.select_entity, true);
  const op = activeOperation();
  const payload = {system_id: sys.id, entity_id: ent.id, old_id: op ? op.id : '', id: op ? op.id : '', name: document.getElementById('op-name').value.trim(), verb: document.getElementById('op-verb').value, service_id: document.getElementById('op-service').value, entity_set: document.getElementById('op-entityset').value, query: collectOperationQuery('op-query-'), mode: 'generated', enabled: true};
  const r = await api('/api/operation/save', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_saved); await loadAll();
}
async function deleteOperation() {
  const sys = activeSystem(); const ent = activeEntity(); const op = activeOperation();
  if (!sys || !ent || !op) return toast(T.empty_ops, true);
  if (!await confirmDeleteDialog(opLabel(op.verb))) return;
  const r = await api('/api/operation/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({system_id: sys.id, entity_id: ent.id, operation_id: op.id})});
  if (!r || r.error || r.ok === false) return toast(T.msg_error + ': ' + ((r && (r.error || r.message)) || 'unknown'), true);
  toast(T.msg_deleted); state.selectedOperationId=''; await loadAll();
}
loadAll();
</script>
</body></html>`
