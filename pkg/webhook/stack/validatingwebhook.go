package stack

// The controller-runtime example webhook (v0.10) was used to build this
// webhook implementation.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/stack/utils"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/blang/semver"
)

// BuildValidatingWebhook builds the webhook for the manager to register
func BuildValidatingWebhook(mgr *manager.Manager) *admission.Webhook {
	return &admission.Webhook{Handler: &stackValidator{}}
}

// stackValidator validates Stacks
type stackValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// Implement admission.Handler so the controller can handle admission request.
// This no-op assignment ensures that the struct implements the interface.
var _ admission.Handler = &stackValidator{}

// stackValidator admits a stack if it passes validity checks
func (v *stackValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	stack := &kabanerov1alpha2.Stack{}

	err := v.decoder.Decode(req, stack)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	allowed, reason, err := v.validateStackFn(ctx, stack)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.ValidationResponse(allowed, reason)
}

func (v *stackValidator) validateStackFn(ctx context.Context, stack *kabanerov1alpha2.Stack) (bool, string, error) {

	reason := fmt.Sprintf("")
	err := fmt.Errorf(reason)

	if len(stack.Spec.Name) == 0 {
		reason = fmt.Sprintf("Stack Spec.Name is not set. stack: %v", stack)
		err = fmt.Errorf(reason)
		return false, reason, err
	}

	if len(stack.Spec.Versions) == 0 {
		reason = fmt.Sprintf("Stack %v Spec.Versions[] list is empty. stack: %v", stack.Spec.Name, stack)
		err = fmt.Errorf(reason)
		return false, reason, err
	}

	for _, version := range stack.Spec.Versions {

		if len(version.Version) == 0 {
			reason = fmt.Sprintf("Stack %v must set spec.Versions[].Version. stack: %v", stack.Spec.Name, stack)
			err = fmt.Errorf(reason)
			return false, reason, err
		}

		_, err := semver.Parse(version.Version)
		if err != nil {
			reason = fmt.Sprintf("Stack %v %v spec.Versions[].Version must be semver. %v. stack: %v", stack.Spec.Name, version.Version, err, stack)
			err = fmt.Errorf(reason)
			return false, reason, err
		}

		if (len(version.DesiredState) != 0) && !((strings.ToLower(version.DesiredState) == "active") || (strings.ToLower(version.DesiredState) == "inactive")) {
			reason = fmt.Sprintf("Stack %v %v Spec.Versions[].DesiredState may only be set to active or inactive. stack: %v", stack.Spec.Name, version.Version, stack)
			err = fmt.Errorf(reason)
			return false, reason, err
		}

		if len(version.Images) == 0 {
			reason = fmt.Sprintf("Stack %v %v must contain at least one entry for spec.Versions[].Images. stack: %v", stack.Spec.Name, version.Version, stack)
			err = fmt.Errorf(reason)
			return false, reason, err
		} else {
			for _, image := range version.Images {
				repository, err := utils.GetImageRepository(image.Image)
				if err != nil {
					reason = fmt.Sprintf("Could not parse Image %v associated with Stack %v %v: %v", image.Image, stack.Spec.Name, version.Version, err.Error())
					return false, reason, err
				}
				if repository != image.Image {
					reason = fmt.Sprintf("Image %v associated with Stack %v %v should not contain an image tag. Stack: %v", image.Image, stack.Spec.Name, version.Version, stack)
					err = fmt.Errorf(reason)
					return false, reason, err
				}
			}
		}

		for _, pipeline := range version.Pipelines {
			if len(pipeline.Https.Url) == 0 && pipeline.GitRelease == (kabanerov1alpha2.GitReleaseSpec{}) {
				reason = fmt.Sprintf("Stack %v %v does not contain a Spec.Versions[].Pipelines[].Https.Url or a populated Spec.Versions[].Pipelines[].GitRelease{}. One of them must be specified. If both are specified, Spec.Versions[].Pipelines[].GitRelease{} takes precedence. Stack: %v", stack.Spec.Name, version.Version, stack)
				err = fmt.Errorf(reason)
				return false, reason, err
			}
			
			if len(pipeline.Https.Url) != 0 {
				fileNameURL, err := url.Parse(pipeline.Https.Url)
				if err != nil {
					reason = fmt.Sprintf("Stack %v %v Spec.Versions[].Pipelines[].Https.Url failed to parse. stack: %v", stack.Spec.Name, version.Version, stack)
					return false, reason, err
				}
				
				switch {
					case strings.HasSuffix(fileNameURL.Path, ".tar.gz") || strings.HasSuffix(fileNameURL.Path, ".tgz"):
						if len(pipeline.Sha256) == 0 {
							reason = fmt.Sprintf("Stack %v %v Spec.Versions[].Pipelines[].Sha256 must be set for .tar.gz. stack: %v", stack.Spec.Name, version.Version, stack)
							err = fmt.Errorf(reason)
							return false, reason, err
						}
					case strings.HasSuffix(fileNameURL.Path, ".yaml") || strings.HasSuffix(fileNameURL.Path, ".yml"):
						break
					default:
						reason = fmt.Sprintf("Stack %v %v Spec.Versions[].Pipelines[].Https.Url must be a .tar.gz or .yaml. stack: %v", stack.Spec.Name, version.Version, stack)
						err = fmt.Errorf(reason)
						return false, reason, err
				}
			}

			if len(pipeline.GitRelease.AssetName) != 0 {
				switch {
					case strings.HasSuffix(pipeline.GitRelease.AssetName, ".tar.gz") || strings.HasSuffix(pipeline.GitRelease.AssetName, ".tgz"):
						if len(pipeline.Sha256) == 0 {
							reason = fmt.Sprintf("Stack %v %v Spec.Versions[].Pipelines[].Sha256 must be set for .tar.gz. stack: %v", stack.Spec.Name, version.Version, stack)
							err = fmt.Errorf(reason)
							return false, reason, err
						}
					case strings.HasSuffix(pipeline.GitRelease.AssetName, ".yaml") || strings.HasSuffix(pipeline.GitRelease.AssetName, ".yml"):
						break
					default:
						reason = fmt.Sprintf("Stack %v %v Spec.Versions[].Pipelines[].GitRelease.AssetName must be a .tar.gz or .yaml. stack: %v", stack.Spec.Name, version.Version, stack)
						err = fmt.Errorf(reason)
						return false, reason, err
				}
			}
		}
	}

	return true, reason, nil
}

// InjectClient injects the client.
func (v *stackValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *stackValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
