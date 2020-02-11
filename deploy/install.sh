#!/bin/bash

set -Eeox pipefail

RELEASE="${RELEASE:-0.6.0}"
KABANERO_SUBSCRIPTIONS_YAML="${KABANERO_SUBSCRIPTIONS_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-subscriptions.yaml}"
KABANERO_CUSTOMRESOURCES_YAML="${KABANERO_CUSTOMRESOURCES_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-customresources.yaml}"
SLEEP_LONG="${SLEEP_LONG:-5}"
SLEEP_SHORT="${SLEEP_SHORT:-2}"

# Optional components (yes/no)
ENABLE_KAPPNAV="${ENABLE_KAPPNAV:-no}"

# Find we are running on MAC.
MAC_EXEC=false
if [ "$(uname -s)" == "Darwin" ]; then
   MAC_EXEC=true
fi

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

SEDOPT="nre"
if [[ $MAC_EXEC == true ]]; then
  SEDOPT="nE"
fi

OCVER=$(oc version -o=yaml | grep  gitVersion | head -1 | sed -$SEDOPT 's/^[^0-9]*(([0-9]+\.)*[0-9]+).*/\1/p')
OCHEAD=$(printf "$OCMIN\n$OCVER" | sort -V | head -n 1)
if [ "$OCMIN" != "$OCHEAD" ]; then
  printf "oc client version is $OCVER. Minimum oc client version required is $OCMIN.\nhttps://mirror.openshift.com/pub/openshift-v4/clients/oc/latest".
  exit 1
fi

# Check to see if we're upgrading, and if so, that we're at N-1 or N.
if [ `oc get subscription kabanero-operator -n kabanero --no-headers --ignore-not-found | wc -l` -gt 0 ] ; then
		CSV=$(oc get subscription kabanero-operator -n kabanero --output=jsonpath={.status.installedCSV})
    if ! [[ "$CSV" =~ ^kabanero-operator\.v0\.[56]\..* ]]; then
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
    if [ "$PHASE" == "Failed" ]; then
      set +x
      sleep 3
      echo "InstallPlan $INSTALL_PLAN for subscription $1 failed."
      echo "To investigate the reason of the InstallPlan failure run:"
      echo "oc describe installplan $INSTALL_PLAN -n $2"
      exit 1
    fi
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
    if [ "$PHASE" == "Failed" ]; then
      set +x
      sleep 3
      echo "ClusterServiceVersion $CSV for subscription $1 failed."
      echo "To investigate the reason of the ClusterServiceVersion failure run:"
      echo "oc describe clusterserviceversion $CSV -n $2"
      exit 1
    fi
		sleep $SLEEP_SHORT
	done
}

# Removes a resource instance in the specified namespace.
removeResourceInstance() {
	if [ `oc get crds $1 --no-headers --ignore-not-found | wc -l` -gt 0 ] ; then 
        # Delete an input Kind object in a specific namespace.  
        oc delete $2 $3 -n $4 --ignore-not-found

        # Wait for the specified instance to be deleted.
	echo "Waiting for $2 instance ($3) in the $4 namespace to be deleted...."
    	LOOP_COUNT=0
   		while [ `oc get $2 -n $4 | wc -l` -gt 0 ]
    	do
        	sleep 5
        	LOOP_COUNT=`expr $LOOP_COUNT + 1`
        	if [ $LOOP_COUNT -gt 10 ] ; then
           		echo "Timed out waiting for $2 instance ($3) in the $4 namespace to be deleted"
           		exit 1
        	fi
    	done
    fi
} 

# Remove subscription and associated resources if found.
unsubscribe () {
	# Get CluserServiceVersion
	CSV=$(oc get subscription $1 -n $2 --ignore-not-found=true --output=jsonpath={.status.installedCSV})

	# Delete Subscription 
	oc delete subscription $1 -n $2 --ignore-not-found=true

	if [ -n "$CSV" ] ; then
		# Delete the Installed ClusterServiceVersion
		oc delete clusterserviceversion $CSV -n $2
	
		# Wait for the Copied ClusterServiceVersions to cleanup
		while [ `oc get clusterserviceversions --all-namespaces --field-selector=metadata.name=$CSV | wc -l` -gt 0 ]
		do
			sleep $SLEEP_LONG
			LOOP_COUNT=`expr $LOOP_COUNT + 1`
			if [ $LOOP_COUNT -gt 10 ] ; then
					echo "Timed out waiting for Copied ClusterServiceVersions $CSV to be deleted"
				break
			fi
		done
	fi
}


### Upgrade Prep

# ServiceMeshMemberRole
# serverless-operator.v1.2.0+ manages smmr, clean up
oc delete -f $KABANERO_CUSTOMRESOURCES_YAML --ignore-not-found --selector kabanero.io/install=21-cr-servicemeshmemberrole || true

# ServiceMeshControlPlane
# serverless-operator.v1.3.0+ manages smmr & smcp, cleanup
oc delete -f $KABANERO_CUSTOMRESOURCES_YAML --ignore-not-found --selector kabanero.io/install=20-cr-servicemeshcontrolplane || true


# CatalogSource
# Delete previous CatalogSource to ensure visibility of updated catalog CSVs
oc delete -f $KABANERO_SUBSCRIPTIONS_YAML  --ignore-not-found --selector kabanero.io/install=00-catalogsource
sleep $SLEEP_LONG


### Install

### CatalogSource

# Install Kabanero CatalogSource
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=00-catalogsource
sleep $SLEEP_LONG


# Check the kabanero-catalog CatalogSource pod is Running
unset STATUSALL
until [ "$STATUSALL" == "Running" ]
do
  echo "Waiting for CatalogSource kabanero-catalog pod to be ready."
  STATUSES=$(oc -n openshift-marketplace get pod -l olm.catalogSource=kabanero-catalog --output=jsonpath={.items[*].status.phase})
  for STATUS in ${STATUSES[@]}
  do
    if [ "$STATUS" != "Running" ]; then
      STATUSALL=$STATUS
      break
    fi
    STATUSALL=$STATUS
  done
  sleep $SLEEP_SHORT
done

# Stop the catalog-operator pod to avoid ready state bug
oc -n openshift-operator-lifecycle-manager scale deploy catalog-operator --replicas=0
sleep $SLEEP_LONG

# Restart the catalog-operator pod to avoid ready state bug
sleep $SLEEP_LONG
oc -n openshift-operator-lifecycle-manager scale deploy catalog-operator --replicas=1

# Check the kabanero-catalog CatalogSource is ready
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

# Install 12-subscription (serving)
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=12-subscription

# Verify Subscriptions
checksub serverless-operator openshift-operators

# Install 13-subscription (pipelines, appsody)
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=13-subscription

# Verify Subscriptions
checksub openshift-pipelines openshift-operators
checksub appsody-operator-certified openshift-operators

# Install 14-subscription (codeready-workspaces, kabanero)

# Eclipse-che operator and Codeready-workspaces operator cannot be both installed at the same 
# time in the same namespace. The kabanero operator installation will attempt to uninstall 
# the eclipe-che CR instance (codewind-che) and Eclipse-che operator previously 
# installed by the kabanero operator.
# If the current installation of Eclipse-che was not done through the kabanero operator/install, 
# it must be uninstalled prior to running the kabanero installation script. Furthermore, users 
# need to be sure to delete any operatorgroups other than kabanero (i.e. kabanero-5pn6b) in 
# the kabanero namespace that may have been created as result of a manual Eclipse-che installation.
removeResourceInstance checlusters.org.eclipse.che CheCluster codewind-che kabanero
unsubscribe eclipse-che kabanero

# Codewind is required to run as privileged and as root because it builds container images
oc adm policy add-scc-to-user anyuid system:serviceaccount:kabanero:che-workspace
oc adm policy add-scc-to-user privileged system:serviceaccount:kabanero:che-workspace

# Apply codeready-workspaces and kabanero subscriptions.
oc apply -f $KABANERO_SUBSCRIPTIONS_YAML --selector kabanero.io/install=14-subscription

# Verify Subscriptions
checksub codeready-workspaces kabanero 
checksub kabanero-operator kabanero

### CustomResources

# Serving
oc apply -f $KABANERO_CUSTOMRESOURCES_YAML --selector kabanero.io/install=22-cr-knative-serving

# Check the KnativeServing is ready, last condition should reflect readiness
unset STATUS
unset TYPE
until [ "$STATUS" == "True" ] && [ "$TYPE" == "Ready" ]
do
	echo "Waiting for KnativeServing knative-serving to be ready."
	TYPE=$(oc get knativeserving.serving.knative.dev knative-serving -n knative-serving --output=jsonpath={.status.conditions[-1:].type})
	STATUS=$(oc get knativeserving.serving.knative.dev knative-serving -n knative-serving --output=jsonpath={.status.conditions[-1:].status})
	sleep $SLEEP_SHORT
done


# Need to wait for knative serving CRDs before installing tekton webhook extension
until oc get crd services.serving.knative.dev 
do
	echo "Waiting for CustomResourceDefinition services.serving.knative.dev to be ready."
	sleep $SLEEP_SHORT
done

# Tekton Dashboard
oc new-project tekton-pipelines || true

openshift_master_default_subdomain=$(oc get ingresses.config.openshift.io cluster --output=jsonpath={.spec.domain})

curl -s -L https://github.com/tektoncd/dashboard/releases/download/v0.5.1/openshift-tekton-webhooks-extension-release.yaml | sed "s/{openshift_master_default_subdomain}/${openshift_master_default_subdomain}/" | oc apply -f -
oc apply -f https://github.com/tektoncd/dashboard/releases/download/v0.5.1/openshift-tekton-dashboard-release.yaml

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
