package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kabanero-io/kabanero-operator/pkg/apis"
	"github.com/kabanero-io/kabanero-operator/pkg/controller"

	knsapis "github.com/knative/serving-operator/pkg/apis"
	kneapis "github.com/openshift-knative/knative-eventing-operator/pkg/apis"
	routev1 "github.com/openshift/api/route/v1"
	consolev1 "github.com/openshift/api/console/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	tektonapis "github.com/openshift/tektoncd-pipeline-operator/pkg/apis"
	corev1 "k8s.io/api/core/v1"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
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

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))

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

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
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

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "kabanero-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
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

	if err := routev1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := kneapis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := knsapis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := corev1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := tektonapis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := consolev1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := operatorv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	
	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create Service object to expose the metrics port.
	_, err = metrics.ExposeMetricsPort(ctx, metricsPort)
	if err != nil {
		log.Info(err.Error())
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
