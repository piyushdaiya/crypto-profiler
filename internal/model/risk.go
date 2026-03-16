package model

type LabelCategory string

const (
	LabelCategorySanctions LabelCategory = "SANCTIONS"
	LabelCategoryMixer     LabelCategory = "MIXER"
	LabelCategoryExploit   LabelCategory = "EXPLOIT"
	LabelCategoryScam      LabelCategory = "SCAM"
	LabelCategoryExchange  LabelCategory = "EXCHANGE"
	LabelCategoryProtocol  LabelCategory = "PROTOCOL"
	LabelCategoryTrusted   LabelCategory = "TRUSTED"
	LabelCategoryUnknown   LabelCategory = "UNKNOWN"
)

type LabelSeverity string

const (
	LabelSeverityLow      LabelSeverity = "LOW"
	LabelSeverityMedium   LabelSeverity = "MEDIUM"
	LabelSeverityHigh     LabelSeverity = "HIGH"
	LabelSeverityCritical LabelSeverity = "CRITICAL"
)

type LabelConfidence string

const (
	LabelConfidenceLow    LabelConfidence = "LOW"
	LabelConfidenceMedium LabelConfidence = "MEDIUM"
	LabelConfidenceHigh   LabelConfidence = "HIGH"
)

type EntityLabel struct {
	Address    string          `json:"address"`
	Name       string          `json:"name"`
	Category   LabelCategory   `json:"category"`
	Severity   LabelSeverity   `json:"severity"`
	Confidence LabelConfidence `json:"confidence"`
	Source     string          `json:"source"`
	Trusted    bool            `json:"trusted,omitempty"`
	Notes      string          `json:"notes,omitempty"`
}

// Backward-compatible: keep existing fields, add optional metadata.
type RiskReason struct {
	Code           string          `json:"code,omitempty"`
	Category       string          `json:"category"`
	Description    string          `json:"description"`
	Offset         float64         `json:"offset"`
	Source         string          `json:"source,omitempty"`
	RelatedEntity  string          `json:"related_entity,omitempty"`
	RelatedAddress string          `json:"related_address,omitempty"`
	Severity       LabelSeverity   `json:"severity,omitempty"`
	Confidence     LabelConfidence `json:"confidence,omitempty"`
	EvidenceCount  int             `json:"evidence_count,omitempty"`
}
