DEST=knative-serving.yaml

BASEURL=https://raw.githubusercontent.com/knative/serving-operator/master/config/
$(dirname $0)/get_operator_config.sh $BASEURL $DEST

curl $BASEURL/crds/serving_v1alpha1_knativeserving_crd.yaml -o serving_v1alpha1_knativeserving_crd.yaml
cat serving_v1alpha1_knativeserving_crd.yaml >> $DEST; echo "---" >> $DEST
rm serving_v1alpha1_knativeserving_crd.yaml

sed -i.bak 's/namespace: default/namespace: kabanero/g' $DEST
rm $DEST.bak


# Use openshift-knative image until upstream image is available
sed -i.bak 's|image: github.com/knative/serving-operator/cmd/manager|image: quay.io/openshift-knative/knative-serving-operator:v0.7.0|g' $DEST
rm $DEST.bak