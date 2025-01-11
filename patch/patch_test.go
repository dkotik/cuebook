package patch

import "testing"

func ensureInversible(
	source []byte,
	patch Patch,
) func(*testing.T) {
	return func(t *testing.T) {
		result, err := patch.ApplyToCueSource(source)
		if err != nil {
			t.Fatal("unable to apply patch:", err)
		}
		reversed, err := patch.Invert().ApplyToCueSource(result)
		if err != nil {
			t.Fatal("unable to apply reverse:", err)
		}
		if string(source) != string(reversed) {
			t.Log("original:", string(source))
			t.Log("reversed:", string(reversed))
			t.Fatal("reversed patch does not match original")
		}
	}
}
