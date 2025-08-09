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
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"log/slog"
	"os"
)

// restoreKafkaCmd represents the kafka command
var restoreKafkaCmd = &cobra.Command{
	Use:   "kafka",
	Short: "Restore Strimzi-based Apache Kafka cluster",
	Long:  "Restore Strimzi-based Apache Kafka cluster",
	Run: func(cmd *cobra.Command, args []string) {
		backupFileName := cmd.Flag("filename").Value.String()

		f, err := os.OpenFile(backupFileName, os.O_CREATE|os.O_RDONLY, 0644)
		if err != nil {
			slog.Error("Failed to open file", "err", err, "file", "backup.gz")
		}
		defer f.Close()

		b := bufio.NewReader(f)

		gzr, _ := gzip.NewReader(b)
		defer gzr.Close()

		for {
			gzr.Multistream(false)
			fmt.Printf("Name: %s\nComment: %s\nModTime: %s\n\n", gzr.Name, gzr.Comment, gzr.ModTime.UTC())

			if _, err := io.Copy(os.Stdout, gzr); err != nil {
				log.Fatal(err)
			}

			fmt.Print("\n\n")

			err = gzr.Reset(b)
			if err == io.EOF {
				fmt.Print("EOF\n\n")
				break
			}
			if err != nil {
				fmt.Print("Fatal\n\n")
				log.Fatal(err)
			}
		}

		fmt.Println("restore kafka called")
	},
}

func init() {
	restoreCmd.AddCommand(restoreKafkaCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// kafkaCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// kafkaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
