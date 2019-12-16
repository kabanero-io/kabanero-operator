package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	kabTransforms "github.com/kabanero-io/kabanero-operator/pkg/controller/transforms"
	mf "github.com/kabanero-io/manifestival"
	routev1 "github.com/openshift/api/route/v1"
	consolev1 "github.com/openshift/api/console/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var kllog = rlog.Log.WithName("kabanero-landing")

// Deploys resources and customizes to the Openshift web console.
func deployLandingPage(k *kabanerov1alpha1.Kabanero, c client.Client) error {
	// if enable is false do not deploy the landing page
	if k.Spec.Landing.Enable != nil && *(k.Spec.Landing.Enable) == false {
		err := cleanupLandingPage(k, c)
		return err
	}
	rev, err := resolveSoftwareRevision(k, "landing", k.Spec.Landing.Version)
	if err != nil {
		return err
	}

	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides("", "", "", rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image

	f, err := rev.OpenOrchestration("kabanero-landing.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{mf.InjectOwner(k), mf.InjectNamespace(k.GetNamespace())}
	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.ApplyAll()
	if err != nil {
		return err
	}

	// Retrieve the kabanero landing URL.
	landingURL, err := getLandingURL(k, c)
	if err != nil {
		return err
	}

	// Create a Deployment. The landing application requires knowledge of the landingURL
	// post route creation.
	f, err = rev.OpenOrchestration("kabanero-landing-deployment.yaml")
	if err != nil {
		return err
	}

	s, err = renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	m, err = mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	transforms = []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
		kabTransforms.AddEnvVariable("LANDING_URL", landingURL),
	}

	// See if we should define the OAuth volume and variables
	secretInstance := &corev1.Secret{}
	secretName := "kabanero-github-oauth-secret"
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      secretName,
		Namespace: k.ObjectMeta.Namespace}, secretInstance)

	if err == nil {
		// The secret exists, go ahead and put in the oauth information
		transforms = append(transforms, kabTransforms.MountSecret(secretName, "/etc/oauth"))

		// Try and figure out what the OAuth URL for github is, based on the API url.  If the API
		// URL is not set, assume that it is "api.github.com".
		apiUrlString := k.Spec.Github.ApiUrl
		if len(apiUrlString) == 0 {
			apiUrlString = "https://api.github.com"
		}

		apiUrl, err := url.Parse(apiUrlString)
		if err != nil {
			kllog.Error(err, "Could not parse Github API url %v, assuming api.github.com", apiUrlString)
			apiUrl, _ = url.Parse("https://api.github.com")
		}

		hostname := apiUrl.Hostname()
		if hostname == "api.github.com" {
			transforms = append(transforms, kabTransforms.AddEnvVariable("USER_API", "https://api.github.com/user"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("AUTHORIZATION_ENDPOINT", "https://github.com/login/oauth/authorize"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("TOKEN_ENDPOINT", "https://github.com/login/oauth/access_token"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("WEBSITE", "https://github.com"))
		} else {
			transforms = append(transforms, kabTransforms.AddEnvVariable("USER_API", "https://" + hostname + "/api/v3/user"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("AUTHORIZATION_ENDPOINT", "https://" + hostname + "/login/oauth/authorize"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("TOKEN_ENDPOINT", "https://" + hostname + "/login/oauth/access_token"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("WEBSITE", "https://" + hostname))
		}
	}
	
	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.ApplyAll()
	if err != nil {
		return err
	}

	// Update the web console's ConfigMap with custom data.
	err = customizeWebConsole(k, c, landingURL)

	return err
}

func cleanupLandingPage(k *kabanerov1alpha1.Kabanero, c client.Client) error {
	err := removeWebConsoleCustomization(k, c)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	clientset, err := getClient()
	deploymentClient := clientset.AppsV1().Deployments(k.GetNamespace())
	if err != nil {
		return err
	}

	deletePolicy := metav1.DeletePropagationForeground
	err = deploymentClient.Delete("kabanero-landing", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	rev, err := resolveSoftwareRevision(k, "landing", k.Spec.Landing.Version)
	if err != nil {
		return err
	}

	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides("", "", "", rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image

	f, err := rev.OpenOrchestration("kabanero-landing.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{mf.InjectOwner(k), mf.InjectNamespace(k.GetNamespace())}
	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.DeleteAll()
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

// Retrieves the landing URL from the landing Route.
func getLandingURL(k *kabanerov1alpha1.Kabanero, c client.Client) (string, error) {
	landingURL := ""

	// Get the Route instance.
	landingRoute := &routev1.Route{}
	landingRouteName := types.NamespacedName{Namespace: k.ObjectMeta.Namespace, Name: "kabanero-landing"}
	err := c.Get(context.TODO(), landingRouteName, landingRoute)

	if err != nil {
		return landingURL, err
	}

	// Look for the ingress entry with the status of admitted.
	// There should only be one URL and it should be the one that is auto generated.
	for _, ingress := range landingRoute.Status.Ingress {
		for _, condition := range ingress.Conditions {
			if condition.Type == routev1.RouteAdmitted && condition.Status == corev1.ConditionTrue {
				landingURL = ingress.Host
				break
			}
		}
	}

	// If the URL is invalid, return an error.
	if len(landingURL) == 0 {
		err = errors.New("Invalid kabanero landing URL")
	} else {
		landingURL = "https://" + landingURL
	}

	kllog.Info(fmt.Sprintf("getLandingURL: URL: %v", landingURL))

	return landingURL, err
}

// Returns a copy of a ConsoleLink object
func getConsoleLink(c client.Client, linkName string) (*consolev1.ConsoleLink, error) {
	consoleLink := &consolev1.ConsoleLink{}
	err := c.Get(context.TODO(), types.NamespacedName{
		Namespace: "", Name: linkName}, consoleLink)
	if err != nil {
		return nil, err
	}

	return consoleLink, nil
}

// Adds customizations to the OpenShift web console.
func customizeWebConsole(k *kabanerov1alpha1.Kabanero, c client.Client, landingURL string) error {

	// See if we've added the apps link yet.
	clientOp := c.Update
	consoleLink, err := getConsoleLink(c, "kabanero-app-menu-link")
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		consoleLink = &consolev1.ConsoleLink{}
		consoleLink.Name = "kabanero-app-menu-link"
		consoleLink.Spec.Location = "ApplicationMenu"
		consoleLink.Spec.Text = "Landing Page"
		consoleLink.Spec.ApplicationMenu = &consolev1.ApplicationMenuSpec{}
		consoleLink.Spec.ApplicationMenu.Section = "Kabanero"
		clientOp = c.Create

		kllog.Info("Creating ConsoleLink kabanero-app-menu-link")
	}

	// Stuff that could change (dependent on the landingURL)
	consoleLink.Spec.Href = landingURL
	consoleLink.Spec.ApplicationMenu.ImageURL = landingURL + "/img/favicon/favicon-16x16.png"
	err = clientOp(context.TODO(), consoleLink)
	if err != nil {
		return err
	}

	// See if we've added the help links yet.
	clientOp = c.Update
	consoleLink, err = getConsoleLink(c, "kabanero-help-menu-docs")
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		consoleLink = &consolev1.ConsoleLink{}
		consoleLink.Name = "kabanero-help-menu-docs"
		consoleLink.Spec.Location = "HelpMenu"
		consoleLink.Spec.Text = "Kabanero Docs"
		clientOp = c.Create

		kllog.Info("Creating ConsoleLink kabanero-help-menu-docs")
	}

	// Stuff that could change (dependent on the landing URL)
	consoleLink.Spec.Href = landingURL + "/docs"
	err = clientOp(context.TODO(), consoleLink)
	if err != nil {
		return err
	}

	clientOp = c.Update
	consoleLink, err = getConsoleLink(c, "kabanero-help-menu-guides")
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		consoleLink = &consolev1.ConsoleLink{}
		consoleLink.Name = "kabanero-help-menu-guides"
		consoleLink.Spec.Location = "HelpMenu"
		consoleLink.Spec.Text = "Kabanero Guides"
		clientOp = c.Create
	}

	// Stuff that could change (dependent on the landing URL)
	consoleLink.Spec.Href = landingURL + "/guides"
	err = clientOp(context.TODO(), consoleLink)
	if err != nil {
		return err
	}

	return nil
}

// Removes customizations from the openshift console.
func removeWebConsoleCustomization(k *kabanerov1alpha1.Kabanero, c client.Client) error {
	// Since these are cluster level objects, they cannot set a namespace-level owner and must be
	// removed manually.
	consoleLink, err := getConsoleLink(c, "kabanero-app-menu-link")
	if err == nil {
		err = c.Delete(context.TODO(), consoleLink)
		if err != nil {
			kllog.Error(err, "Unable to delete ConsoleLink")
		}
	}
	
	consoleLink, err = getConsoleLink(c, "kabanero-help-menu-docs")
	if err == nil {
		err = c.Delete(context.TODO(), consoleLink)
		if err != nil {
			kllog.Error(err, "Unable to delete ConsoleLink")
		}
	}

	consoleLink, err = getConsoleLink(c, "kabanero-help-menu-guides")
	if err == nil {
		err = c.Delete(context.TODO(), consoleLink)
		if err != nil {
			kllog.Error(err, "Unable to delete ConsoleLink")
		}
	}

	return nil
}

// Retrieves the current kabanero landing page status.
func getKabaneroLandingPageStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	// If disabled. Nothing to do. No need to display status if disabled.
	if (k.Spec.Landing.Enable != nil) && (*k.Spec.Landing.Enable == false) {
		k.Status.Landing = nil
		return true, nil
	}

	k.Status.Landing = &kabanerov1alpha1.KabaneroLandingPageStatus{}
	k.Status.Landing.ErrorMessage = ""
	k.Status.Landing.Ready = "False"

	// Create a clientset to drive API operations on resources.
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.ErrorMessage = "Failed to build configuration to retrieve status."
		return false, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.ErrorMessage = "Failed to create clientset to retrieve status."
		return false, err
	}

	// By default there should only be one instance of the landing pod (replica count = 1).
	// Aggregate the results if that changes.
	ready := true
	finalErrorMessage := ""
	rev, err := resolveSoftwareRevision(k, "landing", k.Spec.Landing.Version)
	if err != nil {
		return false, err
	}
	k.Status.Landing.Version = rev.Version

	options := metav1.ListOptions{LabelSelector: "app=kabanero-landing"}
	pods, err := clientset.CoreV1().Pods(k.ObjectMeta.Namespace).List(options)
	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.ErrorMessage = "Pod instance(s) with label kabanero-landing under the namespace of " + k.ObjectMeta.Namespace + " could not be retrieved."
		return false, err
	}

	for _, pod := range pods.Items {
		for _, condition := range pod.Status.Conditions {
			if strings.ToLower(string(condition.Type)) == "ready" {
				status := string(condition.Status)
				if strings.ToLower(status) == "false" {
					ready = false
					finalErrorMessage += "Pod " + pod.Name + " not ready: " + condition.Message + ". "
				}
				break
			}
		}
	}

	if ready {
		k.Status.Landing.Ready = "True"
	} else {
		k.Status.Landing.Ready = "False"
	}

	k.Status.Landing.ErrorMessage = finalErrorMessage

	return ready, err
}

// Returns a Clientset object.
func getClient() (*kubernetes.Clientset, error) {
	// Create a clientset to drive API operations on resources.
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, err
}
