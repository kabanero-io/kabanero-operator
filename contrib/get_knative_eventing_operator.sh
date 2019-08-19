#!/bin/bash
set -Eeuox pipefail

DEST=knative-eventing.yaml

RELEASE=v0.6.0

BASEURL=https://raw.githubusercontent.com/openshift-knative/knative-eventing-operator/${RELEASE}/deploy
$(dirname $0)/get_operator_config.sh $BASEURL $DEST

curl -f $BASEURL/crds/eventing_v1alpha1_knativeeventing_crd.yaml -o eventing_v1alpha1_knativeeventing_crd.yaml
cat eventing_v1alpha1_knativeeventing_crd.yaml >> $DEST; echo "---" >> $DEST
rm eventing_v1alpha1_knativeeventing_crd.yaml

sed -i.bak 's/namespace: default/namespace: kabanero/g' $DEST
rm $DEST.bak
