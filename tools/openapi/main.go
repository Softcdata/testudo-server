package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

type serverInterface struct {
	ModuleName     string `json:"module_name"`
	Module         string `json:"module"`
	Method         string `json:"method"`
	FullPath       string `json:"full_path"`
	Handler        string `json:"handler"`
	File           string `json:"file"`
	WebSocket      string `json:"websocket"`
	Auth           string `json:"auth"`
	DirectResource string `json:"direct_resource"`
	Notes          string `json:"notes"`
}

type runAPIInterface struct {
	TargetID          string
	FolderPath        string
	Name              string
	TargetType        string
	Method            string
	Path              string
	URL               string
	MarkID            string
	Version           string
	DescriptionStatus string
}

type routeInfo struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
}

func main() {
	if len(os.Args) < 2 {
		failf("usage: go run ./tools/openapi <routes|generate|validate|diff|checklist> [flags]")
	}

	var err error
	switch os.Args[1] {
	case "routes":
		err = routes(os.Args[2:])
	case "generate":
		err = generate(os.Args[2:])
	case "validate":
		err = validate(os.Args[2:])
	case "diff":
		err = diff(os.Args[2:])
	case "checklist":
		err = checklist(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		failf("%v", err)
	}
}

func routes(args []string) error {
	fs := flag.NewFlagSet("routes", flag.ExitOnError)
	serverPath := fs.String("server", "openspec/changes/add-swagger-openapi-support/artifacts/server-interfaces.json", "server interface JSON")
	output := fs.String("output", "", "output JSON file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	items, err := loadServerInterfaces(*serverPath)
	if err != nil {
		return err
	}
	routes := make([]routeInfo, 0, len(items))
	for _, item := range items {
		routes = append(routes, routeInfo{Method: item.Method, Path: item.FullPath, Handler: item.Handler})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return err
	}
	return writeOutput(*output, data)
}

func generate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	serverPath := fs.String("server", "openspec/changes/add-swagger-openapi-support/artifacts/server-interfaces.json", "server interface JSON")
	runapiCSV := fs.String("runapi", "openspec/changes/add-swagger-openapi-support/artifacts/runapi-interfaces.csv", "RunAPI interface CSV")
	checklist := fs.String("checklist", "openspec/changes/update-runapi-detail-description-standard/artifacts/interface-checklist.md", "RunAPI checklist markdown")
	output := fs.String("output", "openspec/specs/disaster-server-openapi.yaml", "output OpenAPI YAML")
	if err := fs.Parse(args); err != nil {
		return err
	}

	serverItems, err := loadServerInterfaces(*serverPath)
	if err != nil {
		return err
	}
	runapiItems, _ := loadRunAPIInterfaces(*runapiCSV)
	targetsFromChecklist, _ := loadChecklistTargets(*checklist)
	targets := make(map[string]string)
	for _, item := range runapiItems {
		key := interfaceKey(item.Method, item.Path)
		if key != "" && targets[key] == "" {
			targets[key] = item.TargetID
		}
	}
	for key, value := range targetsFromChecklist {
		targets[key] = value
	}

	doc := buildOpenAPI(serverItems, targets)
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return writeOutput(*output, data)
}

func validate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	specPath := fs.String("spec", "openspec/specs/disaster-server-openapi.yaml", "OpenAPI YAML")
	if err := fs.Parse(args); err != nil {
		return err
	}

	data, err := os.ReadFile(*specPath)
	if err != nil {
		return err
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	if _, err := json.Marshal(doc); err != nil {
		return fmt.Errorf("convert OpenAPI YAML to JSON: %w", err)
	}
	if doc["openapi"] != "3.0.3" {
		return fmt.Errorf("openapi must be 3.0.3, got %v", doc["openapi"])
	}
	paths, ok := doc["paths"].(map[string]interface{})
	if !ok || len(paths) == 0 {
		return errors.New("paths must not be empty")
	}
	ids := make(map[string]string)
	for path, rawPathItem := range paths {
		pathItem, ok := rawPathItem.(map[string]interface{})
		if !ok {
			return fmt.Errorf("path %s must be an object", path)
		}
		for method, rawOperation := range pathItem {
			operation, ok := rawOperation.(map[string]interface{})
			if !ok {
				return fmt.Errorf("%s %s must be an operation object", strings.ToUpper(method), path)
			}
			for _, field := range []string{"tags", "summary", "description", "operationId", "parameters", "requestBody", "responses", "security", "x-runapi-target-id", "x-controlled-resources", "x-operator-chain"} {
				if _, ok := operation[field]; !ok {
					return fmt.Errorf("%s %s missing %s", strings.ToUpper(method), path, field)
				}
			}
			id, _ := operation["operationId"].(string)
			if id == "" {
				return fmt.Errorf("%s %s has empty operationId", strings.ToUpper(method), path)
			}
			if prev := ids[id]; prev != "" {
				return fmt.Errorf("duplicated operationId %s: %s and %s %s", id, prev, strings.ToUpper(method), path)
			}
			ids[id] = strings.ToUpper(method) + " " + path
			if operation["x-disaster-protocol"] == "websocket" {
				if _, ok := operation["x-message-schema"]; !ok {
					return fmt.Errorf("%s %s missing x-message-schema", strings.ToUpper(method), path)
				}
				responses, _ := operation["responses"].(map[string]interface{})
				if _, ok := responses["101"]; !ok {
					return fmt.Errorf("%s %s websocket operation missing 101 response", strings.ToUpper(method), path)
				}
			}
		}
	}
	return nil
}

func diff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	serverPath := fs.String("server", "openspec/changes/add-swagger-openapi-support/artifacts/server-interfaces.json", "server interface JSON")
	runapiCSV := fs.String("runapi", "openspec/changes/add-swagger-openapi-support/artifacts/runapi-interfaces.csv", "RunAPI interface CSV")
	checklist := fs.String("checklist", "openspec/changes/update-runapi-detail-description-standard/artifacts/interface-checklist.md", "RunAPI checklist markdown")
	specPath := fs.String("spec", "openspec/specs/disaster-server-openapi.yaml", "OpenAPI YAML")
	output := fs.String("output", "", "output markdown")
	if err := fs.Parse(args); err != nil {
		return err
	}
	serverItems, err := loadServerInterfaces(*serverPath)
	if err != nil {
		return err
	}
	runapiItems, err := loadRunAPIInterfaces(*runapiCSV)
	if err != nil {
		return err
	}
	checklistTargets, _ := loadChecklistTargets(*checklist)
	openapiKeys, err := loadOpenAPIKeys(*specPath)
	if err != nil {
		return err
	}

	serverKeys := map[string]serverInterface{}
	for _, item := range serverItems {
		serverKeys[interfaceKey(item.Method, item.FullPath)] = item
	}
	runapiKeys := map[string][]runAPIInterface{}
	for _, item := range runapiItems {
		if key := interfaceKey(item.Method, item.Path); key != "" {
			runapiKeys[key] = append(runapiKeys[key], item)
		}
	}
	for key, targetID := range checklistTargets {
		method, path := splitInterfaceKey(key)
		runapiKeys[key] = append(runapiKeys[key], runAPIInterface{
			TargetID:   targetID,
			TargetType: "checklist",
			Method:     method,
			Path:       path,
		})
	}

	var b strings.Builder
	b.WriteString("# Swagger/OpenAPI 三方差异对账\n\n")
	fmt.Fprintf(&b, "- Server 接口总数：%d\n", len(serverItems))
	fmt.Fprintf(&b, "- RunAPI 接口总数：%d\n", len(runapiKeys))
	fmt.Fprintf(&b, "- OpenAPI 接口总数：%d\n\n", len(openapiKeys))

	writeMissing := func(title string, keys []string) {
		b.WriteString("## " + title + "\n\n")
		if len(keys) == 0 {
			b.WriteString("无。\n\n")
			return
		}
		b.WriteString("| 方法路径 | 说明 |\n|---|---|\n")
		for _, key := range keys {
			b.WriteString("| `" + key + "` | 待处理 |\n")
		}
		b.WriteString("\n")
	}

	writeMissing("OpenAPI 缺失的 server 接口", missingServerKeys(serverKeys, openapiKeys))
	writeMissing("RunAPI 缺失的 server 接口", missingRunAPIKeys(serverKeys, runapiKeys))
	writeMissing("RunAPI 额外接口", extraRunAPIKeys(serverKeys, runapiKeys))
	writeMissing("OpenAPI 额外接口", extraOpenAPIKeys(serverKeys, openapiKeys))

	return writeOutput(*output, []byte(b.String()))
}

func checklist(args []string) error {
	fs := flag.NewFlagSet("checklist", flag.ExitOnError)
	serverPath := fs.String("server", "openspec/changes/add-swagger-openapi-support/artifacts/server-interfaces.json", "server interface JSON")
	checklistPath := fs.String("runapi-checklist", "openspec/changes/update-runapi-detail-description-standard/artifacts/interface-checklist.md", "RunAPI checklist markdown")
	output := fs.String("output", "", "output markdown")
	if err := fs.Parse(args); err != nil {
		return err
	}
	serverItems, err := loadServerInterfaces(*serverPath)
	if err != nil {
		return err
	}
	targets, _ := loadChecklistTargets(*checklistPath)

	var b strings.Builder
	b.WriteString("# Swagger/OpenAPI 逐接口勾选清单\n\n")
	b.WriteString("勾选规则：只有完成调用链取证、request schema、response schema、错误响应、扩展字段、Swagger UI 渲染检查后，才能将对应接口勾选为完成。\n\n")

	currentModule := ""
	for _, item := range serverItems {
		if item.ModuleName != currentModule {
			currentModule = item.ModuleName
			b.WriteString("## " + currentModule + "\n\n")
		}
		key := interfaceKey(item.Method, item.FullPath)
		runapiStatus := "已存在"
		if targets[key] == "" {
			runapiStatus = "待复核"
		}
		fmt.Fprintf(&b, "- [ ] `%s %s` - RunAPI：[%s]；OpenAPI：[已补骨架]；Schema：[待确认]；错误：[待确认]；operator：[待取证]\n", item.Method, item.FullPath, runapiStatus)
	}
	return writeOutput(*output, []byte(b.String()))
}

func loadServerInterfaces(path string) ([]serverInterface, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []serverInterface
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func loadRunAPIInterfaces(path string) ([]runAPIInterface, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	var items []runAPIInterface
	for _, row := range rows[1:] {
		for len(row) < 10 {
			row = append(row, "")
		}
		method := strings.TrimSpace(row[4])
		if method == "" {
			if strings.EqualFold(row[3], "websocket2") {
				method = "GET"
			} else {
				continue
			}
		}
		items = append(items, runAPIInterface{
			TargetID: row[0], FolderPath: row[1], Name: row[2], TargetType: row[3],
			Method: method, Path: row[5], URL: row[6], MarkID: row[7], Version: row[8], DescriptionStatus: row[9],
		})
	}
	return items, nil
}

func loadChecklistTargets(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile("`([A-Z]+) ([^`]+)`.*?Target ID：`([^`]+)`")
	targets := map[string]string{}
	for _, match := range re.FindAllStringSubmatch(string(data), -1) {
		targets[interfaceKey(match[1], match[2])] = match[3]
	}
	return targets, nil
}

func loadOpenAPIKeys(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	keys := map[string]struct{}{}
	paths, _ := doc["paths"].(map[string]interface{})
	for path, raw := range paths {
		item, _ := raw.(map[string]interface{})
		for method := range item {
			keys[interfaceKey(method, openAPIPathToServerPath(path))] = struct{}{}
		}
	}
	return keys, nil
}

func buildOpenAPI(items []serverInterface, targets map[string]string) map[string]interface{} {
	tagsSeen := map[string]struct{}{}
	var tags []map[string]interface{}
	paths := map[string]interface{}{}
	operationIDs := map[string]int{}

	for _, item := range items {
		if _, ok := tagsSeen[item.ModuleName]; !ok {
			tagsSeen[item.ModuleName] = struct{}{}
			tags = append(tags, map[string]interface{}{"name": item.ModuleName})
		}
		path := serverPathToOpenAPIPath(item.FullPath)
		method := strings.ToLower(item.Method)
		pathItem, _ := paths[path].(map[string]interface{})
		if pathItem == nil {
			pathItem = map[string]interface{}{}
			paths[path] = pathItem
		}
		key := interfaceKey(item.Method, item.FullPath)
		operationID := uniqueOperationID(operationID(item), operationIDs)
		pathItem[method] = buildOperation(item, operationID, targets[key])
	}

	return map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "Disaster Server API",
			"version":     "0.1.0",
			"description": "disaster-server REST 与 WebSocket 接口契约。该文件由 server 路由清单与 RunAPI 清单生成骨架，逐接口 schema 需要按 operator 调用链继续补齐。",
		},
		"servers": []map[string]interface{}{
			{"url": "/", "description": "当前部署地址"},
		},
		"tags":  tags,
		"paths": paths,
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": commonSchemas(),
		},
	}
}

func buildOperation(item serverInterface, operationID string, targetID string) map[string]interface{} {
	isWebSocket := item.WebSocket == "是"
	operation := map[string]interface{}{
		"tags":               []string{item.ModuleName},
		"summary":            summaryFor(item),
		"description":        descriptionFor(item, isWebSocket),
		"operationId":        operationID,
		"parameters":         parametersFor(item.FullPath),
		"requestBody":        requestBodyFor(item),
		"responses":          responsesFor(isWebSocket),
		"security":           securityFor(item),
		"x-runapi-target-id": emptyAsPending(targetID),
		"x-controlled-resources": map[string]interface{}{
			"direct":     []string{emptyAsPending(item.DirectResource)},
			"downstream": []string{"待逐接口查询 operator controller 后补齐"},
			"scope":      "待逐接口查询 server 与 operator 调用链后补齐",
		},
		"x-operator-chain": map[string]interface{}{
			"serverHandler": item.Handler,
			"serverFile":    item.File,
			"crd":           "待逐接口确认",
			"controller":    "待逐接口确认",
		},
	}
	if isWebSocket {
		operation["x-disaster-protocol"] = "websocket"
		operation["x-message-schema"] = map[string]interface{}{"$ref": "#/components/schemas/WatchEnvelope"}
	}
	if strings.HasPrefix(item.FullPath, "/apis/") && !isWebSocket {
		operation["x-async-failure-status"] = map[string]interface{}{
			"statusFields": []string{"status.phase", "status.reason", "status.message"},
			"readVia":      []string{"GET 详情接口", "GET 列表接口", "WebSocket 事件流接口"},
		}
	}
	return operation
}

func commonSchemas() map[string]interface{} {
	return map[string]interface{}{
		"Envelope": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code":     map[string]interface{}{"type": "integer", "description": "业务码，0 表示成功"},
				"message":  map[string]interface{}{"type": "string", "description": "响应消息"},
				"data":     map[string]interface{}{"description": "业务数据，具体结构由接口决定"},
				"meta":     map[string]interface{}{"description": "分页、筛选、心跳、错误详情等元信息"},
				"trace_id": map[string]interface{}{"type": "string", "description": "请求追踪 ID"},
			},
			"required": []string{"code"},
		},
		"ErrorEnvelope": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code":     map[string]interface{}{"type": "integer", "description": "业务错误码"},
				"message":  map[string]interface{}{"type": "string", "description": "错误消息"},
				"data":     map[string]interface{}{"nullable": true, "description": "错误响应固定为空"},
				"meta":     map[string]interface{}{"description": "错误详情"},
				"trace_id": map[string]interface{}{"type": "string", "description": "请求追踪 ID"},
			},
			"required": []string{"code", "message"},
		},
		"CollectionMeta": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pagination": map[string]interface{}{"$ref": "#/components/schemas/PaginationMeta"},
				"sort":       map[string]interface{}{"type": "object"},
				"filters":    map[string]interface{}{"type": "object"},
				"links":      map[string]interface{}{"type": "object"},
				"type":       map[string]interface{}{"type": "string", "example": "collection"},
				"resourceType": map[string]interface{}{
					"type":        "string",
					"description": "集合资源类型",
				},
			},
		},
		"PaginationMeta": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"page":    map[string]interface{}{"type": "integer"},
				"limit":   map[string]interface{}{"type": "integer"},
				"total":   map[string]interface{}{"type": "integer"},
				"partial": map[string]interface{}{"type": "boolean"},
				"first":   map[string]interface{}{"type": "string"},
				"previous": map[string]interface{}{
					"type": "string",
				},
				"next": map[string]interface{}{"type": "string"},
				"last": map[string]interface{}{"type": "string"},
			},
		},
		"WatchEnvelope": map[string]interface{}{
			"allOf": []map[string]interface{}{
				{"$ref": "#/components/schemas/Envelope"},
				{"type": "object", "properties": map[string]interface{}{
					"data": map[string]interface{}{"$ref": "#/components/schemas/WatchEventDTO"},
				}},
			},
		},
		"WatchEventDTO": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type":   map[string]interface{}{"type": "string", "enum": []string{"ADDED", "MODIFIED", "DELETED", "ERROR"}},
				"object": map[string]interface{}{"description": "资源 DTO"},
			},
			"required": []string{"type"},
		},
		"GenericObject": map[string]interface{}{
			"type":                 "object",
			"additionalProperties": true,
			"description":          "待逐接口按 server DTO 与 operator 调用链补齐具体字段",
		},
	}
}

func parametersFor(path string) []map[string]interface{} {
	var params []map[string]interface{}
	seen := map[string]struct{}{}
	for _, segment := range strings.Split(path, "/") {
		if !strings.HasPrefix(segment, ":") {
			continue
		}
		name := strings.TrimPrefix(segment, ":")
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		params = append(params, map[string]interface{}{
			"name":        name,
			"in":          "path",
			"required":    true,
			"description": "路径参数 `" + name + "`，中文含义待逐接口调用链确认。",
			"schema":      map[string]interface{}{"type": "string"},
		})
	}
	return params
}

func requestBodyFor(item serverInterface) map[string]interface{} {
	description := "该接口不使用请求体。"
	required := false
	if item.Method == "POST" || item.Method == "PUT" || item.Method == "PATCH" {
		description = "请求体字段待逐接口按 server request struct 与 operator 调用链补齐。"
	}
	return map[string]interface{}{
		"required":    required,
		"description": description,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{"$ref": "#/components/schemas/GenericObject"},
			},
		},
	}
}

func responsesFor(websocket bool) map[string]interface{} {
	responses := map[string]interface{}{
		"200": responseRef("处理成功", "Envelope"),
		"400": responseRef("请求参数错误", "ErrorEnvelope"),
		"401": responseRef("认证失败", "ErrorEnvelope"),
		"403": responseRef("权限不足", "ErrorEnvelope"),
		"404": responseRef("资源不存在", "ErrorEnvelope"),
		"409": responseRef("资源冲突", "ErrorEnvelope"),
		"500": responseRef("服务端内部错误", "ErrorEnvelope"),
	}
	if websocket {
		responses["101"] = map[string]interface{}{"description": "WebSocket 连接升级成功"}
		delete(responses, "200")
	}
	return responses
}

func responseRef(description, schema string) map[string]interface{} {
	return map[string]interface{}{
		"description": description,
		"content": map[string]interface{}{
			"application/json": map[string]interface{}{
				"schema": map[string]interface{}{"$ref": "#/components/schemas/" + schema},
			},
		},
	}
}

func securityFor(item serverInterface) []map[string][]string {
	if item.Auth == "公开" {
		return []map[string][]string{}
	}
	return []map[string][]string{{"bearerAuth": []string{}}}
}

func descriptionFor(item serverInterface, websocket bool) string {
	var b strings.Builder
	b.WriteString("## 1. 接口用来干什么\n\n")
	b.WriteString("该接口对应 server handler `" + item.Handler + "`，用途需要在逐接口阶段结合 handler、request struct、response DTO 与 operator 调用链补齐。\n\n")
	b.WriteString("## 2. 控制哪些资源\n\n")
	b.WriteString("直接操作资源：\n- " + emptyAsPending(item.DirectResource) + "\n\n")
	b.WriteString("下层影响资源：\n- 待逐接口查询 operator controller 后补齐。\n\n")
	b.WriteString("资源作用范围：\n- 待逐接口查询 server 与 operator 调用链后补齐。\n\n")
	b.WriteString("## 3. 入参详细说明\n\n")
	b.WriteString("请求字段 schema 已先生成骨架，字段中文含义、可传入值、传入目的、是否必传、约束与默认值需要在逐接口阶段补齐。\n\n")
	b.WriteString("## 4. 返回详细说明\n\n")
	b.WriteString("返回字段 schema 已先生成统一信封骨架，具体 data 字段、字段来源、可能取值、为空条件需要在逐接口阶段补齐。\n\n")
	b.WriteString("## 5. 可能返回的错误\n\n")
	b.WriteString("server 当场返回错误已先列出通用响应码，具体触发条件需要在逐接口阶段补齐。")
	if websocket {
		b.WriteString(" 该接口为 WebSocket 连接，服务端推送消息使用 `Envelope` 包裹 `WatchEventDTO`。")
	} else if strings.HasPrefix(item.FullPath, "/apis/") {
		b.WriteString(" 接口成功后，operator 后续执行失败需要通过详情、列表以及事件流接口读取状态回写。")
	}
	return b.String()
}

func summaryFor(item serverInterface) string {
	if item.WebSocket == "是" {
		return "[WebSocket] " + item.ModuleName + " " + item.Method + " " + item.FullPath
	}
	return item.ModuleName + " " + item.Method + " " + item.FullPath
}

func operationID(item serverInterface) string {
	path := strings.Trim(item.FullPath, "/")
	replacer := strings.NewReplacer("/", "_", ".", "_", "-", "_", ":", "", "{", "", "}", "")
	base := strings.ToLower(item.Method + "_" + replacer.Replace(path))
	base = regexp.MustCompile(`_+`).ReplaceAllString(base, "_")
	return strings.Trim(base, "_")
}

func uniqueOperationID(base string, seen map[string]int) string {
	if seen[base] == 0 {
		seen[base] = 1
		return base
	}
	seen[base]++
	return fmt.Sprintf("%s_%d", base, seen[base])
}

func interfaceKey(method, path string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizePath(path)
	if method == "" || path == "" {
		return ""
	}
	return method + " " + path
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "{{baseurl}}")
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimRight(path, "/")
	if path == "" {
		path = "/"
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			segments[i] = ":" + strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
		}
		if strings.HasPrefix(segment, "{{") && strings.HasSuffix(segment, "}}") {
			segments[i] = ":" + strings.TrimSuffix(strings.TrimPrefix(segment, "{{"), "}}")
		}
	}
	return strings.Join(segments, "/")
}

func serverPathToOpenAPIPath(path string) string {
	var out []string
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, ":") {
			out = append(out, "{"+strings.TrimPrefix(segment, ":")+"}")
			continue
		}
		out = append(out, segment)
	}
	return strings.Join(out, "/")
}

func openAPIPathToServerPath(path string) string {
	var out []string
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			out = append(out, ":"+strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}"))
			continue
		}
		out = append(out, segment)
	}
	return strings.Join(out, "/")
}

func missingServerKeys(serverKeys map[string]serverInterface, openapiKeys map[string]struct{}) []string {
	var keys []string
	for key := range serverKeys {
		if _, ok := openapiKeys[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func missingRunAPIKeys(serverKeys map[string]serverInterface, runapiKeys map[string][]runAPIInterface) []string {
	var keys []string
	for key := range serverKeys {
		if _, ok := runapiKeys[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func extraRunAPIKeys(serverKeys map[string]serverInterface, runapiKeys map[string][]runAPIInterface) []string {
	var keys []string
	for key := range runapiKeys {
		if _, ok := serverKeys[key]; !ok && !matchesAnyServerPattern(key, serverKeys) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func matchesAnyServerPattern(key string, serverKeys map[string]serverInterface) bool {
	method, path := splitInterfaceKey(key)
	if method == "" || path == "" {
		return false
	}
	for serverKey := range serverKeys {
		serverMethod, serverPath := splitInterfaceKey(serverKey)
		if method != serverMethod {
			continue
		}
		if pathPatternMatches(serverPath, path) {
			return true
		}
	}
	return false
}

func pathPatternMatches(pattern, candidate string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	candidateParts := strings.Split(strings.Trim(candidate, "/"), "/")
	if len(patternParts) != len(candidateParts) {
		return false
	}
	for i := range patternParts {
		if strings.HasPrefix(patternParts[i], ":") {
			if candidateParts[i] == "" {
				return false
			}
			continue
		}
		if patternParts[i] != candidateParts[i] {
			return false
		}
	}
	return true
}

func splitInterfaceKey(key string) (string, string) {
	parts := strings.SplitN(key, " ", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func extraOpenAPIKeys(serverKeys map[string]serverInterface, openapiKeys map[string]struct{}) []string {
	var keys []string
	for key := range openapiKeys {
		if _, ok := serverKeys[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func emptyAsPending(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "待逐接口确认"
	}
	return value
}

func writeOutput(path string, data []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.MkdirAll(filepathDir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func filepathDir(path string) string {
	if idx := strings.LastIndex(path, string(os.PathSeparator)); idx >= 0 {
		return path[:idx]
	}
	return "."
}

func failf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
