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

package restorer

import (
	"context"
	"fmt"
	"github.com/scholzj/strimzi-backup/pkg/backuper"
	"github.com/scholzj/strimzi-backup/pkg/utils"
	"github.com/scholzj/strimzi-go/pkg/apis/kafka.strimzi.io/v1beta2"
	"github.com/spf13/cobra"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"sigs.k8s.io/yaml"
)

type KafkaRestorer struct {
	Restorer
}

func NewKafkaRestorer(cmd *cobra.Command) (*KafkaRestorer, error) {
	restorer, err := NewRestorer(cmd)
	if err != nil {
		return nil, err
	}

	return &KafkaRestorer{Restorer: *restorer}, nil
}

func (r *KafkaRestorer) RestoreKafka() error {
	for {
		r.gzipReader.Multistream(false)

		resources, err := io.ReadAll(r.gzipReader)
		if err != nil {
			slog.Error("Failed to read from the backup file", "error", err)
			return err
		}

		switch r.gzipReader.Name {
		case backuper.KafkaFilename:
			slog.Info("Restoring paused Kafka resource")

			if err := r.restoreKafka(resources); err != nil {
				slog.Error("Failed to restore Kafka resource", "error", err)
				return err
			}

			slog.Info("Kafka resource was restored in paused state")

			break
		case backuper.CaSecretsFilename:
			slog.Info("Restoring CA Secrets")

			if err := r.restoreSecrets(resources); err != nil {
				slog.Error("Failed to restore CA Secrets", "error", err)
				return err
			}

			slog.Info("CA Secrets were restored")
			break
		case backuper.KafkaNodePoolsFilename:
			slog.Info("Restoring Kafka Node Pools")

			if err := r.restoreKafkaNodePools(resources); err != nil {
				slog.Error("Failed to restore Kafka Node Pool resources", "error", err)
				return err
			}

			slog.Info("Kafka Node Pools were restored")
			break
		case backuper.KafkaUsersFilename:
			slog.Info("Restoring Kafka Users")

			if err := r.restoreKafkaUsers(resources); err != nil {
				slog.Error("Failed to restore Kafka Users resources", "error", err)
				return err
			}

			slog.Info("Kafka USers were restored")
			break
		case backuper.KafkaTopicsFilename:
			slog.Info("Restoring Kafka Topics")

			if err := r.restoreKafkaTopics(resources); err != nil {
				slog.Error("Failed to restore Kafka Topic resources", "error", err)
				return err
			}

			slog.Info("Kafka Topics were restored")
			break
		case backuper.KafkaUserSecretsFilename:
			slog.Info("Restoring Kafka User Secrets")

			if err := r.restoreSecrets(resources); err != nil {
				slog.Error("Failed to restore Kafka User Secrets", "error", err)
				return err
			}

			slog.Info("Kafka User Secrets were restored")
			break
		default:
			slog.Error("Unknown resources found in backup", "name", r.gzipReader.Name, "comment", r.gzipReader.Comment, "modTime", r.gzipReader.ModTime)
			return fmt.Errorf("unknown resources %v found in backup", r.gzipReader.Name)
		}

		if err := r.gzipReader.Reset(r.bufferedReader); err != nil {
			if err == io.EOF {
				slog.Info("Restoring data completed")
				break
			} else {
				slog.Error("Failed to read the backup", "error", err)
				return err
			}
		}
	}

	if err := r.unpauseKafkaClusterAndWaitForReadiness(); err != nil {
		slog.Error("Failed to unpause Kafka cluster and get it into the Ready state", "error", err)
		return err
	}

	return nil
}

func (r *KafkaRestorer) restoreKafka(resource []byte) error {
	var kafka *v1beta2.Kafka

	if err := yaml.Unmarshal(resource, &kafka); err != nil {
		slog.Error("Failed to unmarshall the Kafka resource", "error", err)
		return err
	}

	// We update the metadata and pause the resource
	utils.CleanseMetadata(&kafka.ObjectMeta)
	kafka.Namespace = r.Namespace
	kafka.Name = r.Name
	if kafka.Annotations == nil {
		kafka.Annotations = map[string]string{"strimzi.io/pause-reconciliation": "true"}
	} else {
		kafka.Annotations["strimzi.io/pause-reconciliation"] = "true"
	}

	if _, err := r.StrimziClient.KafkaV1beta2().Kafkas(r.Namespace).Create(context.TODO(), kafka, metav1.CreateOptions{}); err != nil {
		slog.Error("Failed to restore the Kafka resource", "error", err)
		return err
	}

	// Wait for the paused reconciliation to be confirmed
	pausedKafka, err := utils.WaitUntilReconciliationPaused(r.StrimziClient, r.Name, r.Namespace, r.Timeout)
	if err != nil {
		slog.Error("The Kafka resource was not paused. Please check the Cluster Operator logs for more details.", "error", err)
		return err
	}

	// We restore the Cluster ID
	if kafka.Status != nil && kafka.Status.ClusterId != "" {
		slog.Info("Restoring Kafka Cluster ID", "clusterId", kafka.Status.ClusterId)
		pausedKafkaWithClusterId := pausedKafka.DeepCopy()
		pausedKafkaWithClusterId.Status.ClusterId = kafka.Status.ClusterId

		if _, err := r.StrimziClient.KafkaV1beta2().Kafkas(r.Namespace).UpdateStatus(context.TODO(), pausedKafkaWithClusterId, metav1.UpdateOptions{}); err != nil {
			slog.Error("Failed to update the status of the Kafka resource and set the Cluster ID", "error", err)
			return err
		}
	} else {
		slog.Warn("Cannot restore Kafka Cluster ID as it is not present in the original Kafka resource")
	}

	return nil
}

func (r *KafkaRestorer) unpauseKafkaClusterAndWaitForReadiness() error {
	kafka, err := r.StrimziClient.KafkaV1beta2().Kafkas(r.Namespace).Get(context.TODO(), r.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get the Kafka resource", "name", r.Name, "namespace", r.Namespace, "error", err)
		return err
	}

	if utils.IsReconciliationPaused(kafka) {
		slog.Info("Unpausing the Kafka cluster", "name", r.Name, "namespace", r.Namespace)
		unpausedKafka := kafka.DeepCopy()

		if unpausedKafka.Annotations == nil {
			unpausedKafka.Annotations = map[string]string{"strimzi.io/pause-reconciliation": "false"}
		} else {
			unpausedKafka.Annotations["strimzi.io/pause-reconciliation"] = "false"
		}

		_, err = r.StrimziClient.KafkaV1beta2().Kafkas(r.Namespace).Update(context.TODO(), unpausedKafka, metav1.UpdateOptions{})
		if err != nil {
			slog.Error("Failed to unpause the Kafka resource", "name", r.Name, "namespace", r.Namespace, "error", err)
			return err
		}

		slog.Info("Waiting for the Kafka cluster to get ready", "name", r.Name, "namespace", r.Namespace)
		_, err = utils.WaitUntilReady(r.StrimziClient, r.Name, r.Namespace, r.Timeout)
		if err != nil {
			slog.Error("The Kafka cluster did not become ready. Please check the Cluster Operator logs for more details.", "name", r.Name, "namespace", r.Namespace, "error", err)
			return err
		}

		slog.Info("The Kafka cluster is ready", "name", r.Name, "namespace", r.Namespace)
	} else if utils.IsReady(kafka) {
		slog.Warn("The Kafka cluster is already ready and does not need to be unpaused", "name", r.Name, "namespace", r.Namespace)
	} else {
		slog.Warn("The Kafka cluster is not paused, but it is not ready. Waiting for the Kafka cluster to get ready.", "name", r.Name, "namespace", r.Namespace)
		_, err = utils.WaitUntilReady(r.StrimziClient, r.Name, r.Namespace, r.Timeout)
		if err != nil {
			slog.Error("The Kafka cluster did not become ready. Please check the Cluster Operator logs for more details.", "name", r.Name, "namespace", r.Namespace, "error", err)
			return err
		}

		slog.Info("The Kafka cluster is ready", "name", r.Name, "namespace", r.Namespace)
	}

	return nil
}

func (r *KafkaRestorer) updateNamespaceAndClusterName(metadata *metav1.ObjectMeta) {
	metadata.Namespace = r.Namespace
	if metadata.Labels == nil {
		metadata.Labels = map[string]string{"strimzi.io/cluster": r.Name}
	} else {
		metadata.Labels["strimzi.io/cluster"] = r.Name
	}
}

func (r *KafkaRestorer) restoreKafkaNodePools(resources []byte) error {
	var nodePools *v1beta2.KafkaNodePoolList

	if err := yaml.Unmarshal(resources, &nodePools); err != nil {
		slog.Error("Failed to unmarshall the Kafka Node Pool resources", "error", err)
		return err
	}

	for _, nodePool := range nodePools.Items {
		slog.Info("Restoring Kafka Node Pool", "name", nodePool.Name, "namespace", nodePool.Namespace)

		utils.CleanseMetadata(&nodePool.ObjectMeta)
		r.updateNamespaceAndClusterName(&nodePool.ObjectMeta)

		if _, err := r.StrimziClient.KafkaV1beta2().KafkaNodePools(r.Namespace).Create(context.TODO(), &nodePool, metav1.CreateOptions{}); err != nil {
			slog.Error("Failed to restore the Kafka Node Pool resource", "name", nodePool.Name, "namespace", nodePool.Namespace, "error", err)
			return err
		}
	}

	return nil
}

func (r *KafkaRestorer) restoreKafkaUsers(resources []byte) error {
	var users *v1beta2.KafkaUserList

	if err := yaml.Unmarshal(resources, &users); err != nil {
		slog.Error("Failed to unmarshall the Kafka User resources", "error", err)
		return err
	}

	for _, user := range users.Items {
		slog.Info("Restoring Kafka User", "name", user.Name, "namespace", user.Namespace)

		utils.CleanseMetadata(&user.ObjectMeta)
		r.updateNamespaceAndClusterName(&user.ObjectMeta)

		if _, err := r.StrimziClient.KafkaV1beta2().KafkaUsers(r.Namespace).Create(context.TODO(), &user, metav1.CreateOptions{}); err != nil {
			slog.Error("Failed to restore the Kafka User resource", "name", user.Name, "namespace", user.Namespace, "error", err)
			return err
		}
	}

	return nil
}

func (r *KafkaRestorer) restoreKafkaTopics(resources []byte) error {
	var topics *v1beta2.KafkaTopicList

	if err := yaml.Unmarshal(resources, &topics); err != nil {
		slog.Error("Failed to unmarshall the Kafka Topic resources", "error", err)
		return err
	}

	for _, topic := range topics.Items {
		slog.Info("Restoring Kafka Topic", "name", topic.Name, "namespace", topic.Namespace)

		utils.CleanseMetadata(&topic.ObjectMeta)
		r.updateNamespaceAndClusterName(&topic.ObjectMeta)

		if _, err := r.StrimziClient.KafkaV1beta2().KafkaTopics(r.Namespace).Create(context.TODO(), &topic, metav1.CreateOptions{}); err != nil {
			slog.Error("Failed to restore the Kafka Topic resource", "name", topic.Name, "namespace", topic.Namespace, "error", err)
			return err
		}
	}

	return nil
}

func (r *KafkaRestorer) restoreSecrets(resources []byte) error {
	var secrets *v1.SecretList

	if err := yaml.Unmarshal(resources, &secrets); err != nil {
		slog.Error("Failed to unmarshall the Secret resources", "error", err)
		return err
	}

	for _, secret := range secrets.Items {
		slog.Info("Restoring Secret", "name", secret.Name, "namespace", secret.Namespace)

		utils.CleanseMetadata(&secret.ObjectMeta)
		r.updateNamespaceAndClusterName(&secret.ObjectMeta)

		if _, err := r.KubernetesClient.CoreV1().Secrets(r.Namespace).Create(context.TODO(), &secret, metav1.CreateOptions{}); err != nil {
			slog.Error("Failed to restore the Secret", "name", secret.Name, "namespace", secret.Namespace, "error", err)
			return err
		}
	}

	return nil
}

//func (r *KafkaRestorer) Close() {
//	r.Restorer.Close()
//}
