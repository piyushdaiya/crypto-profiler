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
