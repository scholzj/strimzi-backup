package backuper

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"

	"github.com/scholzj/strimzi-backup/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestBackupKafkaPreservesUnknownFields(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), "backup-*.gz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	resource := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "Kafka",
		"metadata": map[string]interface{}{
			"name":            "my-cluster",
			"namespace":       "kafka",
			"resourceVersion": "12345",
		},
		"spec": map[string]interface{}{
			"kafka": map[string]interface{}{
				"version": "4.1.0",
				"unknown": map[string]interface{}{"future": "field"},
			},
		},
	}}
	resource.SetGroupVersionKind(schema.GroupVersionKind{Group: "kafka.strimzi.io", Version: "v1", Kind: "Kafka"})

	b := &KafkaBackuper{Backuper: Backuper{
		DynamicClient:  newFakeDynamicClient(resource),
		Namespace:      "kafka",
		Name:           "my-cluster",
		backupFile:     tempFile,
		bufferedWriter: bufio.NewWriter(tempFile),
		gzipWriter:     gzip.NewWriter(bufio.NewWriter(tempFile)),
	}}
	// Reuse the same buffered writer for the gzip writer.
	b.bufferedWriter = bufio.NewWriter(tempFile)
	b.gzipWriter = gzip.NewWriter(b.bufferedWriter)

	if err := b.BackupKafka(); err != nil {
		t.Fatalf("backup failed: %v", err)
	}
	if err := b.bufferedWriter.Flush(); err != nil {
		t.Fatalf("failed to flush backup file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("failed to close backup file: %v", err)
	}

	backupBytes, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(backupBytes))
	if err != nil {
		t.Fatalf("failed to open gzip backup: %v", err)
	}
	defer gzipReader.Close()

	decoded, err := utils.DecodeResource(mustReadAll(t, gzipReader))
	if err != nil {
		t.Fatalf("failed to decode backup YAML: %v", err)
	}

	value, found, err := unstructured.NestedString(decoded.Object, "spec", "kafka", "unknown", "future")
	if err != nil || !found {
		t.Fatalf("expected unknown field to be preserved, err=%v found=%v", err, found)
	}
	if value != "field" {
		t.Fatalf("expected unknown field value to be preserved, got %q", value)
	}
	if decoded.GetResourceVersion() != "" {
		t.Fatalf("expected metadata cleansing to remove resourceVersion, got %q", decoded.GetResourceVersion())
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
func mustReadAll(t *testing.T, reader io.Reader) []byte {
	t.Helper()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read data: %v", err)
	}
	return data
}
