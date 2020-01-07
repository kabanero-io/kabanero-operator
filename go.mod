module github.com/kabanero-io/kabanero-operator

go 1.12

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.8 // indirect
	github.com/Azure/go-autorest v12.0.0+incompatible // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/docker/distribution v2.7.0+incompatible // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.3
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-containerregistry v0.0.0-20191218175032-34fb8ff33bed // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/kabanero-io/manifestival v0.0.0-20191230200607-194733cd27d9
	github.com/knative/pkg v0.0.0-20190817231834-12ee58e32cc8 // indirect
	github.com/knative/serving-operator v0.0.0-20190702004031-e30377b852ff
	github.com/openshift-knative/knative-eventing-operator v0.6.0
	github.com/openshift/api v0.0.0-20190927182313-d4a64ec2cbd8
	github.com/operator-framework/operator-lifecycle-manager v3.11.0+incompatible
	github.com/operator-framework/operator-sdk v0.9.1-0.20190715204459-936584d47ff9
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd // indirect
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/operator v0.0.0-20191017104520-be5a46fc149a
	github.com/tektoncd/pipeline v0.7.0
	go.uber.org/atomic v1.4.0 // indirect
	golang.org/x/xerrors v0.0.0-20191011141410-1b5146add898 // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apiextensions-apiserver v0.0.0-20190918161926-8f644eb6e783
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	knative.dev/pkg v0.0.0-20190817231834-12ee58e32cc8 // indirect
	sigs.k8s.io/controller-runtime v0.2.0
	sigs.k8s.io/testing_frameworks v0.1.2 // indirect
)

// Pinned to kubernetes-1.14.1
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190409023720-1bc0c81fa51d
)

replace (
	// Indirect operator-sdk dependencies use git.apache.org, which is frequently
	// down. The github mirror should be used instead.
	// Locking to a specific version (from 'go mod graph'):
	git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.31.1
	// Pinned to v2.10.0 (kubernetes-1.14.1) so https://proxy.golang.org can
	// resolve it correctly.
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20190525122359-d20e84d0fb64
)

replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.11.0
