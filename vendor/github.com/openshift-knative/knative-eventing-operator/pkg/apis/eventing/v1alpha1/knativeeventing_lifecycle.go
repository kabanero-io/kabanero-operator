package v1alpha1

import (
	"github.com/knative/pkg/apis"
)

var conditions = apis.NewLivingConditionSet(
	DeploymentsAvailable,
	InstallSucceeded,
)

// GetConditions implements apis.ConditionsAccessor
func (s *KnativeEventingStatus) GetConditions() apis.Conditions {
	return s.Conditions
}

// SetConditions implements apis.ConditionsAccessor
func (s *KnativeEventingStatus) SetConditions(c apis.Conditions) {
	s.Conditions = c
}

func (is *KnativeEventingStatus) IsReady() bool {
	return conditions.Manage(is).IsHappy()
}

func (is *KnativeEventingStatus) IsInstalled() bool {
	return is.GetCondition(InstallSucceeded).IsTrue()
}

func (is *KnativeEventingStatus) IsAvailable() bool {
	return is.GetCondition(DeploymentsAvailable).IsTrue()
}

func (is *KnativeEventingStatus) IsDeploying() bool {
	return is.IsInstalled() && !is.IsAvailable()
}

func (is *KnativeEventingStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return conditions.Manage(is).GetCondition(t)
}

func (is *KnativeEventingStatus) InitializeConditions() {
	conditions.Manage(is).InitializeConditions()
}

func (is *KnativeEventingStatus) MarkInstallFailed(msg string) {
	conditions.Manage(is).MarkFalse(
		InstallSucceeded,
		"Error",
		"Install failed with message: %s", msg)
}

func (is *KnativeEventingStatus) MarkInstallSucceeded() {
	conditions.Manage(is).MarkTrue(InstallSucceeded)
}

func (is *KnativeEventingStatus) MarkDeploymentsAvailable() {
	conditions.Manage(is).MarkTrue(DeploymentsAvailable)
}

func (is *KnativeEventingStatus) MarkDeploymentsNotReady() {
	conditions.Manage(is).MarkFalse(
		DeploymentsAvailable,
		"NotReady",
		"Waiting on deployments")
}

func (is *KnativeEventingStatus) MarkIgnored(msg string) {
	conditions.Manage(is).MarkFalse(
		InstallSucceeded,
		"Ignored",
		"Install not attempted: %s", msg)
}
