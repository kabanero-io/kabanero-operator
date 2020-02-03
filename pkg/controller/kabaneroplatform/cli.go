package kabaneroplatform

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	kabTransforms "github.com/kabanero-io/kabanero-operator/pkg/controller/transforms"
	mf "github.com/kabanero-io/manifestival"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileKabaneroCli(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) error {
	// Create the AES encryption key secret, if we don't already have one
	err := createEncryptionKeySecret(k, cl, reqLogger)
	if err != nil {
		return err
	}

	// Deploy some of the Kabanero CLI components - service acct, role, etc
	rev, err := resolveSoftwareRevision(k, "cli-services", k.Spec.CliServices.Version)
	if err != nil {
		return err
	}

	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides(k.Spec.CliServices.Repository, k.Spec.CliServices.Tag, k.Spec.CliServices.Image, rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image

	f, err := rev.OpenOrchestration("kabanero-cli.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), cl)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	// The CLI wants to know the Github organization name, if it was provided
	if len(k.Spec.Github.Organization) > 0 {
		transforms = append(transforms, kabTransforms.AddEnvVariable("KABANERO_CLI_GROUP", k.Spec.Github.Organization))
	}

	// The CLI wants to know which teams to bind to the admin role
	if (len(k.Spec.Github.Teams) > 0) && (len(k.Spec.Github.Organization) > 0) {
		// Build a list of fully qualified team names
		teamList := ""
		for _, team := range k.Spec.Github.Teams {
			if len(teamList) > 0 {
				teamList = teamList + ","
			}
			teamList = teamList + team + "@" + k.Spec.Github.Organization
		}
		transforms = append(transforms, kabTransforms.AddEnvVariable("teamsInGroup_admin", teamList))
	}

	// Export the github API URL, if it's set.  This is used by the security portion of the microservice.
	if len(k.Spec.Github.ApiUrl) > 0 {
		apiUrlString := k.Spec.Github.ApiUrl
		apiUrl, err := url.Parse(apiUrlString)

		if err != nil {
			reqLogger.Error(err, "Could not parse Github API url %v, assuming api.github.com", apiUrlString)
			apiUrl, _ = url.Parse("https://api.github.com")
		} else if len(apiUrl.Scheme) == 0 {
			apiUrl.Scheme = "https"
		}
		transforms = append(transforms, kabTransforms.AddEnvVariable("github.api.url", apiUrl.String()))
	}

	// Set JwtExpiration for login duration/timeout
	// Specify a positive integer followed by a unit of time, which can be hours (h), minutes (m), or seconds (s).
	if len(k.Spec.CliServices.SessionExpirationSeconds) > 0 {
		// If the format is incorrect, set the default
		matched, err := regexp.MatchString(`^\d+[smh]{1}$`, k.Spec.CliServices.SessionExpirationSeconds)
		if err != nil {
			return err
		}
		if !matched {
			reqLogger.Info(fmt.Sprintf("Kabanero Spec.CliServices.SessionExpirationSeconds must specify a positive integer followed by a unit of time, which can be hours (h), minutes (m), or seconds (s). Defaulting to 1440m."))
			transforms = append(transforms, kabTransforms.AddEnvVariable("JwtExpiration", "1440m"))
		} else {
			transforms = append(transforms, kabTransforms.AddEnvVariable("JwtExpiration", k.Spec.CliServices.SessionExpirationSeconds))
		}
	} else {
		transforms = append(transforms, kabTransforms.AddEnvVariable("JwtExpiration", "1440m"))
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.ApplyAll()
	if err != nil {
		return err
	}

	// If there is a role binding config map, delete it (previous version)
	err = destroyRoleBindingConfigMap(k, cl, reqLogger)
	if err != nil {
		return err
	}

	return nil
}

// Tries to see if the CLI route has been assigned a hostname.
func getCliRouteStatus(k *kabanerov1alpha2.Kabanero, reqLogger logr.Logger, c client.Client) (bool, error) {

	// Check that the route is accepted
	cliRoute := &routev1.Route{}
	cliRouteName := types.NamespacedName{Namespace: k.ObjectMeta.Namespace, Name: "kabanero-cli"}
	err := c.Get(context.TODO(), cliRouteName, cliRoute)
	if err == nil {
		k.Status.Cli.Hostnames = nil
		// Looking for an ingress that has an admitted status and a hostname
		for _, ingress := range cliRoute.Status.Ingress {
			routeAdmitted := false
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

// Deletes the role binding config map which may have existed in a prior version
func destroyRoleBindingConfigMap(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {

	// Check if the ConfigMap resource already exists.
	cmInstance := &corev1.ConfigMap{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      "kabanero-cli-role-config",
		Namespace: k.ObjectMeta.Namespace}, cmInstance)

	if err != nil {
		if errors.IsNotFound(err) == false {
			return err
		}

		// Not found.  Beautiful.
		return nil
	}

	// Need to delete it.
	reqLogger.Info(fmt.Sprintf("Attempting to delete CLI role binding config map: %v", cmInstance))
	err = c.Delete(context.TODO(), cmInstance)

	return err
}

// Creates the secret containing the AES encryption key used by the CLI.
func createEncryptionKeySecret(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	secretName := "kabanero-cli-aes-encryption-key-secret"

	// Check if the Secret already exists.
	secretInstance := &corev1.Secret{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      secretName,
		Namespace: k.ObjectMeta.Namespace}, secretInstance)

	if err != nil {
		if errors.IsNotFound(err) == false {
			return err
		}

		// Not found.  Make a new one.
		var ownerRef metav1.OwnerReference
		ownerRef, err = getOwnerReference(k, c, reqLogger)
		if err != nil {
			return err
		}

		secretInstance := &corev1.Secret{}
		secretInstance.ObjectMeta.Name = secretName
		secretInstance.ObjectMeta.Namespace = k.ObjectMeta.Namespace
		secretInstance.ObjectMeta.OwnerReferences = append(secretInstance.ObjectMeta.OwnerReferences, ownerRef)

		// Generate a 64 character random value
		possibleChars := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*()-=_+")
		maxVal := big.NewInt(int64(len(possibleChars)))
		var buf bytes.Buffer
		for i := 0; i < 64; i++ {
			curInt, randErr := rand.Int(rand.Reader, maxVal)
			if randErr != nil {
				return randErr
			}
			// Convert int to char
			buf.WriteByte(possibleChars[curInt.Int64()])
		}

		secretMap := make(map[string]string)
		secretMap["AESEncryptionKey"] = buf.String()
		secretInstance.StringData = secretMap

		reqLogger.Info(fmt.Sprintf("Attempting to create the CLI AES Encryption key secret"))
		err = c.Create(context.TODO(), secretInstance)
	}

	return err
}
