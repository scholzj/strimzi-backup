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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"sigs.k8s.io/yaml"
	"time"
)

type KafkaBackuper struct {
	Backuper
}

func NewKafkaBackuper(cmd *cobra.Command) (*KafkaBackuper, error) {
	backuper, err := NewBackuper(cmd)
	if err != nil {
		return nil, err
	}

	return &KafkaBackuper{Backuper: *backuper}, nil
}

func (b *KafkaBackuper) BackupKafka() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "kafka.yaml"
	b.gzipWriter.Comment = "Kafka cluster"
	b.gzipWriter.ModTime = time.Now()

	slog.Info("Backing up the Kafka resource", "name", b.Name)

	resource, err := b.StrimziClient.KafkaV1beta2().Kafkas(b.Namespace).Get(context.TODO(), b.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if b.metadataCleansing {
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

	slog.Info("Backup of the Kafka resource complete", "name", b.Name)

	return nil
}

func (b *KafkaBackuper) BackupKafkaNodePools() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "pools.yaml"
	b.gzipWriter.Comment = "List of Kafka Node Pools"
	b.gzipWriter.ModTime = time.Now()

	slog.Info("Backing up the KafkaNodePool resources", "labelSelector", "strimzi.io/cluster="+b.Name)

	resources, err := b.StrimziClient.KafkaV1beta2().KafkaNodePools(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaNodePools belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if b.metadataCleansing {
		// Cleanse the metadata
		b.cleanseKafkaNodePoolMetadata(resources)
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

	slog.Info("Backup of the KafkaNodePool resources complete", "labelSelector", "strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) BackupCaSecrets() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "ca-secrets.yaml"
	b.gzipWriter.Comment = "List of CA Secrets"
	b.gzipWriter.ModTime = time.Now()

	slog.Info("Backing up the CA Secret resources", "labelSelector", "strimzi.io/component-type=certificate-authority,strimzi.io/cluster="+b.Name)

	resources, err := b.KubernetesClient.CoreV1().Secrets(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/component-type=certificate-authority,strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get CA Secrets belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if b.metadataCleansing {
		// Cleanse the Secret metadata
		b.cleanseSecretMetadata(resources)
	}

	resourcesYaml, err := yaml.Marshal(resources)
	if err != nil {
		slog.Error("Failed to marshal the CA Secrets to YAML", "error", err)
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

	slog.Info("Backup of the CA Secret resources complete", "labelSelector", "strimzi.io/component-type=certificate-authority,strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) BackupKafkaTopics() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "topics.yaml"
	b.gzipWriter.Comment = "List of Kafka Topics"
	b.gzipWriter.ModTime = time.Now()

	slog.Info("Backing up the KafkaTopic resources", "labelSelector", "strimzi.io/cluster="+b.Name)

	resources, err := b.StrimziClient.KafkaV1beta2().KafkaTopics(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaTopics belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if b.metadataCleansing {
		// Cleanse the metadata
		b.cleanseKafkaTopicMetadata(resources)
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

	slog.Info("Backup of the KafkaTopic resources complete", "labelSelector", "strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) BackupKafkaUsers() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "users.yaml"
	b.gzipWriter.Comment = "List of Kafka Users"
	b.gzipWriter.ModTime = time.Now()

	slog.Info("Backing up the KafkaUser resources", "labelSelector", "strimzi.io/cluster="+b.Name)

	resources, err := b.StrimziClient.KafkaV1beta2().KafkaUsers(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaUsers belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if b.metadataCleansing {
		// Cleanse the metadata
		b.cleanseKafkaUserMetadata(resources)
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

	slog.Info("Backup of the KafkaUser resources complete", "labelSelector", "strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) BackupUserSecrets() error {
	b.gzipWriter.Reset(b.bufferedWriter)
	b.gzipWriter.Name = "user-secrets.yaml"
	b.gzipWriter.Comment = "List of User Secrets"
	b.gzipWriter.ModTime = time.Now()

	slog.Info("Backing up the User Secret resources", "labelSelector", "strimzi.io/kind=KafkaUser,strimzi.io/cluster="+b.Name)

	resources, err := b.KubernetesClient.CoreV1().Secrets(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/kind=KafkaUser,strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get User Secrets belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if b.metadataCleansing {
		// Cleanse the Secret metadata
		b.cleanseSecretMetadata(resources)
	}

	resourcesYaml, err := yaml.Marshal(resources)
	if err != nil {
		slog.Error("Failed to marshal the User Secrets to YAML", "error", err)
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

	slog.Info("Backup of the User Secret resources complete", "labelSelector", "strimzi.io/kind=KafkaUser,strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) cleanseSecretMetadata(resources *v1.SecretList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		utils.CleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}

func (b *KafkaBackuper) cleanseKafkaNodePoolMetadata(resources *v1beta2.KafkaNodePoolList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		utils.CleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}

func (b *KafkaBackuper) cleanseKafkaTopicMetadata(resources *v1beta2.KafkaTopicList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		utils.CleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}

func (b *KafkaBackuper) cleanseKafkaUserMetadata(resources *v1beta2.KafkaUserList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		utils.CleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}

//func (b *KafkaBackuper) Close() {
//	b.Backuper.Close()
//}
