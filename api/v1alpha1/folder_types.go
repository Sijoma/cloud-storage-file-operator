/*
Copyright 2024.

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

// FolderSpec defines the desired state of Folder
type FolderSpec struct {
	// The parent bucket of the managed folder.
	BucketName string `json:"bucketName"`

	// The name of the managed folder, expressed as a path. For example, example-dir or example-dir/example-dir1.
	Name string `json:"name"`
}

// FolderStatus defines the observed state of Folder
type FolderStatus struct {
	ServiceAccountName string `json:"serviceAccountName"`
	Email              string `json:"email"`
	Folder             string `json:"folder"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Folder is the Schema for the folders API
type Folder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FolderSpec   `json:"spec,omitempty"`
	Status FolderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FolderList contains a list of Folder
type FolderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Folder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Folder{}, &FolderList{})
}
