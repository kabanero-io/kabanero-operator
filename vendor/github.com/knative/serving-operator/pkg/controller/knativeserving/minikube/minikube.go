/*
Copyright 2019 The Knative Authors

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
package minikube

import (
	"context"

	mf "github.com/jcrossley3/manifestival"
	"github.com/knative/serving-operator/pkg/controller/knativeserving/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	extension = common.Extension{
		Transformers: []mf.Transformer{egress},
	}
	log = logf.Log.WithName("minikube")
)

// Configure minikube if we're soaking in it
func Configure(c client.Client, _ *runtime.Scheme) (*common.Extension, error) {
	node := &v1.Node{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: "minikube"}, node); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Unable to query for minikube node")
		}
		// Not running in minikube
		return nil, nil
	}
	return &extension, nil
}

func egress(u *unstructured.Unstructured) error {
	if u.GetKind() == "ConfigMap" && u.GetName() == "config-network" {
		data := map[string]string{
			"istio.sidecar.includeOutboundIPRanges": "10.0.0.1/24",
		}
		common.UpdateConfigMap(u, data, log)
	}
	return nil
}
