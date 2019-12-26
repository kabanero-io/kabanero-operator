module github.com/kabanero-io/kabanero-operator

go 1.12

require (
	contrib.go.opencensus.io/exporter/ocagent v0.4.12 // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.8 // indirect
	github.com/Azure/go-autorest v12.0.0+incompatible // indirect
	github.com/blang/semver v0.0.0-20190414102917-ba2c2ddd8906
	github.com/docker/distribution v2.7.0+incompatible // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.3
	github.com/gobuffalo/envy v1.7.0 // indirect
	github.com/google/go-containerregistry v0.0.0-20191218175032-34fb8ff33bed // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/kabanero-io/manifestival v0.0.0-20190904195909-5a78e825928e
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
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/xerrors v0.0.0-20191011141410-1b5146add898 // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apiextensions-apiserver v0.0.0-20190820104113-47893d27d7f7
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	knative.dev/pkg v0.0.0-20190817231834-12ee58e32cc8 // indirect
	sigs.k8s.io/controller-runtime v0.1.12
	sigs.k8s.io/testing_frameworks v0.1.2 // indirect
)

// Pinned to kubernetes-1.13.4
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
	// Pinned to v2.9.2 (kubernetes-1.13.1) so https://proxy.golang.org can
	// resolve it correctly.
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20190424153033-d3245f150225
	k8s.io/kube-state-metrics => k8s.io/kube-state-metrics v1.6.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.12
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
)

replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.10.0
