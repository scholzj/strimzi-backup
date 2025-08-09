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
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/scholzj/strimzi-backup/pkg/utils"
	"github.com/scholzj/strimzi-go/pkg/apis/kafka.strimzi.io/v1beta2"
	strimzi "github.com/scholzj/strimzi-go/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log/slog"
	"os"
	"sigs.k8s.io/yaml"
	"time"
)

type Backuper struct {
	KubernetesClient  *kubernetes.Clientset
	StrimziClient     *strimzi.Clientset
	Namespace         string
	Name              string
	metadataCleansing bool
	backupFileName    string
	backupFile        *os.File
	bufferedWriter    *bufio.Writer
	gzipWriter        *gzip.Writer
}

func NewBackuper(cmd *cobra.Command) (*Backuper, error) {
	name := cmd.Flag("name").Value.String()
	if name == "" {
		slog.Error("--name option is required")
		return nil, fmt.Errorf("--name option is required")
	}

	kubeClient, strimziClient, namespace, err := utils.CreateKubernetesClients(cmd)
	if err != nil {
		slog.Error("Failed to create Kubernetes clients", "err", err)
		return nil, err
	}

	metadataCleansing, err := cmd.Flags().GetBool("enable-metadata-cleansing")
	if err != nil {
		slog.Error("Failed to get the --enable-metadata-cleansing flag", "err", err)
		return nil, err
	}

	backupFileName := cmd.Flag("filename").Value.String()
	if backupFileName == "" {
		backupFileName = "backup-" + time.Now().Format("2006-01-02-15-04-05") + ".gz"
	}
	backupFile, err := os.OpenFile(backupFileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open backup file", "err", err, "file", backupFileName)
		return nil, err
	}

	bufferedWriter := bufio.NewWriter(backupFile)
	gzipWriter := gzip.NewWriter(bufferedWriter)

	backup := Backuper{
		KubernetesClient:  kubeClient,
		StrimziClient:     strimziClient,
		Namespace:         namespace,
		Name:              name,
		metadataCleansing: metadataCleansing,
		backupFileName:    backupFileName,
		backupFile:        backupFile,
		bufferedWriter:    bufferedWriter,
		gzipWriter:        gzipWriter,
	}

	return &backup, nil
}

func (b *Backuper) BackupKafka() error {
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
		b.cleanseMetadata(&resource.ObjectMeta)
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

func (b *Backuper) BackupKafkaNodePools() error {
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

func (b *Backuper) BackupCaSecrets() error {
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

func (b *Backuper) BackupKafkaTopics() error {
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

func (b *Backuper) BackupKafkaUsers() error {
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

func (b *Backuper) BackupUserSecrets() error {
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

func (b *Backuper) cleanseSecretMetadata(resources *v1.SecretList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		b.cleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}

func (b *Backuper) cleanseKafkaNodePoolMetadata(resources *v1beta2.KafkaNodePoolList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		b.cleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}

func (b *Backuper) cleanseKafkaTopicMetadata(resources *v1beta2.KafkaTopicList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		b.cleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}

func (b *Backuper) cleanseKafkaUserMetadata(resources *v1beta2.KafkaUserList) {
	// We want to avoid copying the resource, so we use the index
	for i := range resources.Items {
		b.cleanseMetadata(&resources.Items[i].ObjectMeta)
	}
}

func (b *Backuper) cleanseMetadata(metadata *metav1.ObjectMeta) {
	//metadata.ResourceVersion = ""
	//metadata.CreationTimestamp = metav1.NewTime(time.Time{})
	metadata.ManagedFields = nil
	//metadata.Generation = 0
	//metadata.DeletionTimestamp = nil
	metadata.OwnerReferences = nil
	//metadata.DeletionGracePeriodSeconds = nil
	//metadata.UID = ""

	if metadata.Annotations != nil && metadata.Annotations["kubectl.kubernetes.io/last-applied-configuration"] != "" {
		delete(metadata.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
}

func (b *Backuper) Close() {
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
