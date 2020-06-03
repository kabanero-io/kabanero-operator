package main

import (
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kabanero-io/kabanero-operator/pkg/apis"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/stack"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	//imagev1 "github.com/openshift/api/image/v1"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)
var log = logf.Log.WithName("cmd")

// These variables are injected during the build using ldflags
var GitTag string
var GitCommit string
var GitRepoSlug string
var BuildDate string

func printStackControllerData() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))

	if len(GitTag) > 0 {
		log.Info(fmt.Sprintf("kabanero-stack-operator Git tag: %s", GitTag))
	}

	if len(GitCommit) > 0 {
		log.Info(fmt.Sprintf("kabanero-stack-operator Git commit: %s", GitCommit))
	}

	if len(GitRepoSlug) > 0 {
		log.Info(fmt.Sprintf("kabanero-stack-operator Git repository: %s", GitRepoSlug))
	}

	if len(BuildDate) == 0 {
		BuildDate = "unspecified"
	}
	log.Info(fmt.Sprintf("kabanero-stack-operator build date: %s", BuildDate))
}

func main() {
	logf.SetLogger(zap.Logger(false))

	printStackControllerData()

	namespace, err := getStackControllerNamespace()
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

	if err := pipelinev1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

// Use pr to work around issue
// https://github.com/kubernetes-sigs/controller-runtime/issues/362
// https://github.com/openshift/api/issues/270
// https://github.com/openshift/api/pull/461
	if err := AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

/*
	if err := imagev1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
*/
	// Setup all Controllers
	if err := stack.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

// Returns the namespace the stack controller is running in.
func getStackControllerNamespace() (string, error) {
	ns, found := os.LookupEnv("KABANERO_NAMESPACE")
	if !found {
		return "", fmt.Errorf("KABANERO_NAMESPACE must be set as an environment variable")
	}
	return ns, nil
}
