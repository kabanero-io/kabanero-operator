#!/bin/bash

ssotemplate=$(oc get templates -n openshift -o=custom-columns=NAME:.metadata.name --ignore-not-found | grep sso73-x509-postgresql-persistent)

if [[ ${ssotemplate} ]]; then
   echo "Installing Keycloak ..."
   echo "Copying the template ${ssotemplate} to json file ..."
   oc get template ${ssotemplate} -n openshift -o json > ${ssotemplate}.json
   echo "The template json file is: " ${ssotemplate}.json
   echo "Modifying the josn file with postgresql version 9.6 and mountpath to resolve known issues on postgresql ..."
   sed -i 's/"9.5"/"9.6"/g' ${ssotemplate}.json
   sed -i 's/\/var\/lib\/pgsql\/data/\/var\/lib\/pgsql\/data:z/g' ${ssotemplate}.json
   echo "Processing the template json to create objects - deploymentconfig, services, route, and pvc ..."
   oc process -f ${ssotemplate}.json  | oc create -n kabanero -f - 
   echo ""

   if [[ $? -ne 0 ]]; then
        echo "Red Hat sso template ${ssotemplate} is not found"
        exit 1
   fi

fi
