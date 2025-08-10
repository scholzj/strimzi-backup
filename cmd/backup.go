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
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup Strimzi-managed Apache Kafka operands",
	Long:  "Backup Strimzi-managed Apache Kafka operands",
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.PersistentFlags().String("kubeconfig", "", "Path to the kubeconfig file to use for Kubernetes API requests. If not specified, strimzi-backup will try to auto-detect the Kubernetes configuration.")
	backupCmd.PersistentFlags().String("namespace", "", "Namespace of the cluster to backup. If not specified, defaults to the namespace from your Kubernetes configuration.")
	backupCmd.PersistentFlags().String("name", "", "Name of the cluster to backup")
	_ = backupCmd.MarkPersistentFlagRequired("name")
	backupCmd.PersistentFlags().String("filename", "", "The name of the resulting backup file")
	backupCmd.PersistentFlags().Bool("skip-metadata-cleansing", false, "Skips cleansing of metadata when creating the backup")
}
