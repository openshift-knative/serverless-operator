package e2e

import "testing"

func ensureNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
