DEST=knative-eventing.yaml

BASEURL=https://raw.githubusercontent.com/openshift-knative/knative-eventing-operator/master/deploy
$(dirname $0)/get_operator_config.sh $BASEURL $DEST

curl $BASEURL/crds/eventing_v1alpha1_knativeeventing_crd.yaml -o eventing_v1alpha1_knativeeventing_crd.yaml
cat eventing_v1alpha1_knativeeventing_crd.yaml >> $DEST; echo "---" >> $DEST
rm eventing_v1alpha1_knativeeventing_crd.yaml

sed -i '' 's/namespace: default/namespace: kabanero/' $DEST
