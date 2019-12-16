package transforms

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/jsonpath"
)

// ReplaceEnvVariable produces a transformation capable of replacing an environment variable value
func ReplaceEnvVariable(variableName string, variableValue interface{}) func(u *unstructured.Unstructured) error {
	return func(u *unstructured.Unstructured) error {
		p := jsonpath.New("replace-env-var")

		err := p.Parse("{.spec.template.spec.containers[*].env[?(@.name=='" + variableName + "')]}")
		if err != nil {
			return err
		}

		// AllowMissingKeys means that if the query does not match the input document, no
		// error is generated
		p.AllowMissingKeys(true)

		allResults, err := p.FindResults(u.Object)
		if err != nil {
			return err
		} else {
			for _, resources := range allResults {
				for _, localResults := range resources {
					env_var := localResults.Interface().(map[string]interface{})
					env_var["value"] = variableValue
					if _, exists := env_var["valueFrom"]; exists {
						delete(env_var, "valueFrom")
					}
				}
			}
		}

		return nil
	}
}

// AddEnvVariable produces a transformation capable of adding an environment variable value
func AddEnvVariable(variableName string, variableValue interface{}) func(u *unstructured.Unstructured) error {
	return func(u *unstructured.Unstructured) error {
		// Only apply this to deployments
		if u.GetKind() != "Deployment" && u.GetAPIVersion() != "apps/v1" {
			return nil
		}
		
		// Since unstructured get nested does not support slice notation, we need to first retrieve
		// the containers array, and iterate over it.
		containers, ok, err := unstructured.NestedSlice(u.Object, "spec", "template", "spec", "containers")
		if err != nil {
			return fmt.Errorf("Unable to retrieve containers from unstructured: %v", err)
		}

		if !ok {
			return fmt.Errorf("No containers entry in deployment spec: %v", u)
		}

		var newContainers []interface{}
		for _, containerRaw := range containers {
			container, ok := containerRaw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Could not assert map type for containers: %v", containerRaw)
			}

			// Now get the env vars for this container.  NestedSlice will return err if the
			// element is not a slice (ie if it's empty) or !ok if it does not exist.  We
			// should handle both of these cases.
			var newEnvVars []interface{}
			envVars, ok, err := unstructured.NestedSlice(container, "env")
			if (err == nil) && (ok) {
				// Look and see if this env var exists already
				for _, envVarRaw := range envVars {
					envVar, ok := envVarRaw.(map[string]interface{})
					if !ok {
						return fmt.Errorf("Could not assert map type for env var: %v", envVarRaw)
					}

					// Copy all the env vars to the new list, skipping the desired name if it exists.
					if envVar["name"] != variableName {
						newEnvVars = append(newEnvVars, envVar)
					}
				}
			}
			
			// Now add the one we wanted
			newVar := make(map[string]interface{})
			newVar["name"] = variableName
			newVar["value"] = variableValue

			newEnvVars = append(newEnvVars, newVar)

			err = unstructured.SetNestedSlice(container, newEnvVars, "env")
			if err != nil {
				return fmt.Errorf("Unable to set env vars into unstructured: %v", err)
			}

			newContainers = append(newContainers, container)
		}

		err = unstructured.SetNestedSlice(u.Object, newContainers, "spec", "template", "spec", "containers")
		if err != nil {
			return fmt.Errorf("Unable to set containers into unstructured: %v", err)
		}

		return nil
	}
}
