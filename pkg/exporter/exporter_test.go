package exporter

import (
	"bufio"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExportWritesEveryGzipStream(t *testing.T) {
	backupFile, err := os.CreateTemp(t.TempDir(), "backup-*.gz")
	if err != nil {
		t.Fatalf("failed to create backup file: %v", err)
	}

	bufferedWriter := bufio.NewWriter(backupFile)
	gzipWriter := gzip.NewWriter(bufferedWriter)

	writeStream := func(name string, content string) {
		t.Helper()
		gzipWriter.Reset(bufferedWriter)
		gzipWriter.Name = name
		if _, err := gzipWriter.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write gzip stream %s: %v", name, err)
		}
		if err := gzipWriter.Close(); err != nil {
			t.Fatalf("failed to close gzip stream %s: %v", name, err)
		}
	}

	writeStream("kafka.yaml", "kind: Kafka\n")
	writeStream("topics.yaml", "kind: KafkaTopicList\n")

	if err := bufferedWriter.Flush(); err != nil {
		t.Fatalf("failed to flush backup writer: %v", err)
	}
	if err := backupFile.Close(); err != nil {
		t.Fatalf("failed to close backup file: %v", err)
	}

	exportDir := filepath.Join(t.TempDir(), "exported")
	fileReader, err := os.Open(backupFile.Name())
	if err != nil {
		t.Fatalf("failed to open backup file for reading: %v", err)
	}
	defer fileReader.Close()

	bufferedReader := bufio.NewReader(fileReader)
	gzipReader, err := gzip.NewReader(bufferedReader)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}

	e := &Exporter{
		BackupFileName:  backupFile.Name(),
		ExportDirectory: exportDir,
		backupFile:      fileReader,
		bufferedReader:  bufferedReader,
		gzipReader:      gzipReader,
	}

	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		t.Fatalf("failed to create export dir: %v", err)
	}

	if err := e.Export(); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	kafkaBytes, err := os.ReadFile(filepath.Join(exportDir, "kafka.yaml"))
	if err != nil {
		t.Fatalf("failed to read first exported file: %v", err)
	}
	if string(kafkaBytes) != "kind: Kafka\n" {
		t.Fatalf("unexpected first stream content: %q", string(kafkaBytes))
	}

	topicBytes, err := os.ReadFile(filepath.Join(exportDir, "topics.yaml"))
	if err != nil {
		t.Fatalf("failed to read second exported file: %v", err)
	}
	if string(topicBytes) != "kind: KafkaTopicList\n" {
		t.Fatalf("unexpected second stream content: %q", string(topicBytes))
	}
}
