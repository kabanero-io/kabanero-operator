package utils

import (
	"fmt"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
)

// Validates that the stack policy configured in the kabanero CR instance yaml is one of the allowed values.
func ValidateGovernanceStackPolicy(kab *kabanerov1alpha2.Kabanero) (bool, string, error) {
	if len(kab.Spec.GovernacePolicy.StackPolicy) != 0 &&
		!(kab.Spec.GovernacePolicy.StackPolicy == kabanerov1alpha2.StackPolicyActiveDigest ||
			kab.Spec.GovernacePolicy.StackPolicy == kabanerov1alpha2.StackPolicyIgnoreDigest ||
			kab.Spec.GovernacePolicy.StackPolicy == kabanerov1alpha2.StackPolicyNone ||
			kab.Spec.GovernacePolicy.StackPolicy == kabanerov1alpha2.StackPolicyStrictDigest) {
		reason := fmt.Sprintf("The value %v associated with kabanero CR entry spec.governancePolicy.stackPolicy is not valid. The following are allowed values: %v, %v, %v, %v",
			kab.Spec.GovernacePolicy.StackPolicy, kabanerov1alpha2.StackPolicyStrictDigest,
			kabanerov1alpha2.StackPolicyActiveDigest, kabanerov1alpha2.StackPolicyIgnoreDigest,
			kabanerov1alpha2.StackPolicyNone)
		return false, reason, nil
	}

	return true, "", nil
}
