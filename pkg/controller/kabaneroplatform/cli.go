package kabaneroplatform

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	mf "github.com/jcrossley3/manifestival"
	routev1 "github.com/openshift/api/route/v1"
)


func reconcileKabaneroCli(ctx context.Context, k *kabanerov1alpha1.Kabanero, cl client.Client, reqLogger logr.Logger) error {
	// Create a clientset to drive API operations on resources.
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Deploy some of the Kabanero CLI components - service acct, role, etc
	filename := "config/reconciler/kabanero-cli.yaml"
	m, err := mf.NewManifest(filename, true, cl)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.ApplyAll()
	if err != nil {
		return err
	}

	// Create the deployment manually as we have to fill in some env vars.
	image := "kabanero/kabanero-command-line-services:0.1.0"
	env := []corev1.EnvVar{{Name:  "KABANERO_CLI_NAMESPACE",	Value: k.GetNamespace()	}}
	if len(k.Spec.GithubOrganization) > 0 {
		env = append(env, corev1.EnvVar{Name:  "KABANERO_CLI_GROUP", Value: k.Spec.GithubOrganization})
	}
	// Need to construct this the long way due to anonymous fields
	configMapOptional := false
	configMapEnvSource := corev1.ConfigMapEnvSource{Optional: &configMapOptional}
	configMapEnvSource.Name = "kabanero-cli-role-config"
	envFrom := []corev1.EnvFromSource{{ConfigMapRef: &configMapEnvSource}}
	err = createDeployment(k, clientset, cl, "kabanero-cli", image, env, envFrom, reqLogger)
	if err != nil {
		return err
	}

	return nil
}



// Tries to see if the CLI route has been assigned a hostname.
func getCliRouteStatus(k *kabanerov1alpha1.Kabanero, reqLogger logr.Logger) (bool, error) {

	// Get the knative eventing installation instance.
	config, err := clientcmd.BuildConfigFromFlags("", "")
	myScheme := runtime.NewScheme()
	cl, _ := client.New(config, client.Options{Scheme: myScheme})
	routev1.AddToScheme(myScheme)

	// Check that the route is accepted
	cliRoute := &routev1.Route{}
	cliRouteName := types.NamespacedName{Namespace: k.ObjectMeta.Namespace, Name: "kabanero-cli"}
	err = cl.Get(context.TODO(), cliRouteName, cliRoute)
	if err == nil {
		k.Status.Cli.Hostnames = nil
		// Looking for an ingress that has an admitted status and a hostname
		for _, ingress := range cliRoute.Status.Ingress {
			var routeAdmitted bool = false
			for _, condition := range ingress.Conditions {
				if condition.Type == routev1.RouteAdmitted && condition.Status == corev1.ConditionTrue {
					routeAdmitted = true
				}
			}
			if routeAdmitted == true && len(ingress.Host) > 0 {
				k.Status.Cli.Hostnames = append(k.Status.Cli.Hostnames, ingress.Host)
			}
		}
		// If we found a hostname from an admitted route, we're done.
		if len(k.Status.Cli.Hostnames) > 0 {
			k.Status.Cli.Ready = "True"
			k.Status.Cli.ErrorMessage = ""
		} else {
			k.Status.Cli.Ready = "False"
			k.Status.Cli.ErrorMessage = "There were no accepted ingress objects in the Route"
			return false, err
		}
	} else {
		var message string
		if errors.IsNotFound(err) {
			message = "The Route object for the CLI was not found"
		} else {
			message = "An error occurred retrieving the Route object for the CLI"
		}
		reqLogger.Error(err, message)
		k.Status.Cli.Ready = "False"
		k.Status.Cli.ErrorMessage = message + ": " + err.Error()
		k.Status.Cli.Hostnames = nil
		return false, err
	}

	return true, nil
}
