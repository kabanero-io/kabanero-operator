#!/bin/bash

set -Eeox pipefail

RELEASE="${RELEASE:-0.5.0}"
KABANERO_SUBSCRIPTIONS_YAML="${KABANERO_SUBSCRIPTIONS_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-subscriptions.yaml}"
KABANERO_CUSTOMRESOURCES_YAML="${KABANERO_CUSTOMRESOURCES_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-customresources.yaml}"
SLEEP_LONG="${SLEEP_LONG:-5}"
SLEEP_SHORT="${SLEEP_SHORT:-2}"

# Optional components (yes/no)
ENABLE_KAPPNAV="${ENABLE_KAPPNAV:-no}"


### Check prereqs

# oc installed
if ! which oc; then
  printf "oc client is required, please install oc client in your PATH.\nhttps://mirror.openshift.com/pub/openshift-v4/clients/oc/latest"
  exit 1
fi

# oc logged in
if ! oc whoami; then
  printf "Not logged in. Please login as a cluster-admin."
  exit 1
fi

# oc version
OCMIN="4.2.0"
OCVER=$(oc version -o=yaml | grep  gitVersion | head -1 | sed -nE 's/^[^0-9]*(([0-9]+\.)*[0-9]+).*/\1/p')
OCHEAD=$(printf "$OCMIN\n$OCVER" | sort -V | head -n 1)
if [ "$OCMIN" != "$OCHEAD" ]; then
  printf "oc client version is $OCVER. Minimum oc client version required is $OCMIN.\nhttps://mirror.openshift.com/pub/openshift-v4/clients/oc/latest".
  exit 1
fi

# Check to see if we're upgrading, and if so, that we're at N-1 or N.
if [ `oc get subscription kabanero-operator -n kabanero --no-headers --ignore-not-found | wc -l` -gt 0 ] ; then
		CSV=$(oc get subscription kabanero-operator -n kabanero --output=jsonpath={.status.installedCSV})
    if ! [[ "$CSV" =~ ^kabanero-operator\.v0\.[45]\..* ]]; then
        printf "Cannot upgrade kabanero-operator CSV version $CSV to $RELEASE.  Upgrade is supported from the previous minor release."
        exit 1
    fi
fi

# Check Subscriptions: subscription-name, namespace
checksub () {
	echo "Waiting for Subscription $1 InstallPlan to complete."

	# Wait 2 resync periods for OLM to emit new installplan
	sleep 60

	# Wait for the InstallPlan to be generated and available on status
	unset INSTALL_PLAN
	until oc get subscription $1 -n $2 --output=jsonpath={.status.installPlanRef.name}
	do
		sleep $SLEEP_SHORT
	done

	# Get the InstallPlan
	until [ -n "$INSTALL_PLAN" ]
	do
		sleep $SLEEP_SHORT
		INSTALL_PLAN=$(oc get subscription $1 -n $2 --output=jsonpath={.status.installPlanRef.name})
	done

	# Wait for the InstallPlan to Complete
	unset PHASE
	until [ "$PHASE" == "Complete" ]
	do
		PHASE=$(oc get installplan $INSTALL_PLAN -n $2 --output=jsonpath={.status.phase})
		sleep $SLEEP_SHORT
	done
	
	# Get installed CluserServiceVersion
	unset CSV
	until [ -n "$CSV" ]
	do
		sleep $SLEEP_SHORT
		CSV=$(oc get subscription $1 -n $2 --output=jsonpath={.status.installedCSV})
	done
	
	# Wait for the CSV
	unset PHASE
	until [ "$PHASE" == "Succeeded" ]
	do
		PHASE=$(oc get clusterserviceversion $CSV -n $2 --output=jsonpath={.status.phase})
		sleep $SLEEP_SHORT
	done
}

### Upgrade Prep

# ServiceMeshMemberRole
# serverless-operator.v1.2.0+ manages smmr, clean up
oc delete -f $KABANERO_CUSTOMRESOURCES_YAML --ignore-not-found --selector kabanero.io/install=21-cr-servicemeshmemberrole || true

# CatalogSource
# Delete previous CatalogSource to ensure visibility of updated catalog CSVs
oc delete -f $KABANERO_SUBSCRIPTIONS_YAML  --ignore-not-found --selector kabanero.io/install=00-catalogsource


### Install

### CatalogSource

# Stop the catalog-operator pod to avoid ready state bug
oc -n openshift-operator-lifecycle-manager scale deploy catalog-operator --replicas=0
sleep $SLEEP_LONG

# Install Kabanero CatalogSource
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=00-catalogsource

# Restart the catalog-operator pod to avoid ready state bug
sleep $SLEEP_LONG
oc -n openshift-operator-lifecycle-manager scale deploy catalog-operator --replicas=1




# Check the CatalogSource is ready
unset LASTOBSERVEDSTATE
until [ "$LASTOBSERVEDSTATE" == "READY" ]
do
	echo "Waiting for CatalogSource kabanero-catalog to be ready."
	LASTOBSERVEDSTATE=$(oc get catalogsource kabanero-catalog -n openshift-marketplace --output=jsonpath={.status.connectionState.lastObservedState})
	sleep $SLEEP_SHORT
done

### Subscriptions

# Install 10-subscription (elasticsearch, jaeger, kiali)
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=10-subscription

# Verify Subscriptions
checksub elasticsearch-operator openshift-operators
checksub jaeger-product openshift-operators
checksub kiali-ossm openshift-operators

# Install 11-subscription (servicemesh)
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=11-subscription

# Verify Subscriptions
checksub servicemeshoperator openshift-operators

# Install 12-subscription (eventing, serving)
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=12-subscription

# Verify Subscriptions
checksub knative-eventing-operator openshift-operators
checksub serverless-operator openshift-operators

# Install 13-subscription (pipelines, appsody)
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=13-subscription

# Verify Subscriptions
checksub openshift-pipelines openshift-operators
checksub appsody-operator-certified openshift-operators

# Install 14-subscription (che, kabanero)
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=14-subscription

# Verify Subscriptions
checksub eclipse-che kabanero
checksub kabanero-operator kabanero


### CustomResources

# ServiceMeshControlplane
oc apply -f $KABANERO_CUSTOMRESOURCES_YAML --selector kabanero.io/install=20-cr-servicemeshcontrolplane

# Check the ServiceMeshControlPlane is ready, last condition should reflect readiness
unset STATUS
unset TYPE
until [ "$STATUS" == "True" ] && [ "$TYPE" == "Ready" ]
do
	echo "Waiting for ServiceMeshControlPlane basic-install to be ready."
	TYPE=$(oc get servicemeshcontrolplane -n istio-system basic-install --output=jsonpath={.status.conditions[-1:].type})
	STATUS=$(oc get servicemeshcontrolplane -n istio-system basic-install --output=jsonpath={.status.conditions[-1:].status})
	sleep $SLEEP_SHORT
done


# Serving
oc apply -f $KABANERO_CUSTOMRESOURCES_YAML --selector kabanero.io/install=22-cr-knative-serving

# Check the KnativeServing is ready, last condition should reflect readiness
unset STATUS
unset TYPE
until [ "$STATUS" == "True" ] && [ "$TYPE" == "Ready" ]
do
	echo "Waiting for KnativeServing knative-serving to be ready."
	TYPE=$(oc get knativeserving knative-serving -n knative-serving --output=jsonpath={.status.conditions[-1:].type})
	STATUS=$(oc get knativeserving knative-serving -n knative-serving --output=jsonpath={.status.conditions[-1:].status})
	sleep $SLEEP_SHORT
done

# Github Sources
oc apply -f https://github.com/knative/eventing-contrib/releases/download/v0.9.0/github.yaml

# Need to wait for knative serving CRDs before installing tekton webhook extension
until oc get crd services.serving.knative.dev 
do
	echo "Waiting for CustomResourceDefinition services.serving.knative.dev to be ready."
	sleep $SLEEP_SHORT
done

# Tekton Dashboard
oc new-project tekton-pipelines || true

openshift_master_default_subdomain=$(oc get ingresses.config.openshift.io cluster --output=jsonpath={.spec.domain})

curl -s -L https://github.com/tektoncd/dashboard/releases/download/v0.3.0/openshift-tekton-webhooks-extension-release.yaml | sed "s/{openshift_master_default_subdomain}/${openshift_master_default_subdomain}/" | oc apply -f -
oc apply -f https://github.com/tektoncd/dashboard/releases/download/v0.3.0/dashboard-latest-openshift-tekton-dashboard-release.yaml

# Network policy for kabanero and tekton pipelines namespaces
oc apply -f $KABANERO_CUSTOMRESOURCES_YAML --selector kabanero.io/install=23-cr-network-policy

# Install KAppNav if selected
if [ "$ENABLE_KAPPNAV" == "yes" ]
then
  oc apply -f https://raw.githubusercontent.com/kabanero-io/kabanero-operator/${RELEASE}/deploy/optional.yaml --selector=kabanero.io/component=kappnav
fi

# Create service account to used by pipelines
oc apply -f $KABANERO_CUSTOMRESOURCES_YAML --selector kabanero.io/install=24-pipeline-sa

# Role used by the collection controller to manipulate triggers in the
# tekton-pipelines namespace (for use by tekton github webhooks extension)
oc apply -f $KABANERO_CUSTOMRESOURCES_YAML --selector kabanero.io/install=25-triggers-role

# Install complete.  give instructions for how to create an instance.
SAMPLE_KAB_INSTANCE_URL=https://raw.githubusercontent.com/kabanero-io/kabanero-operator/${RELEASE}/config/samples/default.yaml

# Turn off debugging, and wait 3 seconds for it to flush output, before
# printing instructions.
set +x
sleep 3
echo "***************************************************************************"
echo "*                                                                          "
echo "*  The installation script is complete.  You can now create an instance    "
echo "*  of the Kabanero CR.  If you have cloned and curated a collection set,   "
echo "*  apply the Kabanero CR that you created.  Or, to create the default      "
echo "*  instance:                                                               "
echo "*                                                                          "
echo "*      oc apply -n kabanero -f ${SAMPLE_KAB_INSTANCE_URL}                  "
echo "*                                                                          "
echo "***************************************************************************"
