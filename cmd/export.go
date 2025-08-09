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
	"github.com/scholzj/strimzi-backup/pkg/exporter"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Exports all YAMLs from the backup",
	Long:  `Exports all Kubernetes resources from the backup file into separate files by their type`,
	Run: func(cmd *cobra.Command, args []string) {
		e, err := exporter.NewExporter(cmd)
		if err != nil {
			slog.Error("Failed to export backup", "error", err)
			os.Exit(1)
		}
		defer e.Close()

		slog.Info("Starting export or backup", "filename", e.BackupFileName, "target-directory", e.ExportDirectory)

		err = e.Export()
		if err != nil {
			slog.Error("Failed to export the backup", "error", err)
			panic(1)
		}

		slog.Info("Export of backup is complete", "filename", e.BackupFileName, "target-directory", e.ExportDirectory)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.PersistentFlags().String("filename", "", "The name of the file to be exported to files")
	_ = exportCmd.MarkPersistentFlagRequired("filename")

	exportCmd.PersistentFlags().String("target-directory", "", "The directory where the files should be exported")
	_ = exportCmd.MarkPersistentFlagRequired("target-directory")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// exportCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// exportCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
