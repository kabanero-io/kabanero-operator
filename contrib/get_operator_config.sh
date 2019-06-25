set -x

BASEURL=$1
DEST=$2

if [ -z ${BASEURL} ]; then 
    echo "Usage: get_operator_config.sh [base url] [output filename]"
    exit -1
fi

if [ -z ${DEST} ]; then 
    echo "Usage: get_operator_config.sh [base url] [output filename]"
    exit -1
fi

curl $BASEURL/operator.yaml -o operator.yaml
curl $BASEURL/role.yaml -o role.yaml
curl $BASEURL/role_binding.yaml -o role_binding.yaml
curl $BASEURL/service_account.yaml -o service_account.yaml

rm $DEST
for f in operator.yaml role.yaml role_binding.yaml service_account.yaml; do
  cat $f >> $DEST; echo "---" >> $DEST
done

rm operator.yaml role.yaml role_binding.yaml service_account.yaml
