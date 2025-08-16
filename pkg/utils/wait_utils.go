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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"time"
)

func WaitUntilKafkaReady(client *strimzi.Clientset, name string, namespace string, timeout uint32) (*kafkaapi.Kafka, error) {
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
			k := event.Object.(*kafkaapi.Kafka)
			if IsKafkaReady(k) {
				return k, nil
			}
		case <-watchContext.Done():
			return nil, fmt.Errorf("timed out waiting for the Kafka cluster %s in namespace %s to be ready", name, namespace)
		}
	}
}

func WaitUntilConnectReady(client *strimzi.Clientset, name string, namespace string, timeout uint32) (*kafkaapi.KafkaConnect, error) {
	watchContext, watchContextCancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer watchContextCancel()

	watcher, err := client.KafkaV1beta2().KafkaConnects(namespace).Watch(watchContext, metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, name).String()})
	if err != nil {
		panic(err)
	}

	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():
			c := event.Object.(*kafkaapi.KafkaConnect)
			if IsConnectReady(c) {
				return c, nil
			}
		case <-watchContext.Done():
			return nil, fmt.Errorf("timed out waiting for the Kafka Connect cluster %s in namespace %s to be ready", name, namespace)
		}
	}
}

func IsKafkaReady(k *kafkaapi.Kafka) bool {
	if k.Status != nil && k.Status.Conditions != nil && len(k.Status.Conditions) > 0 {
		for _, condition := range k.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				if k.Status.ObservedGeneration == k.ObjectMeta.Generation {
					return true
				}
			}
		}

		return false
	} else {
		return false
	}
}

func IsConnectReady(c *kafkaapi.KafkaConnect) bool {
	if c.Status != nil && c.Status.Conditions != nil && len(c.Status.Conditions) > 0 {
		for _, condition := range c.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				if c.Status.ObservedGeneration == c.ObjectMeta.Generation {
					return true
				}
			}
		}

		return false
	} else {
		return false
	}
}

func WaitUntilKafkaReconciliationPaused(client *strimzi.Clientset, name string, namespace string, timeout uint32) (*kafkaapi.Kafka, error) {
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
			k := event.Object.(*kafkaapi.Kafka)
			if IsKafkaReconciliationPaused(k) {
				return k, nil
			}
		case <-watchContext.Done():
			return nil, fmt.Errorf("timed out waiting for the Kafka cluster %s in namespace %s to be paused", name, namespace)
		}
	}
}

func IsKafkaReconciliationPaused(k *kafkaapi.Kafka) bool {
	if k.Status != nil && k.Status.Conditions != nil && len(k.Status.Conditions) > 0 {
		for _, condition := range k.Status.Conditions {
			if condition.Type == "ReconciliationPaused" && condition.Status == "True" {
				return true
			}
		}

		return false
	} else {
		return false
	}
}
