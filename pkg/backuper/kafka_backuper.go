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
	strimziv1 "github.com/scholzj/strimzi-go/pkg/apis/kafka.strimzi.io/v1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log/slog"
)

type KafkaBackuper struct {
	Backuper
}

const (
	KafkaFilename            = "kafka.yaml"
	CaSecretsFilename        = "ca-secrets.yaml"
	KafkaNodePoolsFilename   = "kafka-node-pools.yaml"
	KafkaRebalancesFilename  = "kafka-rebalances.yaml"
	KafkaUsersFilename       = "kafka-users.yaml"
	KafkaTopicsFilename      = "kafka-topics.yaml"
	KafkaUserSecretsFilename = "kafka-user-secrets.yaml"
)

func NewKafkaBackuper(cmd *cobra.Command) (*KafkaBackuper, error) {
	backuper, err := NewBackuper(cmd)
	if err != nil {
		return nil, err
	}

	return &KafkaBackuper{Backuper: *backuper}, nil
}

func (b *KafkaBackuper) BackupKafka() error {
	b.startStream(KafkaFilename, "Kafka cluster")

	slog.Info("Backing up the Kafka resource", "name", b.Name)

	resource, err := b.DynamicClient.Resource(utils.KafkaGVR).Namespace(b.Namespace).Get(context.TODO(), b.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceMetadata(resource)
	}

	if err := b.writeResource(resource); err != nil {
		slog.Error("Failed to marshal the Kafka cluster to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the Kafka resource complete", "name", b.Name)

	return nil
}

func (b *KafkaBackuper) BackupKafkaNodePools() error {
	b.startStream(KafkaNodePoolsFilename, "List of Kafka Node Pools")

	slog.Info("Backing up the KafkaNodePool resources", "labelSelector", "strimzi.io/cluster="+b.Name)

	resources, err := b.DynamicClient.Resource(utils.KafkaNodePoolGVR).Namespace(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaNodePools belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceListMetadata(resources)
	}

	if err := b.writeResourceList(resources); err != nil {
		slog.Error("Failed to marshal the KafkaNodePools to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the KafkaNodePool resources complete", "labelSelector", "strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) BackupCaSecrets() error {
	b.startStream(CaSecretsFilename, "List of CA Secrets")

	slog.Info("Backing up the CA Secret resources", "labelSelector", "strimzi.io/component-type=certificate-authority,strimzi.io/cluster="+b.Name)

	resources, err := b.DynamicClient.Resource(utils.SecretGVR).Namespace(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/component-type=certificate-authority,strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get CA Secrets belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceListMetadata(resources)
	}

	if err := b.writeResourceList(resources); err != nil {
		slog.Error("Failed to marshal the CA Secrets to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the CA Secret resources complete", "labelSelector", "strimzi.io/component-type=certificate-authority,strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) BackupKafkaTopics() error {
	b.startStream(KafkaTopicsFilename, "List of Kafka Topics")

	slog.Info("Backing up the KafkaTopic resources", "labelSelector", "strimzi.io/cluster="+b.Name)

	resources, err := b.DynamicClient.Resource(utils.KafkaTopicGVR).Namespace(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaTopics belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceListMetadata(resources)
	}

	if err := b.writeResourceList(resources); err != nil {
		slog.Error("Failed to marshal the KafkaTopics to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the KafkaTopic resources complete", "labelSelector", "strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) BackupKafkaRebalances() error {
	b.startStream(KafkaRebalancesFilename, "List of Kafka Rebalance Templates")

	slog.Info("Backing up the KafkaRebalance template resources referenced from the Kafka cluster", "name", b.Name, "namespace", b.Namespace)

	kafka, err := b.StrimziClient.KafkaV1().Kafkas(b.Namespace).Get(context.TODO(), b.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get the Kafka cluster for resolving KafkaRebalance templates", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	templateNames, err := kafkaRebalanceTemplateNames(kafka)
	if err != nil {
		slog.Error("Failed to resolve KafkaRebalance template references from the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	resources := &unstructured.UnstructuredList{Object: map[string]interface{}{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "KafkaRebalanceList",
	}}

	for _, templateName := range templateNames {
		slog.Info("Backing up KafkaRebalance template", "name", templateName, "namespace", b.Namespace)

		resource, err := b.DynamicClient.Resource(utils.KafkaRebalanceGVR).Namespace(b.Namespace).Get(context.TODO(), templateName, metav1.GetOptions{})
		if err != nil {
			slog.Error("Failed to get a referenced KafkaRebalance template", "name", templateName, "namespace", b.Namespace, "error", err)
			return err
		}

		resources.Items = append(resources.Items, *resource)
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceListMetadata(resources)
	}

	if err := b.writeResourceList(resources); err != nil {
		slog.Error("Failed to marshal the KafkaRebalance templates to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the KafkaRebalance template resources complete", "count", len(resources.Items))

	return nil
}

func kafkaRebalanceTemplateNames(kafka *strimziv1.Kafka) ([]string, error) {
	if kafka.Spec == nil || kafka.Spec.CruiseControl == nil || len(kafka.Spec.CruiseControl.AutoRebalance) == 0 {
		return nil, nil
	}

	templateNames := make([]string, 0, len(kafka.Spec.CruiseControl.AutoRebalance))
	seen := make(map[string]struct{}, len(kafka.Spec.CruiseControl.AutoRebalance))

	for _, configuration := range kafka.Spec.CruiseControl.AutoRebalance {
		templateName := kafkaRebalanceTemplateName(configuration.Template)
		if templateName == "" {
			continue
		}

		if _, exists := seen[templateName]; exists {
			continue
		}

		seen[templateName] = struct{}{}
		templateNames = append(templateNames, templateName)
	}

	return templateNames, nil
}

func kafkaRebalanceTemplateName(template *corev1.LocalObjectReference) string {
	if template == nil {
		return ""
	}

	return template.Name
}

func (b *KafkaBackuper) BackupKafkaUsers() error {
	b.startStream(KafkaUsersFilename, "List of Kafka Users")

	slog.Info("Backing up the KafkaUser resources", "labelSelector", "strimzi.io/cluster="+b.Name)

	resources, err := b.DynamicClient.Resource(utils.KafkaUserGVR).Namespace(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get KafkaUsers belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceListMetadata(resources)
	}

	if err := b.writeResourceList(resources); err != nil {
		slog.Error("Failed to marshal the KafkaUsers to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the KafkaUser resources complete", "labelSelector", "strimzi.io/cluster="+b.Name)

	return nil
}

func (b *KafkaBackuper) BackupUserSecrets() error {
	b.startStream(KafkaUserSecretsFilename, "List of User Secrets")

	slog.Info("Backing up the User Secret resources", "labelSelector", "strimzi.io/kind=KafkaUser,strimzi.io/cluster="+b.Name)

	resources, err := b.DynamicClient.Resource(utils.SecretGVR).Namespace(b.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "strimzi.io/kind=KafkaUser,strimzi.io/cluster=" + b.Name})
	if err != nil {
		slog.Error("Failed to get User Secrets belonging to the Kafka cluster", "name", b.Name, "namespace", b.Namespace, "error", err)
		return err
	}

	if !b.skipMetadataCleansing {
		utils.CleanseResourceListMetadata(resources)
	}

	if err := b.writeResourceList(resources); err != nil {
		slog.Error("Failed to marshal the User Secrets to YAML", "error", err)
		return err
	}

	slog.Info("Backup of the User Secret resources complete", "labelSelector", "strimzi.io/kind=KafkaUser,strimzi.io/cluster="+b.Name)

	return nil
}

//func (b *ConnectBackuper) Close() {
//	b.Backuper.Close()
//}
