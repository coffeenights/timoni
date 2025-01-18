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

func TestManager_ApplyObject(t *testing.T) {
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

	applySets, err := manager.GetApplySets()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ApplySets: %v", applySets)
	for _, applySet := range applySets {
		for _, obj := range applySet.Objects {
			if obj.GetKind() == "Service" {
				_, err = manager.ApplyObject(obj, false)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
