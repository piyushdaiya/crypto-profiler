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

package attribution

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/piyushdaiya/crypto-profiler/internal/model"
)

type Resolver struct {
	recordsByAddress map[string][]model.AttributionRecord
}

var (
	defaultResolver     *Resolver
	defaultResolverOnce sync.Once
)

func DefaultResolver() *Resolver {
	defaultResolverOnce.Do(func() {
		defaultResolver = loadDefaultResolver()
	})
	return defaultResolver
}

func ResetDefaultResolverForTesting() {
	defaultResolver = nil
	defaultResolverOnce = sync.Once{}
}

func ResolveAddress(address string, network string) (*model.ResolvedAttribution, bool) {
	return DefaultResolver().Resolve(address, network)
}

func LookupEntityLabel(address string, network string) (model.EntityLabel, bool) {
	resolved, ok := ResolveAddress(address, network)
	if !ok || resolved == nil {
		return model.EntityLabel{}, false
	}

	name := resolved.Label
	if strings.TrimSpace(name) == "" {
		name = resolved.Actor
	}

	return model.EntityLabel{
		Address:    resolved.Address,
		Name:       name,
		Actor:      resolved.Actor,
		Category:   resolved.Category,
		RiskClass:  resolved.RiskClass,
		Severity:   severityForResolved(resolved),
		Confidence: confidenceBucket(resolved.Confidence),
		Source:     resolved.SourceName,
		SourceTier: resolved.SourceTier,
		SourceType: resolved.SourceType,
		Trusted:    resolved.Contextual,
		Contextual: resolved.Contextual,
		Escalating: resolved.Escalating,
	}, true
}

func (r *Resolver) Resolve(address string, network string) (*model.ResolvedAttribution, bool) {
	if r == nil {
		return nil, false
	}

	normalized := normalizeAddress(address)
	if normalized == "" {
		return nil, false
	}

	records := append([]model.AttributionRecord(nil), r.recordsByAddress[normalized]...)
	if len(records) == 0 {
		return nil, false
	}

	filtered := records[:0]
	for _, record := range records {
		if networkMatches(record.Network, network) {
			filtered = append(filtered, record)
		}
	}
	if len(filtered) == 0 {
		filtered = records
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		left := filtered[i]
		right := filtered[j]
		if tierRank(left.Source.Tier) != tierRank(right.Source.Tier) {
			return tierRank(left.Source.Tier) > tierRank(right.Source.Tier)
		}
		if left.Confidence != right.Confidence {
			return left.Confidence > right.Confidence
		}
		if left.Escalating != right.Escalating {
			return left.Escalating
		}
		if left.Contextual != right.Contextual {
			return left.Contextual
		}
		if left.Source.Name != right.Source.Name {
			return left.Source.Name < right.Source.Name
		}
		return left.Label < right.Label
	})

	primary := filtered[0]
	supporting := uniqueResolvedSources(filtered, func(record model.AttributionRecord) bool {
		return corroborates(primary, record) || sameSourceRecord(primary, record)
	})
	corroborating := uniqueResolvedSources(filtered, func(record model.AttributionRecord) bool {
		return !sameSourceRecord(primary, record) && corroborates(primary, record)
	})
	conflicting := uniqueResolvedSources(filtered, func(record model.AttributionRecord) bool {
		return conflicts(primary, record)
	})

	baseConfidence := normalizedBaseConfidence(primary)
	confidence := adjustedConfidence(baseConfidence, primary, corroborating, conflicting)

	return &model.ResolvedAttribution{
		Address:              primary.Address,
		Network:              firstNonEmpty(primary.Network, strings.ToUpper(strings.TrimSpace(network))),
		Label:                primary.Label,
		Actor:                primary.Actor,
		Category:             primary.Category,
		RiskClass:            primary.RiskClass,
		BaseConfidence:       baseConfidence,
		Confidence:           confidence,
		Contextual:           primary.Contextual,
		Escalating:           primary.Escalating,
		SourceName:           primary.Source.Name,
		SourceTier:           primary.Source.Tier,
		SourceType:           primary.Source.Type,
		SecondaryOnly:        primary.Source.Tier == model.AttributionSourceTierSecondary,
		SupportingSources:    supporting,
		CorroboratingSources: corroborating,
		ConflictingSources:   conflicting,
	}, true
}

func loadDefaultResolver() *Resolver {
	resolver := &Resolver{
		recordsByAddress: map[string][]model.AttributionRecord{},
	}

	for _, record := range loadTier1GraphSenseRecords() {
		resolver.addRecord(record)
	}
	for _, record := range loadTier1MiningPoolRecords() {
		resolver.addRecord(record)
	}
	for _, record := range loadBootstrapOverrideRecords() {
		resolver.addRecord(record)
	}
	for _, record := range loadWalletExplorerStyleRecords() {
		resolver.addRecord(record)
	}
	for _, record := range loadSecondaryFixtureRecords() {
		resolver.addRecord(record)
	}

	return resolver
}

func (r *Resolver) addRecord(record model.AttributionRecord) {
	record.Address = normalizeAddress(record.Address)
	record.Network = strings.ToUpper(strings.TrimSpace(record.Network))
	record.Label = strings.TrimSpace(record.Label)
	record.Actor = strings.TrimSpace(record.Actor)
	if record.Address == "" {
		return
	}
	if record.Network == "" {
		record.Network = inferNetwork(record.Address)
	}
	if record.RiskClass == "" {
		record.RiskClass = inferRiskClass(record.Category, record.Contextual)
	}
	r.recordsByAddress[record.Address] = append(r.recordsByAddress[record.Address], record)
}

func loadTier1GraphSenseRecords() []model.AttributionRecord {
	paths := candidatePaths("GRAPHSENSE_LABELS_PATH", []string{
		"./data/labels/tier1_graphsense_entities.json",
		"data/labels/tier1_graphsense_entities.json",
	})
	return loadStructuredRecords(paths, model.AttributionSourceMetadata{
		Name:        "graphsense_structured_labels",
		Tier:        model.AttributionSourceTierPrimaryStructured,
		Type:        model.AttributionSourceTypeGraphSense,
		Description: "GraphSense-style structured actor labels used as Tier 1 attribution input.",
	})
}

func loadTier1MiningPoolRecords() []model.AttributionRecord {
	paths := candidatePaths("BITCOIN_MINING_POOLS_PATH", []string{
		"./data/labels/tier1_bitcoin_mining_pools.json",
		"data/labels/tier1_bitcoin_mining_pools.json",
	})
	return loadStructuredRecords(paths, model.AttributionSourceMetadata{
		Name:        "bitcoin_mining_pool_context",
		Tier:        model.AttributionSourceTierPrimaryStructured,
		Type:        model.AttributionSourceTypeMiningPool,
		Description: "Tier 1 Bitcoin mining-pool context used for contextual false-positive suppression.",
	})
}

func loadWalletExplorerStyleRecords() []model.AttributionRecord {
	paths := candidatePaths("WALLET_EXPLORER_LABELS_PATH", []string{
		"./data/labels/tier2_wallet_explorer_entities.json",
		"data/labels/tier2_wallet_explorer_entities.json",
	})
	return loadStructuredRecords(paths, model.AttributionSourceMetadata{
		Name:        "wallet_explorer_style_labels",
		Tier:        model.AttributionSourceTierSecondary,
		Type:        model.AttributionSourceTypeWalletExplorerStyle,
		Description: "WalletExplorer-style corroborating labels used as secondary attribution input.",
	})
}

func loadSecondaryFixtureRecords() []model.AttributionRecord {
	paths := candidatePaths("CORROBORATING_LABELS_PATH", []string{
		"./data/labels/tier2_corroborating_entities.json",
		"data/labels/tier2_corroborating_entities.json",
	})
	return loadStructuredRecords(paths, model.AttributionSourceMetadata{
		Name:        "repo_safe_corroborating_labels",
		Tier:        model.AttributionSourceTierSecondary,
		Type:        model.AttributionSourceTypeSecondaryFixture,
		Description: "Repo-safe corroborating attribution fixture used for confidence uplift and conflict visibility.",
	})
}

func loadBootstrapOverrideRecords() []model.AttributionRecord {
	paths := candidatePaths("BOOTSTRAP_LABELS_PATH", []string{
		"./data/labels/bootstrap_entities.json",
		"data/labels/bootstrap_entities.json",
		"/root/data/labels/bootstrap_entities.json",
	})

	var lastErr error
	for _, path := range paths {
		raw, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			lastErr = err
			continue
		}

		var labels map[string]model.EntityLabel
		if err := json.Unmarshal(raw, &labels); err != nil {
			lastErr = err
			continue
		}

		out := make([]model.AttributionRecord, 0, len(labels))
		for address, label := range labels {
			normalized := normalizeAddress(address)
			if normalized == "" {
				continue
			}
			out = append(out, model.AttributionRecord{
				Address:    normalized,
				Network:    inferNetwork(normalized),
				Label:      firstNonEmpty(label.Name, label.Actor),
				Actor:      label.Actor,
				Category:   label.Category,
				RiskClass:  inferRiskClass(label.Category, label.Trusted || label.Contextual),
				Confidence: confidenceScore(label.Confidence),
				Contextual: label.Trusted || label.Contextual,
				Escalating: !(label.Trusted || label.Contextual),
				Source: model.AttributionSourceMetadata{
					Name:        "bootstrap_entities",
					Tier:        model.AttributionSourceTierLocalOverride,
					Type:        model.AttributionSourceTypeBootstrapDemo,
					Description: "Repo-local bootstrap labels kept for continuity and demos.",
				},
				Notes: label.Notes,
			})
		}
		return out
	}

	if lastErr != nil {
		log.Printf("[attribution] bootstrap override load failed: %v", lastErr)
	}

	return nil
}

func loadStructuredRecords(paths []string, source model.AttributionSourceMetadata) []model.AttributionRecord {
	var lastErr error
	for _, path := range paths {
		raw, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			lastErr = err
			continue
		}

		var records []model.AttributionRecord
		if err := json.Unmarshal(raw, &records); err != nil {
			lastErr = err
			continue
		}

		for idx := range records {
			records[idx].Source = source
		}
		return records
	}

	if lastErr != nil {
		log.Printf("[attribution] source load failed for %s: %v", source.Name, lastErr)
	}

	return nil
}

func candidatePaths(envKey string, defaults []string) []string {
	if path := strings.TrimSpace(os.Getenv(envKey)); path != "" {
		return []string{path}
	}

	root := repoRoot()
	out := make([]string, 0, len(defaults)*2)
	seen := map[string]struct{}{}

	for _, path := range defaults {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		candidates := []string{path}
		if !filepath.IsAbs(path) && root != "" {
			candidates = append(candidates, filepath.Join(root, strings.TrimPrefix(path, "./")))
		}

		for _, candidate := range candidates {
			cleaned := filepath.Clean(candidate)
			if _, ok := seen[cleaned]; ok {
				continue
			}
			seen[cleaned] = struct{}{}
			out = append(out, cleaned)
		}
	}

	return out
}

func uniqueResolvedSources(records []model.AttributionRecord, keep func(model.AttributionRecord) bool) []model.ResolvedAttributionSource {
	out := make([]model.ResolvedAttributionSource, 0, len(records))
	seen := map[string]struct{}{}

	for _, record := range records {
		if keep != nil && !keep(record) {
			continue
		}

		source := toResolvedSource(record)
		key := string(source.Tier) + "|" + string(source.Type) + "|" + source.Name + "|" + source.Label + "|" + source.Actor
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, source)
	}

	return out
}

func toResolvedSource(record model.AttributionRecord) model.ResolvedAttributionSource {
	return model.ResolvedAttributionSource{
		Name:        record.Source.Name,
		Tier:        record.Source.Tier,
		Type:        record.Source.Type,
		Label:       record.Label,
		Actor:       record.Actor,
		Category:    record.Category,
		RiskClass:   record.RiskClass,
		Confidence:  record.Confidence,
		Contextual:  record.Contextual,
		Escalating:  record.Escalating,
		Description: record.Source.Description,
	}
}

func sameSourceRecord(primary, candidate model.AttributionRecord) bool {
	return primary.Source.Name == candidate.Source.Name &&
		primary.Source.Tier == candidate.Source.Tier &&
		primary.Source.Type == candidate.Source.Type
}

func corroborates(primary, candidate model.AttributionRecord) bool {
	if sameSourceRecord(primary, candidate) {
		return true
	}
	if primary.Contextual != candidate.Contextual || primary.Escalating != candidate.Escalating {
		return false
	}
	if primary.Category != "" && primary.Category == candidate.Category {
		return true
	}
	if primary.RiskClass != "" && primary.RiskClass == candidate.RiskClass {
		return true
	}
	if sameIdentity(primary.Actor, candidate.Actor) || sameIdentity(primary.Label, candidate.Label) {
		return true
	}
	return false
}

func conflicts(primary, candidate model.AttributionRecord) bool {
	if sameSourceRecord(primary, candidate) {
		return false
	}
	if primary.Contextual != candidate.Contextual || primary.Escalating != candidate.Escalating {
		return true
	}
	if primary.Category != "" && candidate.Category != "" && primary.Category != candidate.Category &&
		primary.RiskClass != "" && candidate.RiskClass != "" && primary.RiskClass != candidate.RiskClass &&
		!sameIdentity(primary.Actor, candidate.Actor) &&
		!sameIdentity(primary.Label, candidate.Label) {
		return true
	}
	return false
}

func sameIdentity(left, right string) bool {
	left = strings.TrimSpace(strings.ToLower(left))
	right = strings.TrimSpace(strings.ToLower(right))
	return left != "" && right != "" && left == right
}

func normalizedBaseConfidence(record model.AttributionRecord) float64 {
	confidence := clampConfidence(record.Confidence)
	if record.Source.Tier == model.AttributionSourceTierSecondary {
		return mathMin(confidence, 0.58)
	}
	return confidence
}

func adjustedConfidence(base float64, primary model.AttributionRecord, corroborating []model.ResolvedAttributionSource, conflicting []model.ResolvedAttributionSource) float64 {
	confidence := base

	for _, source := range corroborating {
		if source.Tier == model.AttributionSourceTierSecondary {
			confidence += 0.07
			continue
		}
		confidence += 0.03
	}
	for _, source := range conflicting {
		if source.Tier == model.AttributionSourceTierSecondary {
			confidence -= 0.05
			continue
		}
		confidence -= 0.08
	}

	confidence = clampConfidence(confidence)
	if primary.Source.Tier == model.AttributionSourceTierSecondary {
		confidence = mathMin(confidence, 0.65)
	}
	return confidence
}

func clampConfidence(confidence float64) float64 {
	switch {
	case confidence < 0.35:
		return 0.35
	case confidence > 0.99:
		return 0.99
	default:
		return confidence
	}
}

func mathMin(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func normalizeAddress(address string) string {
	return strings.ToLower(strings.TrimSpace(address))
}

func inferNetwork(address string) string {
	switch {
	case strings.HasPrefix(address, "0x"):
		return "EVM"
	case strings.HasPrefix(address, "bc1"), strings.HasPrefix(address, "1"), strings.HasPrefix(address, "3"):
		return "BITCOIN"
	default:
		return ""
	}
}

func networkMatches(recordNetwork string, requested string) bool {
	recordNetwork = strings.ToUpper(strings.TrimSpace(recordNetwork))
	requested = strings.ToUpper(strings.TrimSpace(requested))
	if recordNetwork == "" || requested == "" {
		return true
	}
	return recordNetwork == requested
}

func tierRank(tier model.AttributionSourceTier) int {
	switch tier {
	case model.AttributionSourceTierLocalOverride:
		return 3
	case model.AttributionSourceTierPrimaryStructured:
		return 2
	case model.AttributionSourceTierSecondary:
		return 1
	default:
		return 0
	}
}

func confidenceBucket(conf float64) model.LabelConfidence {
	switch {
	case conf >= 0.85:
		return model.LabelConfidenceHigh
	case conf >= 0.60:
		return model.LabelConfidenceMedium
	default:
		return model.LabelConfidenceLow
	}
}

func confidenceScore(conf model.LabelConfidence) float64 {
	switch conf {
	case model.LabelConfidenceHigh:
		return 0.95
	case model.LabelConfidenceMedium:
		return 0.75
	case model.LabelConfidenceLow:
		return 0.55
	default:
		return 0.50
	}
}

func severityForResolved(resolved *model.ResolvedAttribution) model.LabelSeverity {
	if resolved == nil {
		return model.LabelSeverityLow
	}
	switch resolved.RiskClass {
	case model.AttributionRiskClassSanctioned:
		return model.LabelSeverityCritical
	case model.AttributionRiskClassIllicitService, model.AttributionRiskClassExploit, model.AttributionRiskClassScam:
		return model.LabelSeverityHigh
	case model.AttributionRiskClassExchange, model.AttributionRiskClassTrustedService, model.AttributionRiskClassMiningPool, model.AttributionRiskClassTreasury:
		return model.LabelSeverityLow
	default:
		if resolved.Escalating {
			return model.LabelSeverityHigh
		}
		return model.LabelSeverityLow
	}
}

func inferRiskClass(category model.LabelCategory, contextual bool) model.AttributionRiskClass {
	switch category {
	case model.LabelCategorySanctions:
		return model.AttributionRiskClassSanctioned
	case model.LabelCategoryMixer:
		return model.AttributionRiskClassIllicitService
	case model.LabelCategoryExploit:
		return model.AttributionRiskClassExploit
	case model.LabelCategoryScam:
		return model.AttributionRiskClassScam
	case model.LabelCategoryExchange:
		return model.AttributionRiskClassExchange
	case model.LabelCategoryMiningPool:
		return model.AttributionRiskClassMiningPool
	case model.LabelCategoryTreasury:
		return model.AttributionRiskClassTreasury
	case model.LabelCategoryTrusted, model.LabelCategoryProtocol:
		return model.AttributionRiskClassTrustedService
	default:
		if contextual {
			return model.AttributionRiskClassTrustedService
		}
		return model.AttributionRiskClassUnknown
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
