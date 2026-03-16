package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

func TestGetKnownEntities_LoadsBootstrapJSON(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap_entities.json")

	content := `{
	  "0xABCDEF1234567890": {
	    "address": "",
	    "name": "Test Mixer",
	    "category": "MIXER",
	    "severity": "HIGH",
	    "confidence": "HIGH",
	    "source": "bootstrap_entities",
	    "trusted": false,
	    "notes": "test label"
	  }
	}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write bootstrap file: %v", err)
	}

	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", path)

	got := GetKnownEntities()
	if len(got) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(got))
	}

	label, ok := got["0xabcdef1234567890"]
	if !ok {
		t.Fatalf("expected normalized lowercase key to exist")
	}

	if label.Name != "Test Mixer" {
		t.Fatalf("expected label name %q, got %q", "Test Mixer", label.Name)
	}

	if label.Address != "0xabcdef1234567890" {
		t.Fatalf("expected normalized address to be populated from key, got %q", label.Address)
	}

	if label.Category != model.LabelCategoryMixer {
		t.Fatalf("expected category %q, got %q", model.LabelCategoryMixer, label.Category)
	}
}

func TestLookupEntityLabel_NormalizesAddress(t *testing.T) {
	resetKnownEntitiesCacheForTest(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap_entities.json")

	content := `{
	  "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b": {
	    "address": "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
	    "name": "Tornado Cash Router",
	    "category": "MIXER",
	    "severity": "HIGH",
	    "confidence": "HIGH",
	    "source": "bootstrap_entities",
	    "trusted": false,
	    "notes": "Known mixer-related routing contract"
	  }
	}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write bootstrap file: %v", err)
	}

	setEnvForTest(t, "BOOTSTRAP_LABELS_PATH", path)

	label, ok := LookupEntityLabel("0xD90E2F925DA726B50C4ED8D0FB90AD053324F31B")
	if !ok {
		t.Fatalf("expected lookup to succeed with mixed-case address")
	}

	if label.Name != "Tornado Cash Router" {
		t.Fatalf("expected Tornado Cash Router, got %q", label.Name)
	}
}
