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

package cmd

import (
	"github.com/scholzj/strimzi-backup/pkg/backuper"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

var (
	backupConnectCmd = &cobra.Command{
		Use:   "connect",
		Short: "Backup Strimzi-based Apache Kafka Connect cluster",
		Long:  "Backup Strimzi-based Apache Kafka Connect cluster",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := backuper.NewConnectBackuper(cmd)
			if err != nil {
				slog.Error("Failed to create backuper", "error", err)
				os.Exit(1)
			}
			defer b.Close()

			slog.Info("Starting backup of Kafka Connect cluster", "name", b.Name, "namespace", b.Namespace)

			if err := b.BackupConnect(); err != nil {
				slog.Error("Failed to backup Kafka Connect", "error", err)
				b.Discard()
				os.Exit(1)
			}

			if err := b.BackupKafkaConnectors(); err != nil {
				slog.Error("Failed to backup Kafka Connectors", "error", err)
				b.Discard()
				os.Exit(1)
			}

			slog.Info("Backup of Kafka Connect cluster is complete", "name", b.Name, "namespace", b.Namespace)
		},
	}
)

func init() {
	backupCmd.AddCommand(backupConnectCmd)
}
