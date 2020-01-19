#!/bin/bash

set -Eeox pipefail

ssotemplate=$(oc get templates -n openshift -o=custom-columns=NAME:.metadata.name --ignore-not-found | grep sso73-x509-postgresql-persistent)
keycloakapp=$(oc get deploymentConfig -n kabanero --output=jsonpath='{range .items[0]}{"\n"}{@.metadata.name}{end}' --ignore-not-found)


if [[ ${ssotemplate} ]]; then
   if [[ ${keycloakapp} ]]; then
      echo "Keycloak application" ${keycloakapp} "is already installed. Exiting ..."
      exit 1
   else
      echo "Installing Keycloak ..."
      echo "Copying the template ${ssotemplate} to json file ..."
      oc get template ${ssotemplate} -n openshift -o json > ${ssotemplate}.json
      echo "The template json file is: " ${ssotemplate}.json
      if grep -q "9.5" ${ssotemplate}.json; then
         echo "postgresql version 9.5 is found"
         if ! grep -q "9.6" ${ssotemplate}.json; then
            echo "Modifying the josn file with postgresql version 9.6" 
            sed -i.bak 's/"9.5"/"9.6"/g' ${ssotemplate}.json
         else
            echo "postgresql version 9.6 is found, no modification on postgresql version is required. Proceeding to next step..."
         fi
      else
         echo "postgresql version 9.5 is not found"   
         if grep -q "9.6" ${ssotemplate}.json; then
            echo "postgresql version 9.6 is found, no modification on postgresql version is required. Proceeding to next step..."
         else
            echo "postgresql version 9.6 is not found. The template cannot be processed this time, exiting ... "
            exit 1
         fi
      fi
      if grep -q "/var/lib/pgsql/data" ${ssotemplate}.json; then
         echo "postgresql mount path /var/lib/pgsql/data is found"
         if ! grep -q "/var/lib/pgsql/data:z" ${ssotemplate}.json; then
            echo "Modifying the json file with postgresql mount path fix /var/lib/pgsql/data:z to resolve known issue"
            sed -i.bak 's/\/var\/lib\/pgsql\/data/\/var\/lib\/pgsql\/data:z/g' ${ssotemplate}.json
         else
            echo "postgresql mount path with fix /var/lib/pgsql/data:z is found, no modification is required. Proceeding to next step..."
         fi
      else
         echo "postgresql mount path /var/lib/pgsql/data is not found"
         if grep -q "/var/lib/pgsql/data:z" ${ssotemplate}.json; then
            echo "postgresql mount path with fix /var/lib/pgsql/data:z is found, no modification on postgresql version is required. Proceeding to next step..."   
         else
            echo "postgresql mount path with fix /var/lib/pgsql/data:z is not found. The template cannot be processed at this time, exiting ..."
            exit 1
         fi
      fi 
      echo "Processing the template json to create objects - deploymentconfig, services, route, and pvc ..."
      oc process -f ${ssotemplate}.json  | oc create -n kabanero -f - 
      echo "Removing the template json file and its backup file:" ${ssotemplate}.json, ${ssotemplate}.json.bak 
      rm -f ${ssotemplate}.json ${ssotemplate}.json.bak
      echo ""
   fi
else   
   echo "Red Hat sso template ${ssotemplate} is not found"
   exit 1
fi
