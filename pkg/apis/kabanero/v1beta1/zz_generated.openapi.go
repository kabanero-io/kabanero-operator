// +build !ignore_autogenerated

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1beta1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.Collection":       schema_pkg_apis_kabanero_v1beta1_Collection(ref),
		"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.CollectionSpec":   schema_pkg_apis_kabanero_v1beta1_CollectionSpec(ref),
		"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.CollectionStatus": schema_pkg_apis_kabanero_v1beta1_CollectionStatus(ref),
		"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.Kabanero":         schema_pkg_apis_kabanero_v1beta1_Kabanero(ref),
		"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroSpec":     schema_pkg_apis_kabanero_v1beta1_KabaneroSpec(ref),
		"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroStatus":   schema_pkg_apis_kabanero_v1beta1_KabaneroStatus(ref),
	}
}

func schema_pkg_apis_kabanero_v1beta1_Collection(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Collection is the Schema for the collections API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.CollectionSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.CollectionStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.CollectionSpec", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.CollectionStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_kabanero_v1beta1_CollectionSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "CollectionSpec defines the desired state of Collection",
				Properties: map[string]spec.Schema{
					"repositoryUrl": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"name": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"version": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_kabanero_v1beta1_CollectionStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "CollectionStatus defines the observed state of Collection",
				Properties: map[string]spec.Schema{
					"activeVersion": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"activeDigest": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"activeAssets": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.RepositoryAssetStatus"),
									},
								},
							},
						},
					},
					"availableVersion": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"availableLocation": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"statusMessage": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.RepositoryAssetStatus"},
	}
}

func schema_pkg_apis_kabanero_v1beta1_Kabanero(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Kabanero is the Schema for the kabaneros API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroSpec", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_kabanero_v1beta1_KabaneroSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KabaneroSpec defines the desired state of Kabanero",
				Properties: map[string]spec.Schema{
					"version": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"githubOrganization": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"collections": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.InstanceCollectionConfig"),
						},
					},
					"tekton": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.TektonCustomizationSpec"),
						},
					},
					"appsodyOperator": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.AppsodyCustomizationSpec"),
						},
					},
					"cliServices": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroCliServicesCustomizationSpec"),
						},
					},
					"landing": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroLandingCustomizationSpec"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.AppsodyCustomizationSpec", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.InstanceCollectionConfig", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroCliServicesCustomizationSpec", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroLandingCustomizationSpec", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.TektonCustomizationSpec"},
	}
}

func schema_pkg_apis_kabanero_v1beta1_KabaneroStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "KabaneroStatus defines the observed state of the Kabanero instance",
				Properties: map[string]spec.Schema{
					"kabaneroInstance": {
						SchemaProps: spec.SchemaProps{
							Description: "Kabanero operator instance readiness status. The status is directly correlated to the availability of resources dependencies.",
							Ref:         ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroInstanceStatus"),
						},
					},
					"knativeEventing": {
						SchemaProps: spec.SchemaProps{
							Description: "Knative eventing instance readiness status.",
							Ref:         ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KnativeEventingStatus"),
						},
					},
					"knativeServing": {
						SchemaProps: spec.SchemaProps{
							Description: "Knative serving instance readiness status.",
							Ref:         ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KnativeServingStatus"),
						},
					},
					"tekton": {
						SchemaProps: spec.SchemaProps{
							Description: "Tekton instance readiness status.",
							Ref:         ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.TektonStatus"),
						},
					},
					"cli": {
						SchemaProps: spec.SchemaProps{
							Description: "CLI readiness status.",
							Ref:         ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.CliStatus"),
						},
					},
					"landing": {
						SchemaProps: spec.SchemaProps{
							Description: "Kabanero Landing page readiness status.",
							Ref:         ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroLandingPageStatus"),
						},
					},
					"appsody": {
						SchemaProps: spec.SchemaProps{
							Description: "Appsody instance readiness status.",
							Ref:         ref("github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.AppsodyStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.AppsodyStatus", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.CliStatus", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroInstanceStatus", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KabaneroLandingPageStatus", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KnativeEventingStatus", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.KnativeServingStatus", "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1beta1.TektonStatus"},
	}
}
