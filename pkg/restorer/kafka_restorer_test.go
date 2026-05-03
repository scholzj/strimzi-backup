package restorer

import (
	"context"
	"testing"

	"github.com/scholzj/strimzi-backup/pkg/backuper"
	"github.com/scholzj/strimzi-backup/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestRestoreStreamSkipsKafkaUserSecretsOnlyWhenRequested(t *testing.T) {
	secrets := &unstructured.UnstructuredList{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "SecretList",
	}, Items: []unstructured.Unstructured{{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      "alice",
			"namespace": "source",
		},
		"data": map[string]interface{}{"password": "dGVzdA=="},
	}}}}

	encoded, err := utils.EncodeResourceList(secrets)
	if err != nil {
		t.Fatalf("failed to encode secrets: %v", err)
	}

	t.Run("skip-user-secrets blocks restore", func(t *testing.T) {
		r := newKafkaRestorerForTests()
		r.skipCaSecrets = false
		r.skipUserSecrets = true

		if _, err := r.restoreStream(backuper.KafkaUserSecretsFilename, encoded); err != nil {
			t.Fatalf("restoreStream failed: %v", err)
		}

		items, err := r.DynamicClient.Resource(utils.SecretGVR).Namespace(r.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list secrets: %v", err)
		}
		if len(items.Items) != 0 {
			t.Fatalf("expected user secrets to be skipped, got %d created", len(items.Items))
		}
	})

	t.Run("skip-ca-secrets does not block user secrets", func(t *testing.T) {
		r := newKafkaRestorerForTests()
		r.skipCaSecrets = true
		r.skipUserSecrets = false

		if _, err := r.restoreStream(backuper.KafkaUserSecretsFilename, encoded); err != nil {
			t.Fatalf("restoreStream failed: %v", err)
		}

		created, err := r.DynamicClient.Resource(utils.SecretGVR).Namespace(r.Namespace).Get(context.TODO(), "alice", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("expected user secret to be restored: %v", err)
		}
		if created.GetLabels()["strimzi.io/cluster"] != r.Name {
			t.Fatalf("expected cluster label rewrite, got %q", created.GetLabels()["strimzi.io/cluster"])
		}
	})
}

func TestRestoreCaSecretsRenamesClusterSpecificSecrets(t *testing.T) {
	r := newKafkaRestorerForTests()

	secrets := &unstructured.UnstructuredList{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "SecretList",
	}, Items: []unstructured.Unstructured{{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      "source-cluster-ca",
			"namespace": "source-ns",
			"labels": map[string]interface{}{
				"strimzi.io/cluster": "source-cluster",
			},
		},
		"data":   map[string]interface{}{"ca.crt": "dGVzdA=="},
		"status": map[string]interface{}{"ignored": true},
	}}}}

	encoded, err := utils.EncodeResourceList(secrets)
	if err != nil {
		t.Fatalf("failed to encode secrets: %v", err)
	}

	if err := r.restoreCaSecrets(encoded); err != nil {
		t.Fatalf("restoreCaSecrets failed: %v", err)
	}

	created, err := r.DynamicClient.Resource(utils.SecretGVR).Namespace(r.Namespace).Get(context.TODO(), r.Name+"-cluster-ca", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get renamed secret: %v", err)
	}
	if created.GetNamespace() != r.Namespace {
		t.Fatalf("expected namespace rewrite, got %q", created.GetNamespace())
	}
	if created.GetLabels()["strimzi.io/cluster"] != r.Name {
		t.Fatalf("expected cluster label rewrite, got %q", created.GetLabels()["strimzi.io/cluster"])
	}
	if _, found, _ := unstructured.NestedFieldNoCopy(created.Object, "status"); found {
		t.Fatalf("expected status to be stripped from restored secret")
	}
}

func newKafkaRestorerForTests() *KafkaRestorer {
	return &KafkaRestorer{Restorer: Restorer{
		DynamicClient: newKafkaFakeDynamicClient(),
		Namespace:     "target-ns",
		Name:          "target-cluster",
	}}
}

func newKafkaFakeDynamicClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
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
