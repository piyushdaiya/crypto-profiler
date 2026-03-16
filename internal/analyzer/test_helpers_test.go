package analyzer

import (
	"os"
	"sync"
	"testing"
)

func resetKnownEntitiesCacheForTest(t *testing.T) {
	t.Helper()

	knownEntities = nil
	knownEntitiesOnce = sync.Once{}
}

func setEnvForTest(t *testing.T, key, value string) {
	t.Helper()

	original, existed := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if existed {
			err = os.Setenv(key, original)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("failed to restore env %s: %v", key, err)
		}
	})
}
