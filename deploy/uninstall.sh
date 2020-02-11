#!/bin/bash

set -x pipefail

# By default, this script will remove all instances of AppsodyApplication
# from the cluster, and delete the CRD.  To prevent this, comment the
# following line.
APPSODY_UNINSTALL=1

RELEASE="${RELEASE:-0.5.0}"
KABANERO_SUBSCRIPTIONS_YAML="${KABANERO_SUBSCRIPTIONS_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-subscriptions.yaml}"
KABANERO_CUSTOMRESOURCES_YAML="${KABANERO_CUSTOMRESOURCES_YAML:-https://github.com/kabanero-io/kabanero-operator/releases/download/$RELEASE/kabanero-customresources.yaml}"
SLEEP_LONG="${SLEEP_LONG:-15}"
SLEEP_SHORT="${SLEEP_SHORT:-2}"


# CRD spec.group suffixes of interest
CRDS=(appsody.dev kabanero.io tekton.dev knative.dev istio.io maistra.io kiali.io jaegertracing.io)


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
OCVER=$(oc version -o=yaml | grep  gitVersion | head -1 | sed -nre 's/^[^0-9]*(([0-9]+\.)*[0-9]+).*/\1/p')
OCHEAD=$(printf "$OCMIN\n$OCVER" | sort -V | head -n 1)
if [ "$OCMIN" != "$OCHEAD" ]; then
  printf "oc client version is $OCVER. Minimum oc client version required is $OCMIN.\nhttps://mirror.openshift.com/pub/openshift-v4/clients/oc/latest".
  exit 1
fi

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

# Delete the Role used by the collection controller to manipulate triggers
oc delete --ignore-not-found -f $KABANERO_CUSTOMRESOURCES_YAML --selector kabanero.io/install=25-triggers-role

# Tekton Dashboard
oc delete --ignore-not-found -f https://github.com/tektoncd/dashboard/releases/download/v0.5.1/openshift-tekton-webhooks-extension-release.yaml
oc delete --ignore-not-found -f https://github.com/tektoncd/dashboard/releases/download/v0.5.1/openshift-tekton-dashboard-release.yaml


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

# Delete CRs
for CRD in "${CRDS[@]}"
do
  unset CRDNAMES
  CRDNAMES=$(oc get crds -o=jsonpath='{range .items[*]}{"\n"}{@.metadata.name}{end}' | grep '.*\.'$CRD)
  if [ -n "$CRDNAMES" ]; then
    echo $CRDNAMES | xargs -n 1 oc delete --all --all-namespaces=true
  fi
done


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
	
	# Delete the Installed ClusterServiceVersion
	oc delete clusterserviceversion $CSV -n $2
	
	# Wait for the Copied ClusterServiceVersions to cleanup
	if [ -n "$CSV" ] ; then
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

unsubscribe kabanero-operator kabanero

unsubscribe serverless-operator openshift-operators

unsubscribe openshift-pipelines openshift-operators

unsubscribe appsody-operator-certified openshift-operators

unsubscribe servicemeshoperator openshift-operators

unsubscribe kiali-ossm openshift-operators

unsubscribe jaeger-product openshift-operators

unsubscribe elasticsearch-operator openshift-operators

# Unsubscribe the codereay-workspaces operator. Note that all codeready-workspace operator instances
# are deleted when the kabanero instance that created it is deleted.
unsubscribe codeready-workspaces kabanero

# Remove codewind privileges added during installation.
oc adm policy remove-scc-from-user anyuid system:serviceaccount:kabanero:che-workspace
oc adm policy remove-scc-from-user privileged system:serviceaccount:kabanero:che-workspace

# Delete OperatorGroup
oc delete -n kabanero operatorgroup kabanero

# Delete CatalogSource
oc delete -n openshift-marketplace catalogsource kabanero-catalog


# Ensure CSV Cleanup in all namespaces in case OLM GC failed to delete Copies
OPERATORS=(appsody-operator jaeger-operator kiali-operator openshift-pipelines-operator serverless-operator servicemeshoperator elasticsearch-operator)
for OPERATOR in "${OPERATORS[@]}"
do
  CSV=$(oc --all-namespaces=true get csv --output=jsonpath='{range .items[*]}{"\n"}{@.metadata.name}{end}' | grep ${OPERATOR} | head -1)
  if [ -n "${CSV}" ]; then
    oc --all-namespaces=true delete csv --field-selector=metadata.name="${CSV}"
  fi
done

# Cleanup from the openshift service mesh readme
oc delete validatingwebhookconfiguration/openshift-operators.servicemesh-resources.maistra.io --ignore-not-found
oc delete -n openshift-operators daemonset/istio-node --ignore-not-found
oc delete clusterrole/istio-admin --ignore-not-found

# Delete CRDs
for CRD in "${CRDS[@]}"
do
  unset CRDNAMES
  CRDNAMES=$(oc get crds -o=jsonpath='{range .items[*]}{"\n"}{@.metadata.name}{end}' | grep '.*\.'$CRD)
  if [ -n "$CRDNAMES" ]; then
    echo $CRDNAMES | xargs -n 1 oc delete crd
  fi
done


# Delete tekton-pipelines namespace to clean up loose artifacts from dashboard that cause problems on reinstall
oc delete namespace tekton-pipelines --ignore-not-found
