package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Scaler is scaling definition for deployments
type Scaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ScalerSpec `json:"spec"`
}

// ScalerSpec is the specification for Scalers
type ScalerSpec struct {
	Label       string      `json:"label"`
	MinReplicas int32       `json:"minReplicas"`
	MaxReplicas int32       `json:"maxReplicas"`
	Target      ScaleTarget `json:"target"`
	CPUUsage    int32       `json:cpuUsage`
}

// ScaleTarget is the scaling target for the Scaler
type ScaleTarget struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ScalerList is list of Scalers
type ScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Scaler `json:"items"`
}
