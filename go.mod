module github.com/kabanero-io/kabanero-operator

go 1.13

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/go-semver v0.3.0
	github.com/docker/cli v0.0.0-20200210162036-a4bedce16568
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/elsony/devfile2-registry/tools v0.0.0-20200603181527-db339ef8dd30
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.6
	github.com/google/go-cmp v0.4.0
	github.com/google/go-containerregistry v0.0.0-20200331213917-3d03ed9b1ca2
	github.com/google/go-github/v29 v29.0.3
	github.com/manifestival/controller-runtime-client v0.1.1-0.20200218204725-1af9550ddf8f
	github.com/manifestival/manifestival v0.5.1-0.20200526175228-b0136214e13f
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/operator-framework/operator-lifecycle-manager v3.11.0+incompatible
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/operator v0.0.0-20191017104520-be5a46fc149a
	github.com/tektoncd/pipeline v0.10.1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
	knative.dev/operator v0.15.0
	knative.dev/serving v0.15.1
	sigs.k8s.io/controller-runtime v0.5.3
)

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20190924102528-32369d4db2ad // Required until https://github.com/operator-framework/operator-lifecycle-manager/pull/1241 is resolved

replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.17.1

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM

replace k8s.io/client-go => k8s.io/client-go v0.17.6 // Required by prometheus-operator
