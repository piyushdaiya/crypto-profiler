package analyzer

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

var (
	knownEntities     map[string]model.EntityLabel
	knownEntitiesOnce sync.Once
)

func GetKnownEntities() map[string]model.EntityLabel {
	knownEntitiesOnce.Do(func() {
		knownEntities = loadKnownEntities()
	})
	return knownEntities
}

func loadKnownEntities() map[string]model.EntityLabel {
	paths := candidateBootstrapLabelPaths()

	var lastErr error
	for _, path := range paths {
		raw, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			lastErr = err
			continue
		}

		var entities map[string]model.EntityLabel
		if err := json.Unmarshal(raw, &entities); err != nil {
			lastErr = err
			continue
		}

		normalized := make(map[string]model.EntityLabel, len(entities))
		for address, label := range entities {
			addr := strings.ToLower(strings.TrimSpace(address))
			label.Address = strings.ToLower(strings.TrimSpace(label.Address))
			if label.Address == "" {
				label.Address = addr
			}
			normalized[addr] = label
		}

		return normalized
	}

	if lastErr != nil {
		log.Printf("[labels] bootstrap label load failed: %v", lastErr)
	}

	return map[string]model.EntityLabel{}
}

func candidateBootstrapLabelPaths() []string {
	if path := strings.TrimSpace(os.Getenv("BOOTSTRAP_LABELS_PATH")); path != "" {
		return []string{path}
	}

	return []string{
		"./data/labels/bootstrap_entities.json",
		"data/labels/bootstrap_entities.json",
		"/root/data/labels/bootstrap_entities.json",
	}
}

func LookupEntityLabel(address string) (model.EntityLabel, bool) {
	addr := strings.ToLower(strings.TrimSpace(address))
	label, ok := GetKnownEntities()[addr]
	return label, ok
}
