#!/bin/bash
set -Eeuox pipefail


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

curl -f $BASEURL/operator.yaml -o operator.yaml
curl -f $BASEURL/role.yaml -o role.yaml
curl -f $BASEURL/role_binding.yaml -o role_binding.yaml
curl -f $BASEURL/service_account.yaml -o service_account.yaml

rm -f $DEST
for f in operator.yaml role.yaml role_binding.yaml service_account.yaml; do
  cat $f >> $DEST; echo "---" >> $DEST
done

rm operator.yaml role.yaml role_binding.yaml service_account.yaml
