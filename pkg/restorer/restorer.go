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

package restorer

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
)

type Restorer struct {
	KubernetesClient *kubernetes.Clientset
	StrimziClient    *strimzi.Clientset
	Namespace        string
	Name             string
	Timeout          uint32
	backupFile       *os.File
	bufferedReader   *bufio.Reader
	gzipReader       *gzip.Reader
}

func NewRestorer(cmd *cobra.Command) (*Restorer, error) {
	name := cmd.Flag("name").Value.String()
	if name == "" {
		slog.Error("--name option is required")
		return nil, fmt.Errorf("--name option is required")
	}

	timeout, err := cmd.Flags().GetUint32("timeout")
	if err != nil {
		slog.Error("Failed to get the --timeout flag", "error", err)
		return nil, err
	}

	kubeClient, strimziClient, namespace, err := utils.CreateKubernetesClients(cmd)
	if err != nil {
		slog.Error("Failed to create Kubernetes clients", "error", err)
		return nil, err
	}

	backupFileName := cmd.Flag("filename").Value.String()
	backupFile, err := os.OpenFile(backupFileName, os.O_RDONLY, 0644)
	if err != nil {
		slog.Error("Failed to open file", "error", err, "file", backupFileName)
		return nil, err
	}

	bufferedReader := bufio.NewReader(backupFile)
	gzipReader, err := gzip.NewReader(bufferedReader)
	if err != nil {
		slog.Error("Failed to read file", "error", err, "file", backupFileName)
		return nil, err
	}

	restorer := Restorer{
		KubernetesClient: kubeClient,
		StrimziClient:    strimziClient,
		Namespace:        namespace,
		Name:             name,
		Timeout:          timeout,
		backupFile:       backupFile,
		bufferedReader:   bufferedReader,
		gzipReader:       gzipReader,
	}

	return &restorer, nil
}

func (r *Restorer) Close() {
	if r.gzipReader != nil {
		err := r.gzipReader.Close()
		if err != nil {
			slog.Error("Failed to close the GZIP reader", "error", err)
		}
	}

	if r.backupFile != nil {
		err := r.backupFile.Close()
		if err != nil {
			slog.Error("Failed to close the backup file", "error", err, "backupFile", r.backupFile.Name())
		}
	}
}
