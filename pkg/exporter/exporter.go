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

package exporter

import (
	"bufio"
	"compress/gzip"
	"github.com/spf13/cobra"
	"io"
	"log/slog"
	"os"
)

type Exporter struct {
	BackupFileName  string
	ExportDirectory string
	backupFile      *os.File
	bufferedReader  *bufio.Reader
	gzipReader      *gzip.Reader
}

func NewExporter(cmd *cobra.Command) (*Exporter, error) {
	backupFileName := cmd.Flag("filename").Value.String()
	exportDirectory := cmd.Flag("target-directory").Value.String()

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

	if err := os.MkdirAll(exportDirectory, 0755); err != nil {
		slog.Error("Failed to create target directory", "error", err, "directory", exportDirectory)
		return nil, err
	}

	exporter := Exporter{
		BackupFileName:  backupFileName,
		ExportDirectory: exportDirectory,
		backupFile:      backupFile,
		bufferedReader:  bufferedReader,
		gzipReader:      gzipReader,
	}

	return &exporter, nil
}

func (e *Exporter) Export() error {
	for {
		e.gzipReader.Multistream(false)
		slog.Info("Exporting data", "name", e.gzipReader.Name, "comment", e.gzipReader.Comment, "modTime", e.gzipReader.ModTime)

		exportFilename := e.ExportDirectory + "/" + e.gzipReader.Name
		exportFile, err := os.OpenFile(exportFilename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			slog.Error("Failed to open export file", "error", err, "file", exportFilename)
			return err
		}

		bufferedWriter := bufio.NewWriter(exportFile)

		if _, err := io.Copy(bufferedWriter, e.gzipReader); err != nil {
			slog.Error("Failed to export data", "error", err, "file", exportFilename)
			return err
		}

		if err := e.gzipReader.Reset(e.bufferedReader); err != nil {
			if err == io.EOF {
				slog.Info("Exporting data completed", "name", exportFilename)

				// Cleanup after the exported file
				if err := bufferedWriter.Flush(); err != nil {
					slog.Error("Failed to flush writer", "error", err, "file", exportFilename)
					return err
				}
				if err := exportFile.Close(); err != nil {
					slog.Error("Failed to close export file", "error", err, "file", exportFilename)
					return err
				}

				break
			} else {
				slog.Error("Failed to read the backup", "error", err)
				return err
			}
		}
	}

	return nil
}

func (e *Exporter) Close() {
	if e.gzipReader != nil {
		err := e.gzipReader.Close()
		if err != nil {
			slog.Error("Failed to close the GZIP reader", "error", err)
		}
	}

	if e.backupFile != nil {
		err := e.backupFile.Close()
		if err != nil {
			slog.Error("Failed to close the backup file", "error", err, "backupFile", e.backupFile.Name())
		}
	}
}
