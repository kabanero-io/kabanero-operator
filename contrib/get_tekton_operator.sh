DEST=tekton.yaml

BASEURL=https://raw.githubusercontent.com/openshift/tektoncd-pipeline-operator/master/deploy
$(dirname $0)/get_operator_config.sh $BASEURL $DEST

# The tekton CRD targets namespace openshift-pipelines. The instance will not fully start without it.
cat << EOF >> $DEST
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-pipelines
---
EOF

curl -f $BASEURL/crds/operator_v1alpha1_config_crd.yaml -o operator_v1alpha1_config_crd.yaml
cat operator_v1alpha1_config_crd.yaml >> $DEST; echo "---" >> $DEST
rm operator_v1alpha1_config_crd.yaml

curl -f $BASEURL/crds/operator_v1alpha1_config_cr.yaml -o operator_v1alpha1_config_cr.yaml
cat operator_v1alpha1_config_cr.yaml >> $DEST; echo "---" >> $DEST
rm operator_v1alpha1_config_cr.yaml

sed -i.bak 's/namespace: openshift-operators/namespace: kabanero/g' $DEST
rm $DEST.bak

sed -i.bak 's/name: name\.must\.be-cluster/name: cluster/g' $DEST
rm $DEST.bak
