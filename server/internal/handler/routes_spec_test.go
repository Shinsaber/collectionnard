package handler_test

// Guards against drift between Router() and api/openapi.yaml: every route must
// be documented and every documented operation must be routed.

import (
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shinsaber/colec/internal/testutil"
	"gopkg.in/yaml.v3"
)

var pathParam = regexp.MustCompile(`\{[^}]+\}`)

// normalize replaces every path parameter by {} so that {id} and {collectionId}
// compare equal: OpenAPI and chi may name them differently.
func normalize(method, path string) string {
	return strings.ToUpper(method) + " " + pathParam.ReplaceAllString(path, "{}")
}

func TestRoutesMatchOpenAPI(t *testing.T) {
	raw, err := os.ReadFile("../../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec struct {
		Servers []struct {
			URL string `yaml:"url"`
		} `yaml:"servers"`
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if len(spec.Servers) == 0 {
		t.Fatal("spec has no servers entry")
	}
	base := spec.Servers[0].URL

	documented := map[string]bool{}
	for p, ops := range spec.Paths {
		for method := range ops {
			switch method {
			case "get", "post", "put", "delete", "patch":
				documented[normalize(method, base+p)] = true
			}
		}
	}

	routed := map[string]bool{}
	h := testutil.NewTestHandler(&testutil.MockStore{})
	err = chi.Walk(h.Router(), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimSuffix(route, "/")
		// chi's catch-all for served files corresponds to GET /uploads/{filename}.
		route = strings.Replace(route, "/*", "/{filename}", 1)
		routed[normalize(method, route)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	var missingInSpec, missingInCode []string
	for r := range routed {
		if !documented[r] {
			missingInSpec = append(missingInSpec, r)
		}
	}
	for d := range documented {
		if !routed[d] {
			missingInCode = append(missingInCode, d)
		}
	}
	sort.Strings(missingInSpec)
	sort.Strings(missingInCode)
	if len(missingInSpec) > 0 {
		t.Errorf("routes absentes de api/openapi.yaml :\n  %s", strings.Join(missingInSpec, "\n  "))
	}
	if len(missingInCode) > 0 {
		t.Errorf("opérations OpenAPI sans route dans Router() :\n  %s", strings.Join(missingInCode, "\n  "))
	}
	if len(documented) == 0 || len(routed) == 0 {
		t.Fatalf("documented=%d routed=%d", len(documented), len(routed))
	}
}
