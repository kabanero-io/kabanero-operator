package versioning

import (
	"testing"
)

// Verifies validitity of the internal static model
func TestStaticVersionData(t *testing.T) {
	v := Data

	if v.DefaultKabaneroRevision == "" {
		t.Fatal("No default version")
	}

	if len(v.KabaneroRevisions) < 1 {
		t.Fatal("No Kabanero versions are present")
	}

	for _, k := range v.KabaneroRevisions {
		if k.Version == "" {
			t.Fatal("Kabanero version has an empty version identifier")
		}

		//Verify the integrity of relationships from the kabanero revision table
		//into the version data for those components
		for sw, v := range k.RelatedVersions {
			var found bool
			for _, rev := range k.Document.RelatedSoftwareRevisions[sw] {
				if rev.Version == v {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("The Kabanero version `%v` points to the software %v version `%v`, but that reference cannot be resolved", k.Version, sw, v)
			}
		}
	}

	if len(v.RelatedSoftwareRevisions) < 1 {
		t.Fatal("No Kabanero versions are present")
	}

	for software_package, versions := range v.RelatedSoftwareRevisions {
		if len(versions) < 1 {
			t.Fatalf("Expected '%v' to have at least one software version", software_package)
		}
	}
}

func TestKabaneroRevisions(t *testing.T) {
	rev := Data.KabaneroRevision("0.6.0")
	if rev == nil {
		t.Fatal("Revision was nil")
	}
}
