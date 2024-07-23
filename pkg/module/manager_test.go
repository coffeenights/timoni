package module

import (
	"context"
	"testing"
)

func Test_Main(t *testing.T) {
	ctx := context.Background()
	vals := map[string]interface{}{
		"values": map[string]interface{}{
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":    "100m",
					"memory": "100Mi",
				},
				"limits": map[string]interface{}{
					"cpu":    "100m",
					"memory": "100Mi",
				},
			},
		},
	}
	manager, err := NewManager(ctx, "podinfo", "oci://ghcr.io/stefanprodan/modules/podinfo", "latest", "test", "", vals)
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Build()
	if err != nil {
		t.Fatal(err)
	}
	err = manager.Apply()
	if err != nil {
		t.Fatal(err)
	}
}
