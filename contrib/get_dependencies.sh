$(dirname $0)/get_knative_eventing_operator.sh
$(dirname $0)/get_knative_serving_operator.sh
$(dirname $0)/get_tekton_operator.sh

DEST=dependencies.yaml
cat knative-eventing.yaml >> $DEST; echo "---" >> $DEST
cat knative-serving.yaml >> $DEST; echo "---" >> $DEST
cat tekton.yaml >> $DEST; echo "---" >> $DEST

rm knative-eventing.yaml knative-serving.yaml tekton.yaml
