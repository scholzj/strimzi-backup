package utils

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCleanseResourceMetadata(t *testing.T) {
	grace := int64(30)
	resource := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "Kafka",
		"metadata": map[string]interface{}{
			"name":            "my-cluster",
			"namespace":       "source",
			"resourceVersion": "12345",
			"generation":      int64(7),
			"uid":             "abc",
			"managedFields":   []interface{}{map[string]interface{}{"manager": "kubectl"}},
			"ownerReferences": []interface{}{map[string]interface{}{"name": "owner"}},
			"annotations": map[string]interface{}{
				"kubectl.kubernetes.io/last-applied-configuration": "x",
				"keep": "yes",
			},
		},
	}}
	resource.SetCreationTimestamp(metav1.Now())
	resource.SetDeletionGracePeriodSeconds(&grace)

	CleanseResourceMetadata(resource)

	if resource.GetResourceVersion() != "" {
		t.Fatalf("expected empty resourceVersion, got %q", resource.GetResourceVersion())
	}
	if resource.GetGeneration() != 0 {
		t.Fatalf("expected generation to be cleared, got %d", resource.GetGeneration())
	}
	if resource.GetUID() != "" {
		t.Fatalf("expected uid to be cleared, got %q", resource.GetUID())
	}
	if !resource.GetCreationTimestamp().Time.IsZero() {
		t.Fatalf("expected creation timestamp to be cleared, got %v", resource.GetCreationTimestamp())
	}
	if resource.GetDeletionGracePeriodSeconds() != nil {
		t.Fatalf("expected deletion grace period to be cleared")
	}
	if len(resource.GetOwnerReferences()) != 0 {
		t.Fatalf("expected owner references to be cleared")
	}
	if _, found, _ := unstructured.NestedFieldNoCopy(resource.Object, "metadata", "managedFields"); found {
		t.Fatalf("expected managedFields to be removed")
	}
	annotations := resource.GetAnnotations()
	if annotations["kubectl.kubernetes.io/last-applied-configuration"] != "" {
		t.Fatalf("expected last-applied annotation to be removed")
	}
	if annotations["keep"] != "yes" {
		t.Fatalf("expected non-kubectl annotation to be preserved")
	}
}

func TestEncodeDecodeResourcePreservesUnknownFields(t *testing.T) {
	resource := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "KafkaConnect",
		"metadata": map[string]interface{}{
			"name":      "connect",
			"namespace": "source",
		},
		"spec": map[string]interface{}{
			"replicas": int64(3),
			"unknown": map[string]interface{}{
				"future": "value",
			},
		},
	}}

	encoded, err := EncodeResource(resource)
	if err != nil {
		t.Fatalf("failed to encode resource: %v", err)
	}

	decoded, err := DecodeResource(encoded)
	if err != nil {
		t.Fatalf("failed to decode resource: %v", err)
	}

	value, found, err := unstructured.NestedString(decoded.Object, "spec", "unknown", "future")
	if err != nil || !found {
		t.Fatalf("expected unknown field to exist, err=%v found=%v", err, found)
	}
	if value != "value" {
		t.Fatalf("expected unknown field to round-trip, got %q", value)
	}
}

func TestUpdateNamespaceAndClusterName(t *testing.T) {
	resource := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "topic-a",
			"namespace": "old-ns",
			"labels": map[string]interface{}{
				"strimzi.io/cluster": "old",
				"keep":               "yes",
			},
		},
	}}

	UpdateNamespaceAndClusterName(resource, "new-ns", "new-cluster")

	if resource.GetNamespace() != "new-ns" {
		t.Fatalf("expected namespace to be rewritten, got %q", resource.GetNamespace())
	}
	labels := resource.GetLabels()
	if labels["strimzi.io/cluster"] != "new-cluster" {
		t.Fatalf("expected cluster label to be rewritten, got %q", labels["strimzi.io/cluster"])
	}
	if labels["keep"] != "yes" {
		t.Fatalf("expected unrelated labels to be preserved")
	}
}

func TestEncodeDecodeResourceListPreservesMetadata(t *testing.T) {
	resources := &unstructured.UnstructuredList{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "KafkaTopicList",
		"metadata": map[string]interface{}{
			"continue":        "token-1",
			"resourceVersion": "999",
		},
	}, Items: []unstructured.Unstructured{{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "KafkaTopic",
		"metadata": map[string]interface{}{
			"name":      "topic-a",
			"namespace": "kafka",
		},
	}}}}

	encoded, err := EncodeResourceList(resources)
	if err != nil {
		t.Fatalf("failed to encode resource list: %v", err)
	}

	decoded, err := DecodeResourceList(encoded)
	if err != nil {
		t.Fatalf("failed to decode resource list: %v", err)
	}

	if decoded.GetResourceVersion() != "999" {
		t.Fatalf("expected list resourceVersion to round-trip, got %q", decoded.GetResourceVersion())
	}
	continueValue, found, err := unstructured.NestedString(decoded.Object, "metadata", "continue")
	if err != nil || !found || continueValue != "token-1" {
		t.Fatalf("expected list metadata.continue to round-trip, got value=%q found=%v err=%v", continueValue, found, err)
	}
}
