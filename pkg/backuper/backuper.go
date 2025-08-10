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

package backuper

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/scholzj/strimzi-backup/pkg/utils"
	strimzi "github.com/scholzj/strimzi-go/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"log/slog"
	"os"
	"time"
)

type Backuper struct {
	KubernetesClient      *kubernetes.Clientset
	StrimziClient         *strimzi.Clientset
	Namespace             string
	Name                  string
	skipMetadataCleansing bool
	backupFile            *os.File
	bufferedWriter        *bufio.Writer
	gzipWriter            *gzip.Writer
}

func NewBackuper(cmd *cobra.Command) (*Backuper, error) {
	name := cmd.Flag("name").Value.String()
	if name == "" {
		slog.Error("--name option is required")
		return nil, fmt.Errorf("--name option is required")
	}

	kubeClient, strimziClient, namespace, err := utils.CreateKubernetesClients(cmd)
	if err != nil {
		slog.Error("Failed to create Kubernetes clients", "error", err)
		return nil, err
	}

	metadataCleansing, err := cmd.Flags().GetBool("skip-metadata-cleansing")
	if err != nil {
		slog.Error("Failed to get the --skip-metadata-cleansing flag", "error", err)
		return nil, err
	}

	backupFileName := cmd.Flag("filename").Value.String()
	if backupFileName == "" {
		backupFileName = "backup-" + time.Now().Format("2006-01-02-15-04-05") + ".gz"
	}
	backupFile, err := os.OpenFile(backupFileName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open backup file", "error", err, "file", backupFileName)
		return nil, err
	}

	bufferedWriter := bufio.NewWriter(backupFile)
	gzipWriter := gzip.NewWriter(bufferedWriter)

	backuper := Backuper{
		KubernetesClient:      kubeClient,
		StrimziClient:         strimziClient,
		Namespace:             namespace,
		Name:                  name,
		skipMetadataCleansing: metadataCleansing,
		backupFile:            backupFile,
		bufferedWriter:        bufferedWriter,
		gzipWriter:            gzipWriter,
	}

	return &backuper, nil
}

func (b *Backuper) Close() {
	if b.gzipWriter != nil {
		err := b.gzipWriter.Flush()
		if err != nil {
			slog.Error("Failed to flush the GZIP writer", "error", err)
		}

		err = b.gzipWriter.Close()
		if err != nil {
			slog.Error("Failed to close the GZIP writer", "error", err)
		}
	}

	if b.bufferedWriter != nil {
		err := b.bufferedWriter.Flush()
		if err != nil {
			slog.Error("Failed to flush the buffered writer", "error", err)
		}
	}

	if b.backupFile != nil {
		err := b.backupFile.Close()
		if err != nil {
			slog.Error("Failed to close the backup file", "error", err, "backupFile", b.backupFile.Name())
		}
	}
}
