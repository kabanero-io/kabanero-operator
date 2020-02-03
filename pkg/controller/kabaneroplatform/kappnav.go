package kabaneroplatform

import (
	"context"
	goerrors "errors"
	"strings"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	routev1 "github.com/openshift/api/route/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/log"
)

var kanlog = rlog.Log.WithName("kabanero-kappnav")

const currentVersion = "0.1.0"

// Retrieves the Kubernetes Application Navigator deployment status.
func getKappnavStatus(k *kabanerov1alpha2.Kabanero, c client.Client) (bool, error) {
	// KNavApp is optional.  We're basically just reporting if we found it.
	// If we found the default instance, we'll wait until its components
	// are ready.
	status := kabanerov1alpha2.KappnavStatus{}
	k.Status.Kappnav = &status
	k.Status.Kappnav.Ready = "False"

	// We're looking in the default location for KAppNav.
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "KAppNav",
		Group:   "charts.helm.k8s.io",
		Version: "v1alpha1",
	})
	err := c.Get(context.Background(), client.ObjectKey{
		Namespace: "kappnav",
		Name:      "instance",
	}, u)

	// If we could not retrieve the default instance, lets assume that
	// KAppNav is not configured.  We do not want to put any KAppNav status
	// in the Kabanero status for fear of misleading the user into thinking
	// that there is a configuration problem.
	if err != nil {
		k.Status.Kappnav = nil
		return true, nil // Don't fail Kabanero status for this.
	}

	// The default instance is there, see if the UI pod is available.

	listOptions := []client.ListOption{
		client.InNamespace("kappnav"),
		client.MatchingLabels(map[string]string{"app.kubernetes.io/component": "kappnav-ui"}),
	}
	podList := &corev1.PodList{}
	err = c.List(context.Background(), podList, listOptions...)
	if err != nil {
		if errors.IsNotFound(err) {
			k.Status.Kappnav.Message = "The KAppNav UI pod could not be located."
			return false, err // We should wait for it to start
		}

		k.Status.Kappnav.Message = "Could not detect the status of KAppNav: " + err.Error()
		return false, err // We should wait for it to start
	}

	// Look thru all of the UI pods, and see if any are ready.
	finalErrorMessage := ""
	ready := false
	for _, pod := range podList.Items {
		for _, condition := range pod.Status.Conditions {
			if strings.ToLower(string(condition.Type)) == "ready" {
				readyStatus := string(condition.Status)
				if strings.ToLower(readyStatus) == "true" {
					ready = true
				} else {
					finalErrorMessage += "Pod " + pod.Name + " not ready: " + condition.Message + ". "
				}
				break
			}
		}
		if ready {
			break
		}
	}

	// If we could not find a pod that was ready, report the condition.
	if ready == false {
		k.Status.Kappnav.Message = finalErrorMessage
		return false, nil // We should retry to see if the pod is available
	}

	// Check that the UI route is accepted
	var uiLocations []string = nil
	uiLocations, err = getRouteLocations("kappnav-ui-service", "kappnav", c)
	if err != nil {
		k.Status.Kappnav.Message = err.Error()
		return false, err // We should wait until the route is ready.
	}

	k.Status.Kappnav.UiLocations = uiLocations

	// Now do the same for the API route
	var apiLocations []string = nil
	apiLocations, err = getRouteLocations("kappnav-api-service", "kappnav", c)
	if err != nil {
		k.Status.Kappnav.Message = err.Error()
		return false, err // We should wait until the route is ready
	}

	k.Status.Kappnav.ApiLocations = apiLocations

	// All is well.
	k.Status.Kappnav.Ready = "True"
	return true, nil
}

// Gets route locations (hostname + path)
func getRouteLocations(routeName string, namespace string, c client.Client) ([]string, error) {
	route := &routev1.Route{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      routeName,
		Namespace: namespace}, route)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, goerrors.New("The Route for " + routeName + " could not be found")
		}

		return nil, goerrors.New("Could not get the route for " + routeName + ": " + err.Error())
	}

	// Looking for an ingress that has an admitted status and a hostname
	var locations []string = nil
	for _, ingress := range route.Status.Ingress {
		var routeAdmitted bool = false
		for _, condition := range ingress.Conditions {
			if condition.Type == routev1.RouteAdmitted && condition.Status == corev1.ConditionTrue {
				routeAdmitted = true
			}
		}
		if routeAdmitted == true && len(ingress.Host) > 0 {
			locations = append(locations, ingress.Host+route.Spec.Path)
		}
	}

	// Make sure we got something back.
	if len(locations) == 0 {
		return nil, goerrors.New("There were no accepted ingress objects in the " + routeName + " Route")
	}

	return locations, nil
}
