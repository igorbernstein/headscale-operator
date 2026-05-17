package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterHeadscaleSpec defines the desired state of ClusterHeadscale
type ClusterHeadscaleSpec struct {
	HeadscaleSpec `json:",inline"`

	// TargetNamespace is the namespace where Headscale server resources
	// (StatefulSet, Service, ConfigMap, etc.) will be created.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	TargetNamespace string `json:"targetNamespace"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=chs

// ClusterHeadscale is the Schema for the clusterheadscales API
type ClusterHeadscale struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec ClusterHeadscaleSpec `json:"spec"`
	// +optional
	Status HeadscaleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ClusterHeadscaleList contains a list of ClusterHeadscale
type ClusterHeadscaleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ClusterHeadscale `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterHeadscale{}, &ClusterHeadscaleList{})
}
