/*
Copyright 2023.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EmberStorageBackendSpec defines the desired state of EmberStorageBackend
type EmberStorageBackendSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of EmberStorageBackend. Edit emberstoragebackend_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// EmberStorageBackendStatus defines the observed state of EmberStorageBackend
type EmberStorageBackendStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EmberStorageBackend is the Schema for the emberstoragebackends API
type EmberStorageBackend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmberStorageBackendSpec   `json:"spec,omitempty"`
	Status EmberStorageBackendStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmberStorageBackendList contains a list of EmberStorageBackend
type EmberStorageBackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmberStorageBackend `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmberStorageBackend{}, &EmberStorageBackendList{})
}
