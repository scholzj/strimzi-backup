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
	skipCaSecrets   bool
	skipUserSecrets bool
	backupKafkaCmd  = &cobra.Command{
		Use:   "kafka",
		Short: "Backup Strimzi-based Apache Kafka cluster",
		Long:  "Backup Strimzi-based Apache Kafka cluster",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := backuper.NewKafkaBackuper(cmd)
			if err != nil {
				slog.Error("Failed to create backuper", "error", err)
				os.Exit(1)
			}
			defer b.Close()

			slog.Info("Starting backup of Kafka cluster", "name", b.Name, "namespace", b.Namespace)

			if err := b.BackupKafka(); err != nil {
				slog.Error("Failed to backup Kafka", "error", err)
				b.Discard()
				os.Exit(1)
			}

			if err := b.BackupKafkaNodePools(); err != nil {
				slog.Error("Failed to backup Kafka node pools", "error", err)
				b.Discard()
				os.Exit(1)
			}

			if !skipCaSecrets {
				if err := b.BackupCaSecrets(); err != nil {
					slog.Error("Failed to backup CA Secrets", "error", err)
					b.Discard()
					os.Exit(1)
				}
			}

			if err := b.BackupKafkaTopics(); err != nil {
				slog.Error("Failed to backup Kafka topics", "error", err)
				b.Discard()
				os.Exit(1)
			}

			if err := b.BackupKafkaUsers(); err != nil {
				slog.Error("Failed to backup Kafka users", "error", err)
				b.Discard()
				os.Exit(1)
			}

			if !skipUserSecrets {
				if err := b.BackupUserSecrets(); err != nil {
					slog.Error("Failed to backup User Secrets", "error", err)
					b.Discard()
					os.Exit(1)
				}
			}

			slog.Info("Backup of Kafka cluster is complete", "name", b.Name, "namespace", b.Namespace)
		},
	}
)

func init() {
	backupCmd.AddCommand(backupKafkaCmd)

	backupCmd.PersistentFlags().BoolVar(&skipCaSecrets, "skip-ca-secrets", false, "Skip backup of the Cluster and Client Certification Authority Secrets")
	backupCmd.PersistentFlags().BoolVar(&skipUserSecrets, "skip-user-secrets", false, "Skip backup of the Kafka User Secrets")
}
