package bootkube

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseManifests(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []manifest
	}{
		{
			name: "ingress",
			raw: `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: test-namespace
spec:
  rules:
  - http:
      paths:
      - path: /testpath
        backend:
          serviceName: test
          servicePort: 80
`,
			want: []manifest{
				{
					kind:       "Ingress",
					apiVersion: "extensions/v1beta1",
					namespace:  "test-namespace",
					name:       "test-ingress",
				},
			},
		},
		{
			name: "configmap",
			raw: `
apiVersion: v1
kind: ConfigMap
metadata:
  name: a-config
  namespace: default
data:
  color: "red"
  multi-line: |
    hello world
    how are you?
`,
			want: []manifest{
				{
					kind:       "ConfigMap",
					apiVersion: "v1",
					namespace:  "default",
					name:       "a-config",
				},
			},
		},
		{
			name: "two-resources",
			raw: `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  namespace: test-namespace
spec:
  rules:
  - http:
      paths:
      - path: /testpath
        backend:
          serviceName: test
          servicePort: 80
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: a-config
  namespace: default
data:
  color: "red"
  multi-line: |
    hello world
    how are you?
`,
			want: []manifest{
				{
					kind:       "Ingress",
					apiVersion: "extensions/v1beta1",
					namespace:  "test-namespace",
					name:       "test-ingress",
				},
				{
					kind:       "ConfigMap",
					apiVersion: "v1",
					namespace:  "default",
					name:       "a-config",
				},
			},
		},
		{
			name: "empty-manifests",
			raw: `
---
# Source: x
---
# Source: y
---
# Source: z
`,
			want: []manifest{
				{},
				{},
				{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseManifests(strings.NewReader(test.raw))
			if err != nil {
				t.Fatalf("failed to parse manifest: %v", err)
			}
			for i := range got {
				got[i].raw = nil
			}
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("wanted %#v, got %#v", test.want, got)
			}
		})
	}

}

func TestManifestURLPath(t *testing.T) {
	tests := []struct {
		apiVersion string
		namespace  string

		plural     string
		namespaced bool

		want string
	}{
		{"v1", "my-ns", "pods", true, "/api/v1/namespaces/my-ns/pods"},
		{"apps.k8s.io/v1beta1", "my-ns", "deployments", true, "/apis/apps.k8s.io/v1beta1/namespaces/my-ns/deployments"},
		{"v1", "", "nodes", false, "/api/v1/nodes"},
		{"apiextensions.k8s.io/v1beta1", "", "customresourcedefinitions", false, "/apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions"},
		// If non-namespaced, ignore the namespace field. This is to mimic kubectl create
		// behavior, which allows this but drops the namespace.
		{"apiextensions.k8s.io/v1beta1", "my-ns", "customresourcedefinitions", false, "/apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions"},
	}

	for _, test := range tests {
		m := manifest{
			apiVersion: test.apiVersion,
			namespace:  test.namespace,
		}
		got := m.urlPath(test.plural, test.namespaced)
		if test.want != got {
			t.Errorf("{&manifest{apiVersion:%q, namespace: %q}).urlPath(%q, %t); wanted=%q, got=%q",
				test.apiVersion, test.namespace, test.plural, test.namespaced, test.want, got)
		}
	}
}
