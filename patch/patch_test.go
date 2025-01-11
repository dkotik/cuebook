package patch

import "testing"

func ensureInversible(
	source []byte,
	patch Patch,
) func(*testing.T) {
	return func(t *testing.T) {
		intermediate, err := patch.ApplyToCueSource(source)
		if err != nil {
			t.Log("original:", string(source))
			t.Log("intermediate:", string(intermediate))
			t.Fatal("unable to apply patch:", err)
		}
		reversed, err := patch.Invert().ApplyToCueSource(intermediate)
		if err != nil {
			t.Fatal("unable to apply reverse:", err)
		}
		if string(source) != string(reversed) {
			t.Log("original:", string(source))
			t.Log("intermediate:", string(intermediate))
			t.Log("reversed:", string(reversed))
			t.Fatal("reversed patch does not match original")
		}
	}
}
