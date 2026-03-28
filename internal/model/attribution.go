/*
 *
 *  *  Copyright (c) 2026 Piyush Daiya
 *  *  *
 *  *  * Permission is hereby granted, free of charge, to any person obtaining a copy
 *  *  * of this software and associated documentation files (the "Software"), to deal
 *  *  * in the Software without restriction, including without limitation the rights
 *  *  * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  *  * copies of the Software, and to permit persons to whom the Software is
 *  *  * furnished to do so, subject to the following conditions:
 *  *  *
 *  *  * The above copyright notice and this permission notice shall be included in all
 *  *  * copies or substantial portions of the Software.
 *  *  *
 *  *  * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  *  * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  *  * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  *  * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  *  * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  *  * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *  *  * SOFTWARE.
 *
 */

package model

type AttributionSourceTier string

const (
	AttributionSourceTierPrimaryStructured AttributionSourceTier = "PRIMARY_STRUCTURED"
	AttributionSourceTierSecondary         AttributionSourceTier = "SECONDARY_CORROBORATING"
	AttributionSourceTierLocalOverride     AttributionSourceTier = "LOCAL_OVERRIDE"
)

type AttributionSourceType string

const (
	AttributionSourceTypeGraphSense          AttributionSourceType = "GRAPHSENSE_STRUCTURED"
	AttributionSourceTypeMiningPool          AttributionSourceType = "BITCOIN_MINING_POOL"
	AttributionSourceTypeBootstrapDemo       AttributionSourceType = "BOOTSTRAP_LOCAL"
	AttributionSourceTypeWalletExplorerStyle AttributionSourceType = "WALLET_EXPLORER_STYLE"
	AttributionSourceTypeSecondaryFixture    AttributionSourceType = "SECONDARY_FIXTURE"
)

type AttributionRiskClass string

const (
	AttributionRiskClassSanctioned     AttributionRiskClass = "SANCTIONED"
	AttributionRiskClassIllicitService AttributionRiskClass = "ILLICIT_SERVICE"
	AttributionRiskClassExploit        AttributionRiskClass = "EXPLOIT"
	AttributionRiskClassScam           AttributionRiskClass = "SCAM"
	AttributionRiskClassExchange       AttributionRiskClass = "EXCHANGE"
	AttributionRiskClassTrustedService AttributionRiskClass = "TRUSTED_SERVICE"
	AttributionRiskClassMiningPool     AttributionRiskClass = "MINING_POOL"
	AttributionRiskClassTreasury       AttributionRiskClass = "TREASURY"
	AttributionRiskClassUnknown        AttributionRiskClass = "UNKNOWN"
)

type AttributionSourceMetadata struct {
	Name        string                `json:"name"`
	Tier        AttributionSourceTier `json:"tier"`
	Type        AttributionSourceType `json:"type"`
	Description string                `json:"description,omitempty"`
}

type AttributionRecord struct {
	Address    string                    `json:"address"`
	Network    string                    `json:"network,omitempty"`
	Label      string                    `json:"label"`
	Actor      string                    `json:"actor,omitempty"`
	Category   LabelCategory             `json:"category"`
	RiskClass  AttributionRiskClass      `json:"risk_class"`
	Confidence float64                   `json:"confidence"`
	Contextual bool                      `json:"contextual"`
	Escalating bool                      `json:"risk_escalating"`
	Source     AttributionSourceMetadata `json:"source"`
	Notes      string                    `json:"notes,omitempty"`
}

type ResolvedAttributionSource struct {
	Name        string                `json:"name"`
	Tier        AttributionSourceTier `json:"tier"`
	Type        AttributionSourceType `json:"type"`
	Label       string                `json:"label,omitempty"`
	Actor       string                `json:"actor,omitempty"`
	Category    LabelCategory         `json:"category,omitempty"`
	RiskClass   AttributionRiskClass  `json:"risk_class,omitempty"`
	Confidence  float64               `json:"confidence,omitempty"`
	Contextual  bool                  `json:"contextual,omitempty"`
	Escalating  bool                  `json:"risk_escalating,omitempty"`
	Description string                `json:"description,omitempty"`
}

type ResolvedAttribution struct {
	Address              string                      `json:"address"`
	Network              string                      `json:"network,omitempty"`
	Label                string                      `json:"label"`
	Actor                string                      `json:"actor,omitempty"`
	Category             LabelCategory               `json:"category"`
	RiskClass            AttributionRiskClass        `json:"risk_class"`
	BaseConfidence       float64                     `json:"base_confidence,omitempty"`
	Confidence           float64                     `json:"confidence"`
	Contextual           bool                        `json:"contextual"`
	Escalating           bool                        `json:"risk_escalating"`
	SourceName           string                      `json:"source_name"`
	SourceTier           AttributionSourceTier       `json:"source_tier"`
	SourceType           AttributionSourceType       `json:"source_type"`
	SecondaryOnly        bool                        `json:"secondary_only,omitempty"`
	SupportingSources    []ResolvedAttributionSource `json:"supporting_sources,omitempty"`
	CorroboratingSources []ResolvedAttributionSource `json:"corroborating_sources,omitempty"`
	ConflictingSources   []ResolvedAttributionSource `json:"conflicting_sources,omitempty"`
}
