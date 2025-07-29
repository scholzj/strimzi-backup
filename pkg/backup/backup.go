/*
Copyright © 2025 Jakub Scholz

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backup

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/scholzj/strimzi-backup/pkg/utils"
	strimzi "github.com/scholzj/strimzi-go/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log/slog"
	"os"
	"sigs.k8s.io/yaml"
	"time"
)

type Backup struct {
	KubernetesClient *kubernetes.Clientset
	StrimziClient    *strimzi.Clientset
	Namespace        string
	Name             string
	backupFileName   string
	backupFile       *os.File
	bufferedWriter   *bufio.Writer
	gzipWriter       *gzip.Writer
}

func NewBackup(cmd *cobra.Command) (*Backup, error) {
	name := cmd.Flag("name").Value.String()
	if name == "" {
		slog.Error("--name option is required")
		return nil, fmt.Errorf("--name option is required")
	}

	// TODO: Make backup file configurable

	kubeClient, strimziClient, namespace, err := utils.CreateKubernetesClients(cmd)
	if err != nil {
		slog.Error("Failed to create Kubernetes clients", "err", err)
		return nil, err
	}

	backupFileName := "backup-" + time.DateOnly + "-" + time.TimeOnly + ".gz"
	backupFile, err := os.OpenFile(backupFileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open backup file", "err", err, "file", backupFileName)
		return nil, err
	}

	bufferedWriter := bufio.NewWriter(backupFile)
	gzipWriter := gzip.NewWriter(bufferedWriter)

	backup := Backup{
		KubernetesClient: kubeClient,
		StrimziClient:    strimziClient,
		Namespace:        namespace,
		Name:             name,
		backupFileName:   backupFileName,
		backupFile:       backupFile,
		bufferedWriter:   bufferedWriter,
		gzipWriter:       gzipWriter,
	}

	return &backup, nil
}

func (b *Backup) BackupKafka() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "kafka.yaml"
	b.gzipWriter.Comment = "Kafka cluster"
	b.gzipWriter.ModTime = time.Now()

	resource, err := b.StrimziClient.KafkaV1beta2().Kafkas(b.Namespace).Get(context.TODO(), b.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	resourceYaml, err := yaml.Marshal(resource)
	if err != nil {
		slog.Error("Failed to marshal the Kafka cluster to YAML", "error", err)
		return err
	}

	_, err = b.gzipWriter.Write(resourceYaml)
	if err != nil {
		slog.Error("Failed to write the YAML to the backup file", "error", err)
		return err
	}

	err = b.gzipWriter.Close()
	if err != nil {
		slog.Error("Failed to close the GZIP writer when resetting the stream", "error", err)
		return err
	}

	return nil
}

func (b *Backup) BackupKafkaNodePools() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "pools.yaml"
	b.gzipWriter.Comment = "List of Kafka Node Pools"
	b.gzipWriter.ModTime = time.Now()

	resources, err := b.StrimziClient.KafkaV1beta2().KafkaNodePools(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaNodePools belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	resourcesYaml, err := yaml.Marshal(resources)
	if err != nil {
		slog.Error("Failed to marshal the KafkaNodePools to YAML", "error", err)
		return err
	}

	_, err = b.gzipWriter.Write(resourcesYaml)
	if err != nil {
		slog.Error("Failed to write the YAML to the backup file", "error", err)
		return err
	}

	err = b.gzipWriter.Close()
	if err != nil {
		slog.Error("Failed to close the GZIP writer when resetting the stream", "error", err)
		return err
	}

	return nil
}

func (b *Backup) BackupKafkaTopics() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "topics.yaml"
	b.gzipWriter.Comment = "List of Kafka Topics"
	b.gzipWriter.ModTime = time.Now()

	resources, err := b.StrimziClient.KafkaV1beta2().KafkaTopics(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaTopics belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	resourcesYaml, err := yaml.Marshal(resources)
	if err != nil {
		slog.Error("Failed to marshal the KafkaTopics to YAML", "error", err)
		return err
	}

	_, err = b.gzipWriter.Write(resourcesYaml)
	if err != nil {
		slog.Error("Failed to write the YAML to the backup file", "error", err)
		return err
	}

	err = b.gzipWriter.Close()
	if err != nil {
		slog.Error("Failed to close the GZIP writer when resetting the stream", "error", err)
		return err
	}

	return nil
}

func (b *Backup) BackupKafkaUsers() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "users.yaml"
	b.gzipWriter.Comment = "List of Kafka Users"
	b.gzipWriter.ModTime = time.Now()

	resources, err := b.StrimziClient.KafkaV1beta2().KafkaUsers(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaUsers belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	resourcesYaml, err := yaml.Marshal(resources)
	if err != nil {
		slog.Error("Failed to marshal the KafkaUsers to YAML", "error", err)
		return err
	}

	_, err = b.gzipWriter.Write(resourcesYaml)
	if err != nil {
		slog.Error("Failed to write the YAML to the backup file", "error", err)
		return err
	}

	err = b.gzipWriter.Close()
	if err != nil {
		slog.Error("Failed to close the GZIP writer when resetting the stream", "error", err)
		return err
	}

	return nil
}

func (b *Backup) Close() {
	if b.gzipWriter != nil {
		err := b.gzipWriter.Flush()
		if err != nil {
			slog.Error("Failed to flush the GZIP writer", "error", err)
		}

		err = b.gzipWriter.Close()
		if err != nil {
			slog.Error("Failed to close the GZIP writer", "error", err)
		}
	}

	if b.bufferedWriter != nil {
		err := b.bufferedWriter.Flush()
		if err != nil {
			slog.Error("Failed to flush the buffered writer", "error", err)
		}
	}

	if b.backupFile != nil {
		err := b.backupFile.Close()
		if err != nil {
			slog.Error("Failed to close the backup file", "error", err, "backupFile", b.backupFile.Name())
		}
	}

}
