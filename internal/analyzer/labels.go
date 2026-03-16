package analyzer

import "github.com/piyushdaiya/crypto-profiler/internal/model"

var knownEntities = map[string]model.EntityLabel{
	"0xd90e2f925da726b50c4ed8d0fb90ad053324f31b": {
		Address:    "0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
		Name:       "Tornado Cash Router",
		Category:   model.LabelCategoryMixer,
		Severity:   model.LabelSeverityHigh,
		Confidence: model.LabelConfidenceHigh,
		Source:     "static_bootstrap_labels",
		Trusted:    false,
		Notes:      "Known mixer-related routing contract",
	},
}
