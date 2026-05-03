package restorer

import (
	"context"
	"testing"

	"github.com/scholzj/strimzi-backup/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestRestoreKafkaConnectPreservesUnknownFieldsAndStripsStatus(t *testing.T) {
	r := &ConnectRestorer{Restorer: Restorer{
		DynamicClient: newFakeDynamicClient(),
		Namespace:     "target-ns",
		Name:          "target-name",
	}}

	resource := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "KafkaConnect",
		"metadata": map[string]interface{}{
			"name":      "source-name",
			"namespace": "source-ns",
		},
		"spec": map[string]interface{}{
			"replicas": int64(2),
			"unknown":  map[string]interface{}{"future": "field"},
		},
		"status": map[string]interface{}{
			"conditions": []interface{}{map[string]interface{}{"type": "Ready", "status": "True"}},
		},
	}}

	encoded, err := utils.EncodeResource(resource)
	if err != nil {
		t.Fatalf("failed to encode resource: %v", err)
	}

	if err := r.restoreKafkaConnect(encoded); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	created, err := r.DynamicClient.Resource(utils.KafkaConnectGVR).Namespace("target-ns").Get(context.TODO(), "target-name", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get restored resource: %v", err)
	}

	if created.GetNamespace() != "target-ns" || created.GetName() != "target-name" {
		t.Fatalf("expected name/namespace rewrite, got %s/%s", created.GetNamespace(), created.GetName())
	}
	value, found, err := unstructured.NestedString(created.Object, "spec", "unknown", "future")
	if err != nil || !found || value != "field" {
		t.Fatalf("expected unknown spec field to be preserved, got value=%q found=%v err=%v", value, found, err)
	}
	if _, found, _ := unstructured.NestedFieldNoCopy(created.Object, "status"); found {
		t.Fatalf("expected status to be stripped before restore")
	}
}

func newFakeDynamicClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		utils.KafkaGVR:          "KafkaList",
		utils.KafkaNodePoolGVR:  "KafkaNodePoolList",
		utils.KafkaTopicGVR:     "KafkaTopicList",
		utils.KafkaUserGVR:      "KafkaUserList",
		utils.KafkaConnectGVR:   "KafkaConnectList",
		utils.KafkaConnectorGVR: "KafkaConnectorList",
		utils.SecretGVR:         "SecretList",
	}, objects...)
}
