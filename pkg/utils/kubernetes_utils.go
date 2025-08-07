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

package utils

import (
	"context"
	"fmt"
	kafkaapi "github.com/scholzj/strimzi-go/pkg/apis/kafka.strimzi.io/v1beta2"
	strimzi "github.com/scholzj/strimzi-go/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func CreateKubernetesClients(cmd *cobra.Command) (*kubernetes.Clientset, *strimzi.Clientset, string, error) {
	kubeConfigFlag := cmd.Flag("kubeconfig").Value.String()
	namespaceFlag := cmd.Flag("namespace").Value.String()

	kubeConfig, kubeConfigNamespace, err := tryToFindKubeConfigAndCurrentNamespace(kubeConfigFlag)
	if err != nil {
		return nil, nil, "", err
	}

	namespace, err := determineNamespaceFromOptionOrKubeConfig(namespaceFlag, kubeConfigNamespace)
	if err != nil {
		return nil, nil, "", err
	}

	kubeClient, err := createKubernetesClient(kubeConfig)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
		return nil, nil, "", err
	}

	strimziClient, err := createStrimziClient(kubeConfig)
	if err != nil {
		log.Fatalf("Failed to create Strimzi client: %v", err)
		return nil, nil, "", err
	}

	return kubeClient, strimziClient, namespace, nil
}

func createKubernetesClient(kubeConfig *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(kubeConfig)
}

func createStrimziClient(kubeConfig *rest.Config) (*strimzi.Clientset, error) {
	return strimzi.NewForConfig(kubeConfig)
}

func tryToFindKubeConfigAndCurrentNamespace(kubeConfigOption string) (*rest.Config, string, error) {
	kubeConfigPath := tryToFindKubeConfigPath(kubeConfigOption)

	if kubeConfigPath != "" {
		// Create the config
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to instantiate Kubernetes configuration from %v: %v", kubeConfigPath, err)
		}

		// Try to get the namespace -> we might not need it, so we silence the errors
		var namespace string
		fileConfig, err := clientcmd.LoadFromFile(kubeConfigPath)
		if err != nil {
			slog.Debug("Failed to parse Kubernetes client configuration to get default namespace from kubeconfig file", "error", err)
		} else {
			ns := fileConfig.Contexts[fileConfig.CurrentContext].Namespace

			if ns != "" {
				namespace = fileConfig.Contexts[fileConfig.CurrentContext].Namespace
			}
		}

		return config, namespace, nil
	} else {
		// Create the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, "", fmt.Errorf("failed to instantiate Kubernetes in-cluster configuration: %v", err)
		}

		// Try to get the namespace -> we might not need it, so we silence the errors
		var namespace string
		namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			slog.Debug("Failed to read namespace from /var/run/secrets/kubernetes.io/serviceaccount/namespace", "error", err)
			namespace = ""
		} else {
			namespace = string(namespaceBytes)
		}

		return config, namespace, nil
	}
}

func tryToFindKubeConfigPath(kubeConfigOption string) string {
	if kubeConfigOption == "" {
		var kubeConfigPath string

		if os.Getenv("KUBECONFIG") != "" {
			kubeConfigPath = os.Getenv("KUBECONFIG")
			slog.Info("Using kubeconfig from KUBECONFIG environment variable", "kubeConfigPath", kubeConfigPath)
			return kubeConfigPath
		} else if home := homedir.HomeDir(); home != "" {
			homeKubeConfigPath := filepath.Join(home, ".kube", "config")
			_, err := os.Stat(homeKubeConfigPath)
			if err == nil {
				kubeConfigPath = homeKubeConfigPath
				slog.Info("Using kubeconfig from home directory", "kubeConfigPath", kubeConfigPath)
			}
		} else {
			log.Printf("Could not find Kubernetes configuration file. In-cluster configuration will be used.")
			slog.Info("Could not find Kubernetes configuration file. In-cluster configuration will be used.")
		}

		return kubeConfigPath
	} else {
		return kubeConfigOption
	}
}

func determineNamespaceFromOptionOrKubeConfig(namespaceOption string, kubeConfigNamespace string) (string, error) {
	if namespaceOption != "" {
		return namespaceOption, nil
	} else if kubeConfigNamespace != "" {
		return kubeConfigNamespace, nil
	} else {
		return "", fmt.Errorf("namespace has to be specified using the --namespace option or as part of the Kubernetes client configuration")
	}
}

func waitUntilReady(client *strimzi.Clientset, name string, namespace string, timeout uint32) (bool, error) {
	watchContext, watchContextCancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer watchContextCancel()

	watcher, err := client.KafkaV1beta2().Kafkas(namespace).Watch(watchContext, metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, name).String()})
	if err != nil {
		panic(err)
	}

	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():
			if isReady(event.Object.(*kafkaapi.Kafka)) {
				return true, nil
			}
		case <-watchContext.Done():
			return false, fmt.Errorf("timed out waiting for the Kafka cluster %s in namespace %s to be ready", name, namespace)
		}
	}
}

func isReady(k *kafkaapi.Kafka) bool {
	if k.Status != nil && k.Status.Conditions != nil && len(k.Status.Conditions) > 0 {
		for _, condition := range k.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				if k.Status.ObservedGeneration == k.ObjectMeta.Generation {
					//log.Print("The Kafka cluster is ready and up-to-date")
					return true
				}
			}
		}

		//log.Print("The Kafka cluster has conditions but is not ready")
		return false
	} else {
		//log.Print("The Kafka cluster has no conditions")
		return false
	}
}

func waitUntilReconciliationPaused(client *strimzi.Clientset, name string, namespace string, timeout uint32) (bool, error) {
	watchContext, watchContextCancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer watchContextCancel()

	watcher, err := client.KafkaV1beta2().Kafkas(namespace).Watch(watchContext, metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, name).String()})
	if err != nil {
		panic(err)
	}

	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():
			if isReconciliationPaused(event.Object.(*kafkaapi.Kafka)) {
				return true, nil
			}
		case <-watchContext.Done():
			return false, fmt.Errorf("timed out waiting for the Kafka cluster %s in namespace %s to be paused", name, namespace)
		}
	}
}

func isReconciliationPaused(k *kafkaapi.Kafka) bool {
	if k.Status != nil && k.Status.Conditions != nil && len(k.Status.Conditions) > 0 {
		for _, condition := range k.Status.Conditions {
			if condition.Type == "ReconciliationPaused" && condition.Status == "True" {
				//log.Print("The Kafka cluster is ready and up-to-date")
				return true
			}
		}

		//log.Print("The Kafka cluster has conditions but is not ready")
		return false
	} else {
		//log.Print("The Kafka cluster has no conditions")
		return false
	}
}
