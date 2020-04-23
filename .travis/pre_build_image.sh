#!/bin/bash

# Retrieve the current release from the makefile.
CURRENT_RELEASE=$(sed -n '/^CURRENT_RELEASE\( \)*=\( \)*/p' Makefile | sed 's/^.*=\( \)*//')
CSV_FILE=registry/manifests/kabanero-operator/$CURRENT_RELEASE/kabanero-operator.v$CURRENT_RELEASE.clusterserviceversion.yaml

# Build a CSV relatedImages.
GO111MODULE=on go build -o build/_output/contrib/bin/csventrygen github.com/kabanero-io/kabanero-operator/contrib/go
build/_output/contrib/bin/csventrygen true
sed -i -e 's/^/  /' contrib/go/csvRelatedImages.yaml

# Remove any existing spec.relatedImages entry from CSV. 
IFS=''
startdel=false
while read -r line; do
  riRegex='^[ ]{2}relatedImages:.*'
  if [[ $line =~ $riRegex ]]; then
     startdel=true
     continue
  fi

  nextFirstLineEntry='^[ ]{2}[a-zA-Z]+.*'
  if [[ $line =~ $nextFirstLineEntry ]]; then
    startdel=false
  fi

  if [[ "$startdel" == false ]];then
     echo $line >> csv.tmp
  fi
done < $CSV_FILE

# Add the newly created relatedImages entry into the CSV.
cat contrib/go/csvRelatedImages.yaml >> csv.tmp
cp csv.tmp $CSV_FILE

# Delete tmp file.
rm csv.tmp