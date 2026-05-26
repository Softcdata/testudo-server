package policy

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (h *PolicyHandler) Register() {
	path := fmt.Sprintf("policies.%s", v1.GroupVersion.String())
	g := h.Rg.Group(path).Use(h.Mw...)
	g.GET("/policies", h.policies)
	g.GET("/policies/names", h.policyNames)
	g.GET("/policies/:name", h.policy)
	g.POST("/policies", h.createPolicy)
	g.PUT("/policies/:name", h.updatePolicy)
	g.DELETE("/policies/:name", h.deletePolicy)
}
