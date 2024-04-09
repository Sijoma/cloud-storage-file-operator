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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FileTransferSpec defines the desired state of FileTransfer
type FileTransferSpec struct {
	// BucketName is the source bucket
	BucketName string `json:"bucketName"`

	// Query
	Query Query `json:"query"`

	// CopyDestination
	CopyDestination *CopyDestination `json:"copyDestination,omitempty"`

	// Secret
	BucketSecret *v1.SecretReference `json:"bucketSecret,omitempty"`
}

type Query struct {
	Prefix string `json:"prefix,omitempty"`
}

type CopyDestination struct {
	// If a copy destination is specified, the query prefix will be replaced by the destination prefix
	Prefix string `json:"prefix,omitempty"`
}

// FileTransferStatus defines the observed state of FileTransfer
type FileTransferStatus struct {
	FoundObjects int `json:"foundObjects"`
	// Todo: Implement Conditions
	CopyStatus string `json:"copyStatus"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FileTransfer is the Schema for the filetransfers API
type FileTransfer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FileTransferSpec   `json:"spec,omitempty"`
	Status FileTransferStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FileTransferList contains a list of FileTransfer
type FileTransferList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FileTransfer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FileTransfer{}, &FileTransferList{})
}
