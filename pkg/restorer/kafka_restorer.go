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
	"github.com/spf13/cobra"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log/slog"
	"strings"
)

type KafkaRestorer struct {
	Restorer

	skipCaSecrets   bool
	skipUserSecrets bool
	skipClusterID   bool
}

func NewKafkaRestorer(cmd *cobra.Command) (*KafkaRestorer, error) {
	restorer, err := NewRestorer(cmd)
	if err != nil {
		return nil, err
	}

	skipCaSecrets, err := cmd.Flags().GetBool("skip-ca-secrets")
	if err != nil {
		slog.Error("Failed to get the --skip-ca-secrets flag", "error", err)
		return nil, err
	}

	skipUserSecrets, err := cmd.Flags().GetBool("skip-user-secrets")
	if err != nil {
		slog.Error("Failed to get the --skip-user-secrets flag", "error", err)
		return nil, err
	}

	skipClusterId, err := cmd.Flags().GetBool("skip-cluster-id")
	if err != nil {
		slog.Error("Failed to get the --skip-cluster-id flag", "error", err)
		return nil, err
	}

	kafkaRestorer := &KafkaRestorer{
		Restorer:        *restorer,
		skipCaSecrets:   skipCaSecrets,
		skipUserSecrets: skipUserSecrets,
		skipClusterID:   skipClusterId,
	}

	return kafkaRestorer, nil
}

func (r *KafkaRestorer) RestoreKafka() error {
	var clusterId string // Is used later to restore the cluster ID

	for {
		r.gzipReader.Multistream(false)

		resources, err := io.ReadAll(r.gzipReader)
		if err != nil {
			slog.Error("Failed to read from the backup file", "error", err)
			return err
		}

		currentClusterId, err := r.restoreStream(r.gzipReader.Name, resources)
		if err != nil {
			return err
		}
		if currentClusterId != "" {
			clusterId = currentClusterId
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

	// We restore the Cluster ID only now to avoid the race condition from https://github.com/scholzj/strimzi-backup/issues/19
	if err := r.restoreKafkaClusterId(clusterId); err != nil {
		slog.Error("Failed to restore Kafka Cluster ID", "error", err)
		return err
	}

	if err := r.unpauseKafkaClusterAndWaitForReadiness(); err != nil {
		slog.Error("Failed to unpause Kafka cluster and get it into the Ready state", "error", err)
		return err
	}

	return nil
}

func (r *KafkaRestorer) restoreStream(streamName string, resources []byte) (string, error) {
	switch streamName {
	case backuper.KafkaFilename:
		slog.Info("Restoring paused Kafka resource")

		clusterId, err := r.restoreKafka(resources)
		if err != nil {
			slog.Error("Failed to restore Kafka resource", "error", err)
			return "", err
		}

		slog.Info("Kafka resource was restored in paused state")
		return clusterId, nil
	case backuper.CaSecretsFilename:
		if r.skipCaSecrets {
			slog.Warn("Skipping restoring CA Secrets")
			return "", nil
		}

		slog.Info("Restoring CA Secrets")
		if err := r.restoreCaSecrets(resources); err != nil {
			slog.Error("Failed to restore CA Secrets", "error", err)
			return "", err
		}
		slog.Info("CA Secrets were restored")
		return "", nil
	case backuper.KafkaNodePoolsFilename:
		slog.Info("Restoring Kafka Node Pools")
		if err := r.restoreKafkaNodePools(resources); err != nil {
			slog.Error("Failed to restore Kafka Node Pool resources", "error", err)
			return "", err
		}
		slog.Info("Kafka Node Pools were restored")
		return "", nil
	case backuper.KafkaUsersFilename:
		slog.Info("Restoring Kafka Users")
		if err := r.restoreKafkaUsers(resources); err != nil {
			slog.Error("Failed to restore Kafka Users resources", "error", err)
			return "", err
		}
		slog.Info("Kafka Users were restored")
		return "", nil
	case backuper.KafkaTopicsFilename:
		slog.Info("Restoring Kafka Topics")
		if err := r.restoreKafkaTopics(resources); err != nil {
			slog.Error("Failed to restore Kafka Topic resources", "error", err)
			return "", err
		}
		slog.Info("Kafka Topics were restored")
		return "", nil
	case backuper.KafkaUserSecretsFilename:
		if r.skipUserSecrets {
			slog.Warn("Skipping restoring Kafka User Secrets")
			return "", nil
		}

		slog.Info("Restoring Kafka User Secrets")
		if err := r.restoreSecrets(resources); err != nil {
			slog.Error("Failed to restore Kafka User Secrets", "error", err)
			return "", err
		}
		slog.Info("Kafka User Secrets were restored")
		return "", nil
	default:
		slog.Error("Unknown resources found in backup", "name", streamName)
		return "", fmt.Errorf("unknown resources %v found in backup", streamName)
	}
}

func (r *KafkaRestorer) restoreKafka(resource []byte) (string, error) {
	kafka, err := utils.DecodeResource(resource)
	if err != nil {
		slog.Error("Failed to unmarshall the Kafka resource", "error", err)
		return "", err
	}

	clusterId, _, err := unstructured.NestedString(kafka.Object, "status", "clusterId")
	if err != nil {
		slog.Error("Failed to get Kafka Cluster ID from the raw resource", "error", err)
		return "", err
	}

	kafka.SetNamespace(r.Namespace)
	kafka.SetName(r.Name)

	annotations := kafka.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{"strimzi.io/pause-reconciliation": "true"}
	} else {
		annotations["strimzi.io/pause-reconciliation"] = "true"
	}
	kafka.SetAnnotations(annotations)

	if err := r.createRawResource(utils.KafkaGVR, kafka); err != nil {
		slog.Error("Failed to restore the Kafka resource", "error", err)
		return "", err
	}

	// Wait for the paused reconciliation to be confirmed
	_, err = utils.WaitUntilKafkaReconciliationPaused(r.StrimziClient, r.Name, r.Namespace, r.Timeout)
	if err != nil {
		slog.Error("The Kafka resource was not paused. Please check the Cluster Operator logs for more details.", "error", err)
		return "", err
	}

	return clusterId, nil
}

func (r *KafkaRestorer) restoreKafkaClusterId(clusterId string) error {
	kafka, err := r.StrimziClient.KafkaV1().Kafkas(r.Namespace).Get(context.TODO(), r.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to restore the Kafka resource", "error", err)
		return err
	}

	if r.skipClusterID {
		slog.Warn("Skipping restoring Kafka Cluster ID")
	} else {
		// We restore the Cluster ID
		if clusterId != "" {
			slog.Info("Restoring Kafka Cluster ID", "clusterId", clusterId)
			kafkaWithClusterId := kafka.DeepCopy()
			kafkaWithClusterId.Status.ClusterId = clusterId

			if _, err := r.StrimziClient.KafkaV1().Kafkas(r.Namespace).UpdateStatus(context.TODO(), kafkaWithClusterId, metav1.UpdateOptions{}); err != nil {
				slog.Error("Failed to update the status of the Kafka resource and set the Cluster ID", "error", err)
				return err
			}
		} else {
			slog.Warn("Cannot restore Kafka Cluster ID as it is not present in the original Kafka resource")
		}
	}

	return nil
}

func (r *KafkaRestorer) unpauseKafkaClusterAndWaitForReadiness() error {
	kafka, err := r.StrimziClient.KafkaV1().Kafkas(r.Namespace).Get(context.TODO(), r.Name, metav1.GetOptions{})
	if err != nil {
		slog.Error("Failed to get the Kafka resource", "name", r.Name, "namespace", r.Namespace, "error", err)
		return err
	}

	if utils.IsKafkaReconciliationPaused(kafka) {
		slog.Info("Unpausing the Kafka cluster", "name", r.Name, "namespace", r.Namespace)
		unpausedKafka := kafka.DeepCopy()

		if unpausedKafka.Annotations == nil {
			unpausedKafka.Annotations = map[string]string{"strimzi.io/pause-reconciliation": "false"}
		} else {
			unpausedKafka.Annotations["strimzi.io/pause-reconciliation"] = "false"
		}

		_, err = r.StrimziClient.KafkaV1().Kafkas(r.Namespace).Update(context.TODO(), unpausedKafka, metav1.UpdateOptions{})
		if err != nil {
			slog.Error("Failed to unpause the Kafka resource", "name", r.Name, "namespace", r.Namespace, "error", err)
			return err
		}

		slog.Info("Waiting for the Kafka cluster to get ready", "name", r.Name, "namespace", r.Namespace)
		_, err = utils.WaitUntilKafkaReady(r.StrimziClient, r.Name, r.Namespace, r.Timeout)
		if err != nil {
			slog.Error("The Kafka cluster did not become ready. Please check the Cluster Operator logs for more details.", "name", r.Name, "namespace", r.Namespace, "error", err)
			return err
		}

		slog.Info("The Kafka cluster is ready", "name", r.Name, "namespace", r.Namespace)
	} else if utils.IsKafkaReady(kafka) {
		slog.Warn("The Kafka cluster is already ready and does not need to be unpaused", "name", r.Name, "namespace", r.Namespace)
	} else {
		slog.Warn("The Kafka cluster is not paused, but it is not ready. Waiting for the Kafka cluster to get ready.", "name", r.Name, "namespace", r.Namespace)
		_, err = utils.WaitUntilKafkaReady(r.StrimziClient, r.Name, r.Namespace, r.Timeout)
		if err != nil {
			slog.Error("The Kafka cluster did not become ready. Please check the Cluster Operator logs for more details.", "name", r.Name, "namespace", r.Namespace, "error", err)
			return err
		}

		slog.Info("The Kafka cluster is ready", "name", r.Name, "namespace", r.Namespace)
	}

	return nil
}

func (r *KafkaRestorer) restoreKafkaNodePools(resources []byte) error {
	nodePools, err := utils.DecodeResourceList(resources)
	if err != nil {
		slog.Error("Failed to unmarshall the Kafka Node Pool resources", "error", err)
		return err
	}

	for _, nodePool := range nodePools.Items {
		slog.Info("Restoring Kafka Node Pool", "name", nodePool.GetName(), "namespace", nodePool.GetNamespace())

		r.updateNamespaceAndClusterName(&nodePool)

		if err := r.createRawResource(utils.KafkaNodePoolGVR, &nodePool); err != nil {
			slog.Error("Failed to restore the Kafka Node Pool resource", "name", nodePool.GetName(), "namespace", nodePool.GetNamespace(), "error", err)
			return err
		}
	}

	return nil
}

func (r *KafkaRestorer) restoreKafkaUsers(resources []byte) error {
	users, err := utils.DecodeResourceList(resources)
	if err != nil {
		slog.Error("Failed to unmarshall the Kafka User resources", "error", err)
		return err
	}

	for _, user := range users.Items {
		slog.Info("Restoring Kafka User", "name", user.GetName(), "namespace", user.GetNamespace())

		r.updateNamespaceAndClusterName(&user)

		if err := r.createRawResource(utils.KafkaUserGVR, &user); err != nil {
			slog.Error("Failed to restore the Kafka User resource", "name", user.GetName(), "namespace", user.GetNamespace(), "error", err)
			return err
		}
	}

	return nil
}

func (r *KafkaRestorer) restoreKafkaTopics(resources []byte) error {
	topics, err := utils.DecodeResourceList(resources)
	if err != nil {
		slog.Error("Failed to unmarshall the Kafka Topic resources", "error", err)
		return err
	}

	for _, topic := range topics.Items {
		slog.Info("Restoring Kafka Topic", "name", topic.GetName(), "namespace", topic.GetNamespace())

		r.updateNamespaceAndClusterName(&topic)

		if err := r.createRawResource(utils.KafkaTopicGVR, &topic); err != nil {
			slog.Error("Failed to restore the Kafka Topic resource", "name", topic.GetName(), "namespace", topic.GetNamespace(), "error", err)
			return err
		}
	}

	return nil
}

func (r *KafkaRestorer) restoreCaSecrets(resources []byte) error {
	secrets, err := utils.DecodeResourceList(resources)
	if err != nil {
		slog.Error("Failed to unmarshall the CA Secret resources", "error", err)
		return err
	}

	for _, secret := range secrets.Items {
		slog.Info("Restoring CA Secret", "name", secret.GetName(), "namespace", secret.GetNamespace())

		// We have to update the names of the CA secrets so that they are reused when the cluster is renamed
		if strings.HasSuffix(secret.GetName(), "-cluster-ca") {
			secret.SetName(r.Name + "-cluster-ca")
		} else if strings.HasSuffix(secret.GetName(), "-cluster-ca-cert") {
			secret.SetName(r.Name + "-cluster-ca-cert")
		} else if strings.HasSuffix(secret.GetName(), "-clients-ca") {
			secret.SetName(r.Name + "-clients-ca")
		} else if strings.HasSuffix(secret.GetName(), "-clients-ca-cert") {
			secret.SetName(r.Name + "-clients-ca-cert")
		}

		r.updateNamespaceAndClusterName(&secret)

		if err := r.createRawResource(utils.SecretGVR, &secret); err != nil {
			slog.Error("Failed to restore the Secret", "name", secret.GetName(), "namespace", secret.GetNamespace(), "error", err)
			return err
		}
	}

	return nil
}

func (r *KafkaRestorer) restoreSecrets(resources []byte) error {
	secrets, err := utils.DecodeResourceList(resources)
	if err != nil {
		slog.Error("Failed to unmarshall the Secret resources", "error", err)
		return err
	}

	for _, secret := range secrets.Items {
		slog.Info("Restoring Secret", "name", secret.GetName(), "namespace", secret.GetNamespace())

		r.updateNamespaceAndClusterName(&secret)

		if err := r.createRawResource(utils.SecretGVR, &secret); err != nil {
			slog.Error("Failed to restore the Secret", "name", secret.GetName(), "namespace", secret.GetNamespace(), "error", err)
			return err
		}
	}

	return nil
}

//func (r *ConnectRestorer) Close() {
//	r.Restorer.Close()
//}
