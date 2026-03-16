package analyzer

import (
	"encoding/json"
	"log"
	"os"
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
		log.Printf("[labels] loaded %d bootstrap entity labels", len(knownEntities))
	})
	return knownEntities
}

func loadKnownEntities() map[string]model.EntityLabel {
	path := os.Getenv("BOOTSTRAP_LABELS_PATH")
	if path == "" {
		path = "/root/data/labels/bootstrap_entities.json"
	}

	log.Printf("[labels] loading bootstrap labels from %s", path)

	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[labels] failed to read bootstrap labels: %v", err)
		return map[string]model.EntityLabel{}
	}

	var entities map[string]model.EntityLabel
	if err := json.Unmarshal(raw, &entities); err != nil {
		log.Printf("[labels] failed to unmarshal bootstrap labels: %v", err)
		return map[string]model.EntityLabel{}
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

func LookupEntityLabel(address string) (model.EntityLabel, bool) {
	addr := strings.ToLower(strings.TrimSpace(address))
	label, ok := GetKnownEntities()[addr]
	return label, ok
}
