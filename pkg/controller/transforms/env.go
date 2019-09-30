package transforms

import (
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
