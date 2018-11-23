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
	Spec              ScalerSpec   `json:"spec"`
	Status            ScalerStatus `json:"status"`
}

// ScalerSpec is the specification for Scalers
// +k8s:deepcopy-gen=true
type ScalerSpec struct {
	Label         string      `json:"label"`
	MinReplicas   int32       `json:"minReplicas"`
	MaxReplicas   int32       `json:"maxReplicas"`
	Target        ScaleTarget `json:"target"`
	ScaleDown     int32       `json:"scaleDown"`
	ScaleUp       int32       `json:"scaleUp"`
	Evaluations   int32       `json:"evaluations"`
	ScaleUpSize   int32       `json:"scaleUpSize"`
	ScaleDownSize int32       `json:"scaleDownSize"`
}

// ScalerStatus is the status of the Scaler
// +k8s:deepcopy-gen=true
type ScalerStatus struct {
	Condition string `json:"condition"`
}

// ScaleTarget is the scaling target for the Scaler
// +k8s:deepcopy-gen=true
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
	Items           []Scaler `json:"items"`
}
