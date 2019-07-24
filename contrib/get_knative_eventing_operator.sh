TAG=v0.7.1
DEST=knative-eventing.yaml
BASEURL=https://raw.githubusercontent.com/openshift-knative/knative-eventing-operator/$TAG/deploy

curl $BASEURL/operator.yaml -o $DEST; echo "---" >> $DEST
curl $BASEURL/role.yaml >> $DEST; echo "---" >> $DEST
curl $BASEURL/role_binding.yaml >> $DEST; echo "---" >> $DEST
curl $BASEURL/service_account.yaml >> $DEST; echo "---" >> $DEST
curl $BASEURL/crds/operator_v1alpha1_config_crd.yaml >> $DEST; echo "---" >> $DEST
curl $BASEURL/crds/eventing_v1alpha1_knativeeventing_crd.yaml  >> $DEST; echo "---" >> $DEST

sed -i.bak 's/namespace: default/namespace: knative-eventing/g' $DEST
rm $DEST.bak