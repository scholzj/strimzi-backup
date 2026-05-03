package backuper

import (
	"bufio"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scholzj/strimzi-backup/pkg/exporter"
	"github.com/scholzj/strimzi-backup/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestBackupArchiveRoundTripsThroughExport(t *testing.T) {
	backupFile, err := os.CreateTemp(t.TempDir(), "backup-*.gz")
	if err != nil {
		t.Fatalf("failed to create backup file: %v", err)
	}

	bufferedWriter := bufio.NewWriter(backupFile)
	b := &KafkaBackuper{Backuper: Backuper{
		DynamicClient:  newRoundTripFakeDynamicClient(newRoundTripKafkaResource(), newRoundTripKafkaTopicResource()),
		Namespace:      "kafka",
		Name:           "my-cluster",
		backupFile:     backupFile,
		bufferedWriter: bufferedWriter,
		gzipWriter:     gzip.NewWriter(bufferedWriter),
	}}

	if err := b.BackupKafka(); err != nil {
		t.Fatalf("failed to back up Kafka resource: %v", err)
	}
	if err := b.BackupKafkaTopics(); err != nil {
		t.Fatalf("failed to back up Kafka topics: %v", err)
	}
	if err := bufferedWriter.Flush(); err != nil {
		t.Fatalf("failed to flush archive: %v", err)
	}
	if err := backupFile.Close(); err != nil {
		t.Fatalf("failed to close backup file: %v", err)
	}

	exportDir := filepath.Join(t.TempDir(), "exported")
	cmd := &cobra.Command{}
	cmd.Flags().String("filename", backupFile.Name(), "")
	cmd.Flags().String("target-directory", exportDir, "")

	e, err := exporter.NewExporter(cmd)
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}
	defer e.Close()

	if err := e.Export(); err != nil {
		t.Fatalf("failed to export backup: %v", err)
	}

	kafkaBytes, err := os.ReadFile(filepath.Join(exportDir, KafkaFilename))
	if err != nil {
		t.Fatalf("failed to read exported Kafka resource: %v", err)
	}
	if !strings.Contains(string(kafkaBytes), "futureField:") || !strings.Contains(string(kafkaBytes), "preserved") {
		t.Fatalf("expected exported Kafka YAML to preserve unknown fields, got:\n%s", string(kafkaBytes))
	}
	if strings.Contains(string(kafkaBytes), "resourceVersion") {
		t.Fatalf("expected exported Kafka YAML to have cleansed metadata, got:\n%s", string(kafkaBytes))
	}

	topicsBytes, err := os.ReadFile(filepath.Join(exportDir, KafkaTopicsFilename))
	if err != nil {
		t.Fatalf("failed to read exported Kafka topics: %v", err)
	}
	if !strings.Contains(string(topicsBytes), "items:") || !strings.Contains(string(topicsBytes), "topic-a") {
		t.Fatalf("expected exported topic list to contain items, got:\n%s", string(topicsBytes))
	}
	if !strings.Contains(string(topicsBytes), "metadata:") || !strings.Contains(string(topicsBytes), "resourceVersion:") {
		t.Fatalf("expected exported topic list to preserve list metadata, got:\n%s", string(topicsBytes))
	}
	if strings.Contains(string(topicsBytes), "resourceVersion: \"123\"") {
		t.Fatalf("expected item metadata to be cleansed in exported topic list, got:\n%s", string(topicsBytes))
	}
}

func newRoundTripFakeDynamicClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
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

func newRoundTripKafkaResource() *unstructured.Unstructured {
	resource := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "Kafka",
		"metadata": map[string]interface{}{
			"name":            "my-cluster",
			"namespace":       "kafka",
			"resourceVersion": "123",
		},
		"spec": map[string]interface{}{
			"kafka": map[string]interface{}{
				"version":     "4.1.0",
				"futureField": "preserved",
			},
		},
	}}
	resource.SetGroupVersionKind(schema.GroupVersionKind{Group: "kafka.strimzi.io", Version: "v1", Kind: "Kafka"})
	return resource
}

func newRoundTripKafkaTopicResource() *unstructured.Unstructured {
	resource := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "KafkaTopic",
		"metadata": map[string]interface{}{
			"name":            "topic-a",
			"namespace":       "kafka",
			"resourceVersion": "123",
			"labels": map[string]interface{}{
				"strimzi.io/cluster": "my-cluster",
			},
		},
		"spec": map[string]interface{}{
			"partitions": int64(3),
		},
	}}
	resource.SetGroupVersionKind(schema.GroupVersionKind{Group: "kafka.strimzi.io", Version: "v1", Kind: "KafkaTopic"})
	return resource
}
