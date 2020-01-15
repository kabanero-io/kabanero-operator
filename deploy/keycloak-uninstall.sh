#!/bin/bash

keycloakapp=$(oc get deploymentConfig -n kabanero --output=jsonpath='{range .items[0]}{"\n"}{@.metadata.name}{end}' --ignore-not-found)


if [[ ${keycloakapp} ]]; then
    echo ""
    echo "The deployed keycloak application is: "${keycloakapp}
    echo "Deleting deploymentconfig ..."
    kubectl delete deploymentconfig ${keycloakapp}  -n kabanero 
    kubectl delete deploymentconfig ${keycloakapp}-postgresql  -n kabanero 
    echo "Deleting services ..."
    kubectl delete service ${keycloakapp} -n kabanero 
    kubectl delete service ${keycloakapp}-postgresql -n kabanero 
    kubectl delete service ${keycloakapp}-ping  -n kabanero 
    echo "Deleting route ..."
    kubectl delete route ${keycloakapp}  -n kabanero 
    echo "Deleting persistence volume claim ..."
    kubectl delete pvc ${keycloakapp}-postgresql-claim  -n kabanero 
    echo ""

    if [[ $? -ne 0 ]]; then
        echo "keycloak application ${keycloakapp} is not found"
        exit 1
    fi
fi

