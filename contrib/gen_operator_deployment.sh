# This script generates metadata for fully deploying the Kabanero operator along with it's requirements.
# Generated metadata name: kabanero-operator.yaml.
# Generated metadata location: ../deply/releases/<version>.
# Where <version> is provided by the user, and it is the Kabanero operator version for which the 
# metadata is being created.
#--------------------------------------------------------------------------------------------------------

# Iterates over a single resource metadata yaml and appends the kabanero namspace if needed.
# Inputs:
# $1: Temporary file containing a single resource yaml.
# $2: A boolean value stating whether or not the function should add yaml resource delimiters (---).
# NOTES:
#  MAC users. Replace: sed -i '/^[ ]\{0\}metadata:/a \\ \ namespace: kabanero' $1
#             sed -i '' e '/^[ ]\{0\}metadata:/a \
#             \ \ namespace: kabanero' $1
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
                  sed -i '/^[ ]\{0\}metadata:/a \\ \ namespace: kabanero' $1
               fi

               foundMetadata=false
               rm rootMetadataEntry.tmp
            fi
         done < $1

         if [ -f rootMetadataEntry.tmp ]; then
            rm rootMetadataEntry.tmp
         fi
      else
         sed -i '/^[ ]\{0\}metadata:/a \\ \ namespace: kabanero' $1
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

# Ask for user input.
echo Please enter the kabanero operator version:
read version

# Basic input validation. Check for empty strings input or no input.
if [[ -z ${version// } ]]; then
   printf '%s\n' "Invalid input. Please provide a valid version."
   exit 1
fi

# Start processing the request.
DEST_DIR=../deploy/releases/$version
mkdir -p $DEST_DIR
DEST_FILE=$DEST_DIR/kabanero-operator.yaml
DEST_FILE_TMP=$DEST_DIR/kabanero-operator.yaml.tmp
DEST_FILE_BKUP=$DEST_DIR/kabanero-operator.yaml.bkup
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
cat ../deploy/dependencies.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP
cat ../deploy/crds/kabanero_v1alpha1_kabanero_crd.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP
cat ../deploy/service_account.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP
cat ../deploy/operator.yaml >> $DEST_FILE_TMP; echo "---" >> $DEST_FILE_TMP; echo
cat ../deploy/role.yaml >> $DEST_FILE_TMP;  echo "---" >> $DEST_FILE_TMP;
cat ../deploy/role_binding.yaml >> $DEST_FILE_TMP

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

# Cleanup.
if [ -f $DEST_FILE_BKUP ]; then
   rm $DEST_FILE_BKUP
fi

rm $DEST_FILE_TMP

# Issue completion message.
echo 'Completed execution. File location used: ' $DEST_FILE
