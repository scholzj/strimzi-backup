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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

var (
	KafkaGVR          = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1", Resource: "kafkas"}
	KafkaNodePoolGVR  = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1", Resource: "kafkanodepools"}
	KafkaRebalanceGVR = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1", Resource: "kafkarebalances"}
	KafkaTopicGVR     = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1", Resource: "kafkatopics"}
	KafkaUserGVR      = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1", Resource: "kafkausers"}
	KafkaConnectGVR   = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1", Resource: "kafkaconnects"}
	KafkaConnectorGVR = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1", Resource: "kafkaconnectors"}
	SecretGVR         = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
)

func CleanseResourceMetadata(resource *unstructured.Unstructured) {
	resource.SetResourceVersion("")
	resource.SetCreationTimestamp(metav1.Time{})
	resource.SetGeneration(0)
	resource.SetDeletionTimestamp(nil)
	resource.SetDeletionGracePeriodSeconds(nil)
	resource.SetOwnerReferences(nil)
	resource.SetUID("")

	annotations := resource.GetAnnotations()
	if annotations != nil && annotations["kubectl.kubernetes.io/last-applied-configuration"] != "" {
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		if len(annotations) == 0 {
			resource.SetAnnotations(nil)
		} else {
			resource.SetAnnotations(annotations)
		}
	}

	unstructured.RemoveNestedField(resource.Object, "metadata", "managedFields")
}

func RemoveStatus(resource *unstructured.Unstructured) {
	unstructured.RemoveNestedField(resource.Object, "status")
}

func UpdateNamespaceAndClusterName(resource *unstructured.Unstructured, namespace string, name string) {
	resource.SetNamespace(namespace)

	labels := resource.GetLabels()
	if labels == nil {
		labels = map[string]string{"strimzi.io/cluster": name}
	} else {
		labels["strimzi.io/cluster"] = name
	}
	resource.SetLabels(labels)
}

func EncodeResource(resource *unstructured.Unstructured) ([]byte, error) {
	return yaml.Marshal(resource.Object)
}

func EncodeResourceList(resources *unstructured.UnstructuredList) ([]byte, error) {
	serialized := make(map[string]interface{}, len(resources.Object)+1)
	for key, value := range resources.Object {
		serialized[key] = value
	}

	items := make([]interface{}, 0, len(resources.Items))
	for i := range resources.Items {
		items = append(items, resources.Items[i].Object)
	}
	serialized["items"] = items

	return yaml.Marshal(serialized)
}

func CleanseResourceListMetadata(resources *unstructured.UnstructuredList) {
	for i := range resources.Items {
		CleanseResourceMetadata(&resources.Items[i])
	}
}

func DecodeResource(resource []byte) (*unstructured.Unstructured, error) {
	var object map[string]interface{}

	if err := yaml.Unmarshal(resource, &object); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: object}, nil
}

func DecodeResourceList(resources []byte) (*unstructured.UnstructuredList, error) {
	var object map[string]interface{}

	if err := yaml.Unmarshal(resources, &object); err != nil {
		return nil, err
	}

	rawItems, ok := object["items"]
	if !ok {
		return nil, fmt.Errorf("missing items in list resource")
	}

	rawItemList, ok := rawItems.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid items in list resource")
	}

	items := make([]unstructured.Unstructured, 0, len(rawItemList))
	for _, rawItem := range rawItemList {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid item in list resource")
		}

		items = append(items, unstructured.Unstructured{Object: item})
	}

	delete(object, "items")

	return &unstructured.UnstructuredList{Object: object, Items: items}, nil
}
