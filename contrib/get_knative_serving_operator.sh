DEST=knative-serving.yaml

BASEURL=https://raw.githubusercontent.com/openshift-knative/knative-serving-operator/master/deploy
$(dirname $0)/get_operator_config.sh $BASEURL $DEST

curl $BASEURL/crds/serving_v1alpha1_knativeserving_crd.yaml -o serving_v1alpha1_knativeserving_crd.yaml
cat serving_v1alpha1_knativeserving_crd.yaml >> $DEST; echo "---" >> $DEST
rm serving_v1alpha1_knativeserving_crd.yaml

sed -i '' 's/namespace: default/namespace: kabanero/' $DEST
