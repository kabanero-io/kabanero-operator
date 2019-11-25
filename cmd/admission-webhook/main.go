package main

import (
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kabanero-io/kabanero-operator/pkg/apis"
	collectionwebhook "github.com/kabanero-io/kabanero-operator/pkg/webhook/collection"

	apitypes "k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var log = logf.Log.WithName("cmd")

// These variables are injected during the build using ldflags
var GitTag string
var GitCommit string
var GitRepoSlug string
var BuildDate string

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))

	if len(GitTag) > 0 {
		log.Info(fmt.Sprintf("kabanero-operator Git tag: %s", GitTag))
	}

	if len(GitCommit) > 0 {
		log.Info(fmt.Sprintf("kabanero-operator Git commit: %s", GitCommit))
	}

	if len(GitRepoSlug) > 0 {
		log.Info(fmt.Sprintf("kabanero-operator Git repository: %s", GitRepoSlug))
	}

	if len(BuildDate) == 0 {
		BuildDate = "unspecified"
	}
	log.Info(fmt.Sprintf("kabanero-operator build date: %s", BuildDate))
}

// GetHookNamespace returns the namespace the webhooks are running in
func getHookNamespace() (string, error) {
	ns, found := os.LookupEnv("KABANERO_NAMESPACE")
	if !found {
		return "", fmt.Errorf("KABANERO_NAMESPACE must be set")
	}
	return ns, nil
}

func main() {
	logf.SetLogger(logf.ZapLogger(false))

	printVersion()

	namespace, err := getHookNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create the validating webhook
	validatingWebhook, err := collectionwebhook.BuildValidatingWebhook(&mgr)
	if err != nil {
		log.Error(err, "unable to setup validating webhook")
		os.Exit(1)
	}

	// Create the mutating webhook
	mutatingWebhook, err := collectionwebhook.BuildMutatingWebhook(&mgr)
	if err != nil {
		log.Error(err, "unable to setup mutating webhook")
		os.Exit(1)
	}

	// Start the webhook server.  Some things to note:
	// 1) A webhook server requires certificates.  The controller-runtime is
	//    creating a secret and generates a certificate within it.  This allows
	//    the Kube API server to use TLS when calling the webhook.
	// 2) The controller-runtime is auto-generating the secret, service, and
	//    configurations (ValidatingWebhookConfiguration and
	//    MutatingWebhookConfiguration) used here.
	disableWebhookConfigInstaller := false
	admissionServer, err := webhook.NewServer("collection-admission-server", mgr, webhook.ServerOptions{
		Port: 9443,
		CertDir: "/tmp/cert",
		DisableWebhookConfigInstaller: &disableWebhookConfigInstaller,
		BootstrapOptions: &webhook.BootstrapOptions{
			MutatingWebhookConfigName: "webhook.operator.kabanero.io",
			ValidatingWebhookConfigName: "webhook.operator.kabanero.io",
			Secret: &apitypes.NamespacedName{
				Namespace: namespace, // TODO: appropriate namespace
				Name: "kabanero-operator-admission-webhook",
			},
			Service: &webhook.Service{
				Namespace: namespace, // TODO: appropriate namespace
				Name: "kabanero-operator-admission-webhook",
				Selectors: map[string]string{
					"name": "kabanero-operator-admission-webhook",
				},
			},
		},
	})
	if err != nil {
		log.Error(err, "unable to create a new webhook server")
		os.Exit(1)
	}

	err = admissionServer.Register(validatingWebhook, mutatingWebhook)
	if err != nil {
		log.Error(err, "unable to register webhooks in the admission server")
		os.Exit(1)
	}
	
	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
