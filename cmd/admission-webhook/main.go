package main

import (
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kabanero-io/kabanero-operator/pkg/apis"
	collectionwebhook "github.com/kabanero-io/kabanero-operator/pkg/webhook/collection"
	kabanerowebhookv1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/webhook/kabanero/v1alpha1"
	kabanerowebhookv1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/webhook/kabanero/v1alpha2"
	stackwebhook "github.com/kabanero-io/kabanero-operator/pkg/webhook/stack"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
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
	logf.SetLogger(zap.Logger(false))

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
		Namespace: namespace,
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

	// Setup the webhook server
	hookServer := mgr.GetWebhookServer()
	hookServer.Port = 9443
	hookServer.Register("/validate-collections", collectionwebhook.BuildValidatingWebhook(&mgr))
	hookServer.Register("/mutate-collections", collectionwebhook.BuildMutatingWebhook(&mgr))
	hookServer.Register("/validate-kabaneros", kabanerowebhookv1alpha1.BuildValidatingWebhook(&mgr))
	hookServer.Register("/mutate-kabaneros", kabanerowebhookv1alpha1.BuildMutatingWebhook(&mgr))
	hookServer.Register("/validate-kabaneros/v1alpha2", kabanerowebhookv1alpha2.BuildValidatingWebhook(&mgr))
	hookServer.Register("/validate-stacks", stackwebhook.BuildValidatingWebhook(&mgr))
	hookServer.Register("/mutate-stacks", stackwebhook.BuildMutatingWebhook(&mgr))

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
