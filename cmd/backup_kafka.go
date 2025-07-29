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
	"github.com/scholzj/strimzi-backup/pkg/backup"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

// backupKafkaCmd represents the kafka command
var backupKafkaCmd = &cobra.Command{
	Use:   "kafka",
	Short: "Backup Strimzi-based Apache Kafka cluster",
	Long:  "Backup Strimzi-based Apache Kafka cluster",
	Run: func(cmd *cobra.Command, args []string) {
		b, err := backup.NewBackup(cmd)
		if err != nil {
			slog.Error("Failed to create backup", "error", err)
			os.Exit(1)
		}
		defer b.Close()

		slog.Info("Starting backup of Kafka cluster", "name", b.Name, "namespace", b.Namespace)

		err = b.BackupKafka()
		if err != nil {
			slog.Error("Failed to backup Kafka", "error", err)
			panic(1)
		}

		err = b.BackupKafkaNodePools()
		if err != nil {
			slog.Error("Failed to backup Kafka node pools", "error", err)
			panic(1)
		}

		// TODO: Backup CA Secrets

		err = b.BackupKafkaTopics()
		if err != nil {
			slog.Error("Failed to backup Kafka topics", "error", err)
			panic(1)
		}

		err = b.BackupKafkaUsers()
		if err != nil {
			slog.Error("Failed to backup Kafka users", "error", err)
			panic(1)
		}

		// TODO: Backup user passwords

		slog.Info("Backup of Kafka cluster is complete", "name", b.Name, "namespace", b.Namespace)
	},
}

func init() {
	backupCmd.AddCommand(backupKafkaCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// kafkaCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// kafkaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
