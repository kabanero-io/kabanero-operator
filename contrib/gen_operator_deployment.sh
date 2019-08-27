#!/bin/bash
#--------------------------------------------------------------------------------------------------------
# This script generates metadata for fully deploying the Kabanero operator along with it's requirements.
#
# Generated File: deploy/kabanero-operators.yaml.
#
# Usage: 
# ./gen_operator_deployment.sh [IMAGE]
#
# Where [IMAGE] is the complete image name, and it is optional. If [IMAGE] is not specified, the 
# kabanero operator deployment image is not updated. 
#
# Example: 
# ./gen_operator_deployment.sh kabanero/kabanero-operator:x.y.z
#--------------------------------------------------------------------------------------------------------

# Find we are running on MAC.
MAC_EXEC=false
macos=Darwin
if [ "$(uname)" == "Darwin" ]; then
   MAC_EXEC=true
fi

# Iterates over a single resource metadata yaml and appends the kabanero namspace if needed.
# Inputs:
# $1: Temporary file containing a single resource yaml.
# $2: A boolean value stating whether or not the function should add yaml resource delimiters (---).
add_kabanero_namespace () {
   foundMetadata=false
   if [ -s $1 ]; then
      # Avoid updating Cluster level resources
       if grep -q -e '\s\{0\}kind: CustomResourceDefinition' -e '\s\{0\}kind: [A-Z]*Cluster[a-zA-Z]*' $1; then
          cat $1 >> $DEST_FILE;
          if [ $2 = true ]; then
             echo '---' >> $DEST_FILE
          fi

          rm $1
          return 0;
       fi

      # The metadata contains a kabanero namespace entry.
      if grep -q '\s\{2\}namespace: kabanero' $1; then

         # Isolate the root 'metadata:' section in the yaml. See if a namespace entry found
         # is associated with it. If not add it.
         IFS=''
         while read -r line
         do
            rootMetadataEntryRegx='^[ ]{0}metadata:'
            if [[ $line =~ $rootMetadataEntryRegx ]]; then
               foundMetadata=true
               echo $line >> rootMetadataEntry.tmp
               continue
            fi

            if [[ "$foundMetadata" == true ]]; then
               echo $line >> rootMetadataEntry.tmp
            fi

            # If there are no other root entries, there must be the case that the metadata
            # root section has the namespace entry already. Therefore, Nothing to do.
            # If there are other root entries, check if the namespace entry found is in the 
            # metadata root section. If not, add it; otherwise, there is nothing to do.
            anyRootEntryRegx='^[ ]{0}[a-zA-Z]+'
            if [[ $foundMetadata == true && $line =~ $anyRootEntryRegx ]]; then
               if [[ "$(grep -q '\s\{2\}namespace:' rootMetadataEntry.tmp)" -eq 0 ]]; then
                  if [[ $MAC_EXEC == true ]]; then
                     sed -i '' -e '/^[ ]\{0\}metadata:/a \
                         \ \ namespace: kabanero' $1
                  else
                     sed -i '/^[ ]\{0\}metadata:/a \\ \ namespace: kabanero' $1
                  fi
               fi

               foundMetadata=false
               rm rootMetadataEntry.tmp
            fi
         done < $1

         if [ -f rootMetadataEntry.tmp ]; then
            rm rootMetadataEntry.tmp
         fi
      else
         if [[ $MAC_EXEC == true ]]; then
            sed -i '' -e '/^[ ]\{0\}metadata:/a \
               \ \ namespace: kabanero' $1
         else
            sed -i '/^[ ]\{0\}metadata:/a \\ \ namespace: kabanero' $1
         fi
      fi

      cat $1 >> $DEST_FILE;
      if [ $2 = true ]; then
         echo '---' >> $DEST_FILE
      fi

      rm $1
   else
      if [ $2 = true ]; then
         echo '---' >> $DEST_FILE
      fi
   fi
}

#------
# Main.
#------
BASE_DIR=$(dirname $0)
DEST_DIR=$BASE_DIR/../deploy
DEST_FILE=$DEST_DIR/kabanero-operators.yaml
DEST_FILE_TMP=$DEST_DIR/kabanero-operators.yaml.tmp
DEST_FILE_BKUP=$DEST_DIR/kabanero-operators.yaml.bkup

# Remove the existing destination file if it exists
if [ -f $DEST_FILE ]; then
   cp $DEST_FILE $DEST_FILE_BKUP
   rm $DEST_FILE
fi

# Add yaml content to allow for the creation of the kabanero namespace.
cat << EOF >> $DEST_FILE
apiVersion: v1
kind: Namespace
metadata:
  name: kabanero
---
EOF

# Add all needed yaml files.
cat $DEST_DIR/dependencies.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP
cat $DEST_DIR/crds/kabanero_kabanero_crd.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP
cat $DEST_DIR/crds/kabanero_collection_crd.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP
cat $DEST_DIR/service_account.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP
cat $DEST_DIR/operator.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP; echo
cat $DEST_DIR/role.yaml >> $DEST_FILE_TMP;  echo "---" >> $DEST_FILE_TMP;
cat $DEST_DIR/role_binding.yaml >> $DEST_FILE_TMP

# Add the kabanero namespace to the resources in DEST_FILE_TMP. 
# Process multiple entries ending in '---'.
IFS=''
addseparator=true
while read -r line; do
  if [[ $line == "---" ]]; then
     add_kabanero_namespace resource.yaml.tmp $addseparator
  else
     echo $line >> resource.yaml.tmp
  fi
done < "$DEST_FILE_TMP"

# Process the last entry if the file did not end in '---'.
addseparator=false
if [ -s resource.yaml.tmp ]; then
   IFS=''
   while read linex; do
        add_kabanero_namespace resource.yaml.tmp $addseparator
   done < resource.yaml.tmp
fi

# Update the operator deployment image entry if one was supplied.
if [ ! -z "$1" ]; then
   if [[ $MAC_EXEC == true ]]; then
      sed -i '' -e "s!image: kabanero-operator:latest!image: $1!g" $DEST_FILE
   else
      sed -i "s!image: kabanero-operator:latest!image: $1!g" $DEST_FILE
   fi
fi

# Cleanup.
if [ -f $DEST_FILE_BKUP ]; then
   rm $DEST_FILE_BKUP
fi

rm $DEST_FILE_TMP

# Issue completion message.
echo 'Completed execution. Destination file: ' $DEST_FILE
