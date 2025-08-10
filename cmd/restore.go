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

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore Strimzi-managed Apache Kafka operands",
	Long:  "Restore Strimzi-managed Apache Kafka operands",
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.PersistentFlags().String("kubeconfig", "", "Path to the kubeconfig file to use for Kubernetes API requests. If not specified, strimzi-backup will try to auto-detect the Kubernetes configuration.")
	restoreCmd.PersistentFlags().String("namespace", "", "Namespace of the cluster to restore. If not specified, defaults to the namespace from your Kubernetes configuration.")
	restoreCmd.PersistentFlags().String("name", "", "Name of the cluster to restore")
	restoreCmd.PersistentFlags().Uint32("timeout", 300000, "Timeout for how long to wait for the cluster to restore. In milliseconds.")
	restoreCmd.PersistentFlags().String("filename", "", "The name of the file to restore")
	_ = restoreCmd.MarkPersistentFlagRequired("filename")
}
