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

package backuper

import (
	"context"
	"github.com/scholzj/strimzi-backup/pkg/utils"
	"github.com/scholzj/strimzi-go/pkg/apis/kafka.strimzi.io/v1beta2"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"sigs.k8s.io/yaml"
	"time"
)

type ConnectBackuper struct {
	Backuper
}

const (
	ConnectFilename    = "connect.yaml"
	ConnectorsFilename = "connectors.yaml"
)

func NewConnectBackuper(cmd *cobra.Command) (*ConnectBackuper, error) {
	backuper, err := NewBackuper(cmd)
	if err != nil {
		return nil, err
	}

	return &ConnectBackuper{Backuper: *backuper}, nil
}

func (b *ConnectBackuper) BackupConnect() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = ConnectFilename
	b.gzipWriter.Comment = "Connect cluster"
	b.gzipWriter.ModTime = time.Now()

	slog.Info("Backing up the KafkaConnect resource", "name", b.Name)

	resource, err := b.StrimziClient.KafkaV1beta2().KafkaConnects(b.Namespace).Get(context.TODO(), b.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get the KafkaConnect cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		// Cleanse the metadata
		utils.CleanseMetadata(&resource.ObjectMeta)
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

	slog.Info("Backup of the KafkaConnect resource complete", "name", b.Name)

	return nil
}

func (b *ConnectBackuper) BackupKafkaConnectors() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = ConnectorsFilename
	b.gzipWriter.Comment = "List of Kafka Connectors"
	b.gzipWriter.ModTime = time.Now()

	slog.Info("Backing up the KafkaConnector resources", "labelSelector", "strimzi.io/cluster="+b.Name)

	resources, err := b.StrimziClient.KafkaV1beta2().KafkaConnectors(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaConnector belonging to the Kafka Connect cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		// Cleanse the metadata
		b.cleanseKafkaConnectorMetadata(resources)
	}

	resourcesYaml, err := yaml.Marshal(resources)
	if err != nil {
		slog.Error("Failed to marshal the KafkaConnectors to YAML", "error", err)
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

	slog.Info("Backup of the KafkaConnectors resources complete", "labelSelector", "strimzi.io/cluster="+b.Name)

	return nil
}

func (b *ConnectBackuper) cleanseKafkaConnectorMetadata(resources *v1beta2.KafkaConnectorList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		utils.CleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}
