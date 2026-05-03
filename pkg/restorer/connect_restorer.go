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
	"fmt"
	"github.com/scholzj/strimzi-backup/pkg/backuper"
	"github.com/scholzj/strimzi-backup/pkg/utils"
	"github.com/spf13/cobra"
	"io"
	"log/slog"
)

type ConnectRestorer struct {
	Restorer
}

func NewConnectRestorer(cmd *cobra.Command) (*ConnectRestorer, error) {
	restorer, err := NewRestorer(cmd)
	if err != nil {
		return nil, err
	}

	return &ConnectRestorer{Restorer: *restorer}, nil
}

func (r *ConnectRestorer) RestoreKafkaConnect() error {
	for {
		r.gzipReader.Multistream(false)

		resources, err := io.ReadAll(r.gzipReader)
		if err != nil {
			slog.Error("Failed to read from the backup file", "error", err)
			return err
		}

		switch r.gzipReader.Name {
		case backuper.ConnectFilename:
			slog.Info("Restoring Kafka Connect resource")

			err = r.restoreKafkaConnect(resources)
			if err != nil {
				slog.Error("Failed to restore Kafka resource", "error", err)
				return err
			}

			slog.Info("Kafka resource was restored")

			break
		case backuper.ConnectorsFilename:
			slog.Info("Restoring Kafka Connectors")

			if err := r.restoreKafkaConnectors(resources); err != nil {
				slog.Error("Failed to restore Kafka Connector resources", "error", err)
				return err
			}

			slog.Info("Kafka Connectors were restored")
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

	if err := r.waitForReadiness(); err != nil {
		slog.Error("The Kafka Connect cluster is not Ready", "error", err)
		return err
	}

	return nil
}

func (r *ConnectRestorer) restoreKafkaConnect(resource []byte) error {
	connect, err := utils.DecodeResource(resource)
	if err != nil {
		slog.Error("Failed to unmarshall the Kafka Connect resource", "error", err)
		return err
	}

	connect.SetNamespace(r.Namespace)
	connect.SetName(r.Name)

	if err := r.createRawResource(utils.KafkaConnectGVR, connect); err != nil {
		slog.Error("Failed to restore the Kafka Connect resource", "error", err)
		return err
	}

	return nil
}

func (r *ConnectRestorer) waitForReadiness() error {
	slog.Info("Waiting for the Kafka Connect cluster to get ready", "name", r.Name, "namespace", r.Namespace)

	if _, err := utils.WaitUntilConnectReady(r.StrimziClient, r.Name, r.Namespace, r.Timeout); err != nil {
		slog.Error("The Kafka Connect cluster did not become ready. Please check the Cluster Operator logs for more details.", "name", r.Name, "namespace", r.Namespace, "error", err)
		return err
	}

	slog.Info("The Kafka Connect cluster is ready", "name", r.Name, "namespace", r.Namespace)

	return nil
}

func (r *ConnectRestorer) restoreKafkaConnectors(resources []byte) error {
	connectors, err := utils.DecodeResourceList(resources)
	if err != nil {
		slog.Error("Failed to unmarshall the Kafka Connector resources", "error", err)
		return err
	}

	for _, connector := range connectors.Items {
		slog.Info("Restoring Kafka Connectors", "name", connector.GetName(), "namespace", connector.GetNamespace())

		r.updateNamespaceAndClusterName(&connector)

		if err := r.createRawResource(utils.KafkaConnectorGVR, &connector); err != nil {
			slog.Error("Failed to restore the Kafka Connector resource", "name", connector.GetName(), "namespace", connector.GetNamespace(), "error", err)
			return err
		}
	}

	return nil
}
