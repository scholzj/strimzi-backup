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
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
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
	b.startStream(ConnectFilename, "Connect cluster")

	slog.Info("Backing up the KafkaConnect resource", "name", b.Name)

	resource, err := b.DynamicClient.Resource(utils.KafkaConnectGVR).Namespace(b.Namespace).Get(context.TODO(), b.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get the KafkaConnect cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceMetadata(resource)
	}

	if err := b.writeResource(resource); err != nil {
		slog.Error("Failed to marshal the Kafka cluster to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the KafkaConnect resource complete", "name", b.Name)

	return nil
}

func (b *ConnectBackuper) BackupKafkaConnectors() error {
	b.startStream(ConnectorsFilename, "List of Kafka Connectors")

	slog.Info("Backing up the KafkaConnector resources", "labelSelector", "strimzi.io/cluster="+b.Name)

	resources, err := b.DynamicClient.Resource(utils.KafkaConnectorGVR).Namespace(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaConnector belonging to the Kafka Connect cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceListMetadata(resources)
	}

	if err := b.writeResourceList(resources); err != nil {
		slog.Error("Failed to marshal the KafkaConnectors to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the KafkaConnectors resources complete", "labelSelector", "strimzi.io/cluster="+b.Name)

	return nil
}
