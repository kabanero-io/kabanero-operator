package transforms

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Mount a secret by name
func MountSecret(secretName string, mountPoint string) func(u *unstructured.Unstructured) error {
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

			// Now get the volumeMounts for this container
			volumeMounts, ok, err := unstructured.NestedSlice(container, "volumeMounts")

			// NestedSlice will return err if the type is not a slice.  This can happen when
			// volumeMounts is defined but has no volume mounts listed (it's nil).  NestedSlice
			// will return !ok when the volumeMounts element does not exist at all.
			var newVolumeMounts []interface{}
			if (err == nil) && (ok) {
				// Look and see if this volume mount exists already
				for _, volumeMountRaw := range volumeMounts {
					volumeMount, ok := volumeMountRaw.(map[string]interface{})
					if !ok {
						return fmt.Errorf("Could not assert map type for volume mount: %v", volumeMountRaw)
					}

					// Copy all the volume mounts to the new list, skipping the desired name if it exists.
					if volumeMount["name"] != secretName {
						newVolumeMounts = append(newVolumeMounts, volumeMount)
					}
				}
			}

			// Now add the one we wanted
			newVolumeMount := make(map[string]interface{})
			newVolumeMount["name"] = secretName
			newVolumeMount["mountPath"] = mountPoint
			newVolumeMount["readOnly"] = true
			newVolumeMounts = append(newVolumeMounts, newVolumeMount)

			err = unstructured.SetNestedSlice(container, newVolumeMounts, "volumeMounts")
			if err != nil {
				return fmt.Errorf("Unable to set volumeMounts into unstructured: %v", err)
			}

			newContainers = append(newContainers, container)
		}

		err = unstructured.SetNestedSlice(u.Object, newContainers, "spec", "template", "spec", "containers")
		if err != nil {
			return fmt.Errorf("Unable to set containers into unstructured: %v", err)
		}

		// Now go change the volumes.  We can be confident that the path up to "volumes" exists
		// because we were able to modify the containers element in the previous step.  We'll
		// assume that if NestedSlice returns err or !ok that it's because "volumes" is empty.
		volumes, ok, err := unstructured.NestedSlice(u.Object, "spec", "template", "spec", "volumes")
		var newVolumes []interface{}
		if (err == nil) && (ok) {
			for _, volumeRaw := range volumes {
				volume, ok := volumeRaw.(map[string]interface{})
				if !ok {
					return fmt.Errorf("Could not assert map type for volume: %v", volumeRaw)
				}

				// Copy all the volumes to the new list, skipping the desired name if it exists.
				if volume["name"] != secretName {
					newVolumes = append(newVolumes, volume)
				}
			}
		}

		// Now add the one we wanted
		newVolume := make(map[string]interface{})
		newVolume["name"] = secretName
		newVolume["secret"] = map[string]interface{}{"secretName": secretName}
		newVolumes = append(newVolumes, newVolume)

		err = unstructured.SetNestedSlice(u.Object, newVolumes, "spec", "template", "spec", "volumes")
		if err != nil {
			return fmt.Errorf("Unable to set volumes into unstructured: %v", err)
		}
		
		return nil
	}
}
