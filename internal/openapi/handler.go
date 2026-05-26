package openapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"sigs.k8s.io/yaml"
)

const specPath = "openspec/specs/disaster-server-openapi.yaml"

// Register mounts public Swagger/OpenAPI documentation routes.
func Register(s *server.Hertz) {
	s.GET("/openapi.yaml", serveYAML)
	s.GET("/openapi.json", serveJSON)
	s.GET("/swagger/", serveSwaggerUI)
}

func serveYAML(_ context.Context, c *app.RequestContext) {
	data, err := readSpec()
	if err != nil {
		c.JSON(consts.StatusNotFound, map[string]string{"message": err.Error()})
		return
	}
	c.Data(consts.StatusOK, "application/yaml; charset=utf-8", data)
}

func serveJSON(_ context.Context, c *app.RequestContext) {
	data, err := readSpec()
	if err != nil {
		c.JSON(consts.StatusNotFound, map[string]string{"message": err.Error()})
		return
	}
	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	var normalized interface{}
	if err := json.Unmarshal(jsonData, &normalized); err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	jsonData, err = json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	c.Data(consts.StatusOK, "application/json; charset=utf-8", jsonData)
}

func serveSwaggerUI(_ context.Context, c *app.RequestContext) {
	c.Data(consts.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
}

func readSpec() ([]byte, error) {
	if data, err := os.ReadFile(specPath); err == nil {
		return data, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(wd, specPath))
}

const swaggerHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Disaster Server API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #f6f8fa; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      persistAuthorization: true,
      displayRequestDuration: true
    });
  </script>
</body>
</html>`
