DEST=tekton-operator.yaml
BASEURL=https://raw.githubusercontent.com/openshift/tektoncd-pipeline-operator/master/deploy

curl $BASEURL/operator.yaml -o $DEST; echo "---" >> $DEST
curl $BASEURL/role.yaml >> $DEST; echo "---" >> $DEST
curl $BASEURL/role_binding.yaml >> $DEST; echo "---" >> $DEST
curl $BASEURL/service_account.yaml >> $DEST; echo "---" >> $DEST
curl $BASEURL/crds/operator_v1alpha1_config_crd.yaml >> $DEST; echo "---" >> $DEST