# This script generates metadata for fully deploying the Kabanero operator along with it's requirements.
# Generated metadata name: kabanero-operator.yaml.
# Generated metadata location: ../deply/releases/<version>.
# Where <version> is provided by the user, and it is the Kabanero operator version for which the metadata is being created.

# Ask for user input.
echo Please enter the kabanero operator version:
read version

# Basic input validation. Check for empty strings input or no input.
if [[ -z ${version// } ]]; then
   printf '%s\n' "Invalid input. Please provide a valid version."
   exit 1
fi

# Process the request.
DEST_DIR=../deploy/releases/$version
mkdir -p $DEST_DIR
DEST_FILE=$DEST_DIR/kabanero-operator.yaml
cat ../deploy/crds/kabanero_v1alpha1_kabanero_crd.yaml >> $DEST_FILE; echo >> $DEST_FILE; echo "---" >> $DEST_FILE; echo >> $DEST_FILE
cat ../deploy/service_account.yaml >> $DEST_FILE; echo >> $DEST_FILE; echo "---" >> $DEST_FILE; echo >> $DEST_FILE
cat ../deploy/operator.yaml >> $DEST_FILE; echo >> $DEST_FILE; echo "---" >> $DEST_FILE; echo >> $DEST_FILE
cat ../deploy/role.yaml >> $DEST_FILE; echo >> $DEST_FILE; echo "---" >> $DEST_FILE; echo >> $DEST_FILE
cat ../deploy/role_binding.yaml >> $DEST_FILE
