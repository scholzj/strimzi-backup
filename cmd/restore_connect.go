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
	"github.com/scholzj/strimzi-backup/pkg/restorer"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

var restoreConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Restore Strimzi-based Apache Kafka Connect cluster",
	Long:  "Restore Strimzi-based Apache Kafka Connect cluster",
	Run: func(cmd *cobra.Command, args []string) {
		r, err := restorer.NewConnectRestorer(cmd)
		if err != nil {
			slog.Error("Failed to create restorer", "error", err)
			os.Exit(1)
		}
		defer r.Close()

		slog.Info("Starting restoration of Kafka Connect cluster", "name", r.Name, "namespace", r.Namespace)

		if err := r.RestoreKafkaConnect(); err != nil {
			slog.Error("Failed to restore the Kafka Connect cluster", "name", r.Name, "namespace", r.Namespace, "error", err)
			panic(1)
		}

		slog.Info("Kafka Connect cluster was restored", "name", r.Name, "namespace", r.Namespace)
	},
}

func init() {
	restoreCmd.AddCommand(restoreConnectCmd)
}
