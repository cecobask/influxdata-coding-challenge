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

// EmailRequestSpec defines the desired state of EmailRequest
type EmailRequestSpec struct {
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	RecipientEmail string          `json:"recipientEmail,omitempty"`
	RecipientName  string          `json:"recipientName,omitempty"`
	BaseDelay      metav1.Duration `json:"baseDelay,omitempty"`
	MaxDelay       metav1.Duration `json:"maxDelay,omitempty"`
}

// EmailRequestStatus defines the observed state of EmailRequest
type EmailRequestStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
	Failures   int                `json:"failures"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EmailRequest is the Schema for the emailrequests API
type EmailRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmailRequestSpec   `json:"spec,omitempty"`
	Status EmailRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmailRequestList contains a list of EmailRequest
type EmailRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmailRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EmailRequest{}, &EmailRequestList{})
}
