#!/bin/bash
set -Eeuox pipefail

DEST=knative-contrib.yaml

RELEASE=v0.6.0
GITHUB_SOURCE=eventing-sources.yaml

#RELEASE=v0.7.1
#GITHUB_SOURCE=github.yaml

#v0.7.2+ is needed to fix
#https://github.com/knative/eventing-contrib/issues/481

BASEURL=https://github.com/knative/eventing-contrib/releases/download/${RELEASE}

curl -f -L $BASEURL/${GITHUB_SOURCE} -o ${GITHUB_SOURCE}
cat ${GITHUB_SOURCE} >> $DEST; echo "---" >> $DEST
rm ${GITHUB_SOURCE}
