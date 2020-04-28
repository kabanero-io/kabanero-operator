package kabaneroplatform

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	kabTransforms "github.com/kabanero-io/kabanero-operator/pkg/controller/transforms"
	"github.com/kabanero-io/kabanero-operator/pkg/versioning"
	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/log"
)

var kllog = rlog.Log.WithName("kabanero-landing")

// Deploys resources and customizes to the Openshift web console.
func deployLandingPage(_ context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, logger logr.Logger) error {
	// If enable is false do not deploy the landing page.
	if k.Spec.Landing.Enable != nil && *(k.Spec.Landing.Enable) == false {
		err := cleanupLandingPage(k, c)
		return err
	}

	// Resolve the landing software infomation (versions.yaml) with applied overrides (CR instance spec).
	rev, err := resolveSoftwareRevision(k, "landing", k.Spec.Landing.Version)
	if err != nil {
		return err
	}

	// If the orchestration version being used has a route that uses passthrough TLS termination, the existing
	// route might require cleanup. This is done if the previous instance of the landing service configured
	// the route to use reencrypt TLS termination.
	usingPassthroughTLS := strings.HasSuffix(rev.OrchestrationPath, "0.1")
	if usingPassthroughTLS {
		err = removeTLSCertsFromLandingRoute(k, c)
		if err != nil {
			return err
		}
	}

	// Apply CLI service resources excluding the deployment, which is applied separatelly.
	f, err := rev.OpenOrchestration("kabanero-landing.yaml")
	if err != nil {
		return err
	}

	// The context which will be used to render any templates
	templateContext := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.Landing.Repository, k.Spec.Landing.Tag, k.Spec.Landing.Image, rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(logger.WithName("manifestival")))
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{mf.InjectOwner(k), mf.InjectNamespace(k.GetNamespace())}
	m, err := mOrig.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.Apply()
	if err != nil {
		return err
	}

	// Only 0.2+ orchestrations support landing page with reencypt tls termination.
	if !usingPassthroughTLS {
		err = addTLSConfigToLandingRoute(k, c, rev)
		if err != nil {
			return err
		}
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

	mOrig, err = mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(logger.WithName("manifestival")))
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
		} else if len(apiUrl.Scheme) == 0 {
			apiUrl.Scheme = "https"
		}
		hostname := apiUrl.Hostname()
		if hostname == "api.github.com" {
			transforms = append(transforms, kabTransforms.AddEnvVariable("USER_API", "https://api.github.com/user"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("AUTHORIZATION_ENDPOINT", "https://github.com/login/oauth/authorize"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("TOKEN_ENDPOINT", "https://github.com/login/oauth/access_token"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("WEBSITE", "https://github.com"))
		} else {
			transforms = append(transforms, kabTransforms.AddEnvVariable("USER_API", "https://"+hostname+"/api/v3/user"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("AUTHORIZATION_ENDPOINT", "https://"+hostname+"/login/oauth/authorize"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("TOKEN_ENDPOINT", "https://"+hostname+"/login/oauth/access_token"))
			transforms = append(transforms, kabTransforms.AddEnvVariable("WEBSITE", "https://"+hostname))
		}
	}

	m, err = mOrig.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.Apply()
	if err != nil {
		return err
	}

	// Update the web console's ConfigMap with custom data.
	err = customizeWebConsole(k, c, landingURL)

	return err
}

func cleanupLandingPage(k *kabanerov1alpha2.Kabanero, c client.Client) error {
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

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(rlog.Log.WithName("manifestival")))
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{mf.InjectOwner(k), mf.InjectNamespace(k.GetNamespace())}
	m, err := mOrig.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.Delete()
	if err != nil {
		return err
	}

	return nil
}

// Updates the route with openshift generated TLS key and certificate.
// The certificate and TLS key were produced by OpenShift and were added to a secret during service creation.
// The service annotation that triggered the creation of the secret/cert/key is:
// service.beta.openshift.io/serving-cert-secret-name: kabanero-landing-service-cert-secret
func addTLSConfigToLandingRoute(k *kabanerov1alpha2.Kabanero, c client.Client, rev versioning.SoftwareRevision) error {
	// Retrieve the sevice created secret and get the TLS cert/key.
	secretName := "kabanero-landing-service-cert-secret"
	secretInstance := &corev1.Secret{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      secretName,
		Namespace: k.GetNamespace()}, secretInstance)

	if err != nil {
		return fmt.Errorf("Unable to retrieve a secret object. Secret name: %v. Namespace: %v. Error: %v", secretName, k.GetNamespace(), err)
	}

	tlskey, ok := secretInstance.Data["tls.key"]
	if !ok {
		return fmt.Errorf("The data.tls.key entry under secret %v was not found", secretName)
	}

	encodedKey := base64.StdEncoding.EncodeToString(tlskey)
	decodedStringkey, err := base64.StdEncoding.DecodeString(encodedKey)

	tlscrt, ok := secretInstance.Data["tls.crt"]
	if !ok {
		return fmt.Errorf("The data.tls.crt entry under secret %v was not found", secretName)
	}
	encodedCrt := base64.StdEncoding.EncodeToString(tlscrt)
	decodedCrtString, err := base64.StdEncoding.DecodeString(encodedCrt)

	// Get the landing route.
	routeName := "kabanero-landing"
	ri := &routev1.Route{}
	err = c.Get(context.Background(), types.NamespacedName{
		Name:      routeName,
		Namespace: k.ObjectMeta.Namespace}, ri)

	if err != nil {
		return fmt.Errorf("Unable to retrieve landing route. Route Name: %v. Namespace: %v. Error: %v", routeName, k.GetNamespace(), err)
	}

	// Add TLS cert/key to route.
	ri.Spec.TLS.Key = string(decodedStringkey)
	ri.Spec.TLS.Certificate = string(decodedCrtString)

	err = c.Update(context.Background(), ri)
	if err != nil {
		return fmt.Errorf("Unable to update landing route with secret data. Route Name: %v. Namespace: %v. Error: %v", routeName, k.GetNamespace(), err)
	}

	return nil
}

// Removes the openshift generated TLS key and certificate from the currently deployed route if they were
// previously specified.
func removeTLSCertsFromLandingRoute(k *kabanerov1alpha2.Kabanero, c client.Client) error {
	// Get the landing route.
	routeName := "kabanero-landing"
	ri := &routev1.Route{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      routeName,
		Namespace: k.ObjectMeta.Namespace}, ri)

	if err != nil {
		return fmt.Errorf("Unable to retrieve landing route. Route Name: %v. Namespace: %v. Error: %v", routeName, k.GetNamespace(), err)
	}

	// Remove TLS cert/key from route.
	if len(ri.Spec.TLS.Key) != 0 || len(ri.Spec.TLS.Certificate) != 0 {
		ri.Spec.TLS.Key = ""
		ri.Spec.TLS.Certificate = ""

		err = c.Update(context.Background(), ri)
		if err != nil {
			return fmt.Errorf("Unable to update landing route. Route Name: %v. Namespace: %v. Error: %v", routeName, k.GetNamespace(), err)
		}
	}

	return nil
}

// Retrieves the landing URL from the landing Route.
func getLandingURL(k *kabanerov1alpha2.Kabanero, c client.Client) (string, error) {
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
func customizeWebConsole(k *kabanerov1alpha2.Kabanero, c client.Client, landingURL string) error {

	// See if we've added the apps link yet.
	clientOp := utils.Update
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
		clientOp = utils.Create

		kllog.Info("Creating ConsoleLink kabanero-app-menu-link")
	}

	// Stuff that could change (dependent on the landingURL)
	consoleLink.Spec.Href = landingURL
	consoleLink.Spec.ApplicationMenu.ImageURL = landingURL + "/img/favicon/favicon-16x16.png"
	err = clientOp(c, context.TODO(), consoleLink)
	if err != nil {
		return err
	}

	// See if we've added the help links yet.
	clientOp = utils.Update
	consoleLink, err = getConsoleLink(c, "kabanero-help-menu-docs")
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		consoleLink = &consolev1.ConsoleLink{}
		consoleLink.Name = "kabanero-help-menu-docs"
		consoleLink.Spec.Location = "HelpMenu"
		consoleLink.Spec.Text = "Kabanero Docs"
		clientOp = utils.Create

		kllog.Info("Creating ConsoleLink kabanero-help-menu-docs")
	}

	// Stuff that could change (dependent on the landing URL)
	consoleLink.Spec.Href = landingURL + "/docs"
	err = clientOp(c, context.TODO(), consoleLink)
	if err != nil {
		return err
	}

	clientOp = utils.Update
	consoleLink, err = getConsoleLink(c, "kabanero-help-menu-guides")
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		consoleLink = &consolev1.ConsoleLink{}
		consoleLink.Name = "kabanero-help-menu-guides"
		consoleLink.Spec.Location = "HelpMenu"
		consoleLink.Spec.Text = "Kabanero Guides"
		clientOp = utils.Create
	}

	// Stuff that could change (dependent on the landing URL)
	consoleLink.Spec.Href = landingURL + "/guides"
	err = clientOp(c, context.TODO(), consoleLink)
	if err != nil {
		return err
	}

	return nil
}

// Removes customizations from the openshift console.
func removeWebConsoleCustomization(k *kabanerov1alpha2.Kabanero, c client.Client) error {
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
func getKabaneroLandingPageStatus(k *kabanerov1alpha2.Kabanero, c client.Client) (bool, error) {
	// If disabled. Nothing to do. No need to display status if disabled.
	if (k.Spec.Landing.Enable != nil) && (*k.Spec.Landing.Enable == false) {
		k.Status.Landing = nil
		return true, nil
	}

	k.Status.Landing = &kabanerov1alpha2.KabaneroLandingPageStatus{}
	k.Status.Landing.Message = ""
	k.Status.Landing.Ready = "False"

	// Create a clientset to drive API operations on resources.
	config, err := rest.InClusterConfig()
	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.Message = "Failed to build configuration to retrieve status."
		return false, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.Message = "Failed to create clientset to retrieve status."
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
		k.Status.Landing.Message = "Pod instance(s) with label kabanero-landing under the namespace of " + k.ObjectMeta.Namespace + " could not be retrieved."
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

	k.Status.Landing.Message = finalErrorMessage

	return ready, err
}

// Returns a Clientset object.
func getClient() (*kubernetes.Clientset, error) {
	// Create a clientset to drive API operations on resources.
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, err
}
