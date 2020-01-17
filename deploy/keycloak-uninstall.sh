#!/bin/bash

keycloakapp=$(oc get deploymentConfig -n kabanero --output=jsonpath='{range .items[0]}{"\n"}{@.metadata.name}{end}' --ignore-not-found)

if [[ ${keycloakapp} ]]; then
    echo "Uninstalling Keycloak ..."
    echo "The deployed Keycloak application is:"${keycloakapp}
    echo "Deleting deploymentconfig ..."
    oc delete deploymentconfig ${keycloakapp} -n kabanero 
    oc delete deploymentconfig ${keycloakapp}-postgresql -n kabanero 
    echo "Deleting services ..."
    oc delete service ${keycloakapp} -n kabanero 
    oc delete service ${keycloakapp}-postgresql -n kabanero 
    oc delete service ${keycloakapp}-ping -n kabanero 
    echo "Deleting route ..."
    oc delete route ${keycloakapp} -n kabanero 
    echo "Deleting persistence volume claim ..."
    oc delete pvc ${keycloakapp}-postgresql-claim -n kabanero 
    echo ""

    if [[ $? -ne 0 ]]; then
        echo "keycloak application ${keycloakapp} is not found"
        exit 1
    fi
fi

