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
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows a version of the Strimzi Backup application",
	Long:  `Shows a version of the Strimzi Backup application.`,
	Run: func(cmd *cobra.Command, args []string) {
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			slog.Error("Failed to get Strimzi Backup version information")
			os.Exit(1)
		} else {
			slog.Info("Strimzi Backup version: " + buildInfo.Main.Version)
			slog.Info("Go version: " + buildInfo.GoVersion)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
