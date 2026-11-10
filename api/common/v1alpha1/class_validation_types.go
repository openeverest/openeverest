// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

// ClassValidation is a CEL rule a class applies to the objects that reference
// it, for constraints its parameters schema cannot express: rules on typed
// fields of the object, or across them and its parameters. It is evaluated
// with the CEL library and cost limits of CRD validation rules.
type ClassValidation struct {
	// Expression is a CEL expression that is true for a valid object. self is
	// the referencing object's spec, with its parameters typed by the class's
	// parameters schema. oldSelf is not available.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=4096
	Expression string `json:"expression"`
	// Message explains a violation to the user.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	Message string `json:"message"`
	// Reason is the reason of the condition a violation sets on the
	// referencing object. Each class kind lists the reasons it accepts.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^[A-Z][A-Za-z0-9]*$`
	Reason string `json:"reason"`
}
