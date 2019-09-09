package kabaneroplatform

import (
	"fmt"
	"testing"
)

func TestAppsodyImageUri(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Default",
		},
		{
			name: "Override Appsody Image",
		},
		{
			name: "Override Appsody Tag",
		},
		{
			name: "Override Appsody Repository",
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s", tc.name), func(t *testing.T) {

		})
	}
}
