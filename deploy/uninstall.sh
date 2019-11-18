#!/bin/bash

set -x pipefail

# By default, this script will remove all instances of AppsodyApplication
# from the cluster, and delete the CRD.  To prevent this, comment the
# following line.
APPSODY_UNINSTALL=1

RELEASE="${RELEASE:-0.3.0}"
KABANERO_SUBSCRIPTIONS_YAML="${KABANERO_SUBSCRIPTIONS_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-subscriptions.yaml}"
KABANERO_CUSTOMRESOURCES_YAML="${KABANERO_CUSTOMRESOURCES_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-customresources.yaml}"
SLEEP_LONG="${SLEEP_LONG:-5}"
SLEEP_SHORT="${SLEEP_SHORT:-2}"



### CustomResources

# If we're completely removing Appsody, make sure all instances of the
# Appsody application CRD are deleted.  This gives the Appsody operator
# a chance to process any finalizers which may be set, before the operator
# is removed (by removing the Kabanero instance, later).
if [ "$APPSODY_UNINSTALL" -eq 1 ] ; then

    # Make sure the Appsody CRD still exists...
    if [ `oc get crds appsodyapplications.appsody.dev --no-headers --ignore-not-found | wc -l` -gt 0 ] ; then

        # Delete any "Kind: AppsodyApplication" objects in this cluster.  Print
        # a list of each instance along with its namespace.  Then delete them 
        # one by one.
        oc get AppsodyApplication --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace --no-headers --ignore-not-found | while read APP_NAME APP_NAMESPACE; do oc delete AppsodyApplication $APP_NAME --namespace $APP_NAMESPACE; done

        # Wait for all of the application instances to be deleted.  We don't 
        # want to delete the Appsody operator until the operator has had a
        # chance to process its finalizer.
        echo "Waiting for AppsodyApplication instances to be deleted...."
        LOOP_COUNT=0
        while [ `oc get AppsodyApplication --all-namespaces | wc -l` -gt 0 ]
        do
            sleep 5
            LOOP_COUNT=`expr $LOOP_COUNT + 1`
            if [ $LOOP_COUNT -gt 10 ] ; then
                echo "Timed out waiting for AppsodyApplication instances to be deleted"
                exit 1
            fi
        done
    fi
fi

# Remove KAppNav if it was installed by the Kabanero install script.  Delete the instances first, allowing the
# operator to run its finalizer.  Then delete the operator, then the cluster level resources.
if [ `oc get crds kappnavs.kappnav.io --no-headers --ignore-not-found | wc -l` -gt 0 ] ; then 
    oc delete kappnavs --selector=kabanero.io/component=kappnav --namespace kappnav --ignore-not-found

    # Wait for the kappnav instances to be deleted, to give the kappnav operator a chance to
    # process its finalizer.
    echo "Waiting for KAppNav instances to stop...."
    LOOP_COUNT=0
    while [ `oc get kappnav --namespace kappnav --selector=kabanero.io/component=kappnav --no-headers --ignore-not-found | wc -l` -gt 0 ]
    do
        sleep 5
        LOOP_COUNT=`expr $LOOP_COUNT + 1`
        if [ $LOOP_COUNT -gt 20 ] ; then
            echo "Timed out waiting for KAppNav instances to stop"
            exit 1
        fi
    done

fi
oc delete serviceaccounts,deployments --selector=kabanero.io/component=kappnav --namespace kappnav --ignore-not-found
oc delete clusterroles,clusterrolebindings,crds --selector=kabanero.io/component=kappnav --ignore-not-found
oc delete namespaces --selector=kabanero.io/component=kappnav --ignore-not-found

# Github Sources
oc delete --all-namespaces=true githubsources.sources.eventing.knative.dev --all
echo "Waiting for githubsources instances to be deleted...."
LOOP_COUNT=0
while [ `oc get githubsources.sources.eventing.knative.dev --all-namespaces | wc -l` -gt 0 ]
do
  sleep $SLEEP_LONG
  LOOP_COUNT=`expr $LOOP_COUNT + 1`
  if [ $LOOP_COUNT -gt 10 ] ; then
    echo "Timed out waiting for githubsources.sources.eventing.knative.dev instances to be deleted"
    exit 1
  fi
done
oc delete -f https://github.com/knative/eventing-contrib/releases/download/v0.9.0/github.yaml


# Delete CustomResources, do not delete namespaces , which can lead to finalizer problems.
oc delete -f $KABANERO_CUSTOMRESOURCES_YAML --ignore-not-found --selector kabanero.io/install=23-cr-network-policy,kabanero.io/namespace!=true
oc delete -f $KABANERO_CUSTOMRESOURCES_YAML --ignore-not-found --selector kabanero.io/install=22-cr-knative-serving,kabanero.io/namespace!=true
oc delete -f $KABANERO_CUSTOMRESOURCES_YAML --ignore-not-found --selector kabanero.io/install=21-cr-servicemeshmemberrole,kabanero.io/namespace!=true
oc delete -f $KABANERO_CUSTOMRESOURCES_YAML --ignore-not-found --selector kabanero.io/install=20-cr-servicemeshcontrolplane,kabanero.io/namespace!=true

# Delete service account to used by pipelines
oc delete -f $KABANERO_CUSTOMRESOURCES_YAML --ignore-not-found --selector kabanero.io/install=24-pipeline-sa,kabanero.io/namespace!=true


# CRDs still exist
if [ `oc get crds kabaneros.kabanero.io --no-headers --ignore-not-found | wc -l` -gt 0 ] ; then 

    # Delete any "Kind: Kabanero" objects in this cluster.  Print a list of
    # each Kabanero instance along with its namespace.  Then delete them one
    # by one.
    oc get kabaneros --all-namespaces -o=custom-columns=NAME:.metadata.name,NAMESPACE:.metadata.namespace --no-headers --ignore-not-found | while read KAB_NAME KAB_NAMESPACE; do oc delete kabanero $KAB_NAME --namespace $KAB_NAMESPACE; done

    # Wait for all of the Kabanero instances to be deleted.  We don't want to
    # delete the Kabanero operator until the operator has had a chance to
    # process its finalizer.
    echo "Waiting for Kabanero instances to be deleted...."
    LOOP_COUNT=0
    while [ `oc get kabaneros --all-namespaces | wc -l` -gt 0 ]
    do
        sleep 5
        LOOP_COUNT=`expr $LOOP_COUNT + 1`
        if [ $LOOP_COUNT -gt 10 ] ; then
            echo "Timed out waiting for Kabanero instances to be deleted"
            exit 1
        fi
    done
fi


# Tekton Dashboard
oc delete --ignore-not-found -f https://github.com/tektoncd/dashboard/releases/download/v0.2.1/openshift-tekton-dashboard-release.yaml
oc delete --ignore-not-found -f https://github.com/tektoncd/dashboard/releases/download/v0.2.1/openshift-tekton-webhooks-extension-release.yaml

### Subscriptions

# Args: subscription metadata.name, namespace
# Deletes Subscription, InstallPlan, CSV
unsubscribe () {
	# Get InstallPlan
	INSTALL_PLAN=$(oc get subscription $1 -n $2 --output=jsonpath={.status.installPlanRef.name})
	
	# Get CluserServiceVersion
	CSV=$(oc get subscription $1 -n $2 --output=jsonpath={.status.installedCSV})
	
	# Delete Subscription 
	oc delete subscription $1 -n $2

	# Delete the InstallPlan
	oc delete installplan $INSTALL_PLAN -n $2
	
	# Delete the Installed ClusterServiceVersion
	oc delete clusterserviceversion $CSV -n $2
	
	# Waat for the Copied ClusterServiceVersions to cleanup
	if [ -n "$CSV" ] ; then
		while [ `oc get clusterserviceversions $CSV --all-namespaces | wc -l` -gt 0 ]
		do
			sleep 5
			LOOP_COUNT=`expr $LOOP_COUNT + 1`
			if [ $LOOP_COUNT -gt 10 ] ; then
					echo "Timed out waiting for Copied ClusterServiceVersions $CSV to be deleted"
					exit 1
			fi
		done
	fi
}

unsubscribe kabanero-operator kabanero

unsubscribe serverless-operator openshift-operators

unsubscribe openshift-pipelines-operator-dev-preview-community-operators-openshift-marketplace openshift-operators

unsubscribe knative-eventing-operator-alpha-community-operators-openshift-marketplace openshift-operators

unsubscribe appsody-operator-certified-beta-certified-operators-openshift-marketplace openshift-operators

unsubscribe servicemeshoperator openshift-operators

unsubscribe kiali-ossm openshift-operators

unsubscribe jaeger-product openshift-operators

unsubscribe elasticsearch-operator openshift-operators

unsubscribe eclipse-che kabanero

# Delete OperatorGroup
oc delete -n kabanero operatorgroup kabanero

# Delete CatalogSource
oc delete -n openshift-marketplace catalogsource kabanero-catalog


# Ensure CSV Cleanup in all namespaces
OPERATORS=(appsody-operator jaeger-operator kiali-operator knative-eventing-operator openshift-pipelines-operator servicemeshoperator elasticsearch-operator)
for OPERATOR in "${OPERATORS[@]}"
do
  CSV=$(oc --all-namespaces=true get csv --output=jsonpath={.items[*].metadata.name} | tr " " "\n" | grep ${OPERATOR} | head -1)
  if [ -n "${CSV}" ]; then
    oc --all-namespaces=true delete csv --field-selector=metadata.name="${CSV}"
  fi
done

# Cleanup from the openshift service mesh readme
oc delete validatingwebhookconfiguration/openshift-operators.servicemesh-resources.maistra.io
oc delete -n openshift-operators daemonset/istio-node
oc delete clusterrole/istio-admin
oc get crds -o name | grep '.*\.istio\.io' | xargs -r -n 1 oc delete
oc get crds -o name | grep '.*\.maistra\.io' | xargs -r -n 1 oc delete
