/*
Copyright 2025.

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

type RavenDBClusterSpec struct {
	Image            string            `json:"image"`
	ImagePullPolicy  string            `json:"imagePullPolicy"`
	Mode             string            `json:"mode"`
	Email            string            `json:"email,omitempty"`
	License          string            `json:"license"`
	Domain           string            `json:"domain"`
	ServerUrl        string            `json:"serverUrl"`
	ServerUrlTcp     string            `json:"serverUrlTcp"`
	StorageSize      string            `json:"storageSize"`
	Environment      map[string]string `json:"environment,omitempty"` // env vars
	Nodes            []RavenDBNode     `json:"nodes,omitempty"`
	IngressClassName string            `json:"ingressClassName,omitempty"`
}

type RavenDBNode struct {
	Name               string `json:"name"`
	PublicServerUrl    string `json:"publicServerUrl"`
	PublicServerUrlTcp string `json:"publicServerUrlTcp"`
	CertsSecretRef     string `json:"certsSecretRef,omitempty"`
}

type RavenDBClusterStatus struct {
	Phase   string              `json:"phase,omitempty"`
	Message string              `json:"message,omitempty"`
	Nodes   []RavenDBNodeStatus `json:"nodes,omitempty"`
}

type RavenDBNodeStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RavenDBCluster is the Schema for the ravendbclusters API
type RavenDBCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RavenDBClusterSpec   `json:"spec,omitempty"`
	Status RavenDBClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RavenDBClusterList contains a list of RavenDBCluster
type RavenDBClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RavenDBCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RavenDBCluster{}, &RavenDBClusterList{})
}
