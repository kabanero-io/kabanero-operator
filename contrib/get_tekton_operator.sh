DEST=tekton.yaml

BASEURL=https://raw.githubusercontent.com/openshift/tektoncd-pipeline-operator/master/deploy
$(dirname $0)/get_operator_config.sh $BASEURL $DEST

curl $BASEURL/crds/openshift-pipelines-operator-tekton_v1alpha1_install_crd.yaml -o openshift-pipelines-operator-tekton_v1alpha1_install_crd.yaml
cat openshift-pipelines-operator-tekton_v1alpha1_install_crd.yaml >> $DEST; echo "---" >> $DEST
rm openshift-pipelines-operator-tekton_v1alpha1_install_crd.yaml

sed -i.bak 's/namespace: openshift-pipelines-operator/namespace: kabanero/g' $DEST
rm $DEST.bak