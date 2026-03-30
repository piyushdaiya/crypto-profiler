package model

type GraphActorSummary struct {
	Actor            string  `json:"actor"`
	Category         string  `json:"category,omitempty"`
	RiskClass        string  `json:"risk_class,omitempty"`
	Contextual       bool    `json:"contextual,omitempty"`
	RiskEscalating   bool    `json:"risk_escalating,omitempty"`
	Confidence       float64 `json:"confidence,omitempty"`
	InteractionCount int     `json:"interaction_count"`
	InboundCount     int     `json:"inbound_count,omitempty"`
	OutboundCount    int     `json:"outbound_count,omitempty"`
	UniqueAddresses  int     `json:"unique_addresses,omitempty"`
	Share            float64 `json:"share,omitempty"`
}

type GraphMotif struct {
	Code         string  `json:"code"`
	Summary      string  `json:"summary"`
	FromActor    string  `json:"from_actor,omitempty"`
	ToActor      string  `json:"to_actor,omitempty"`
	FromCategory string  `json:"from_category,omitempty"`
	ToCategory   string  `json:"to_category,omitempty"`
	Count        int     `json:"count,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
}

type GraphSummary struct {
	TotalInteractions          int                 `json:"total_interactions"`
	AttributedInteractions     int                 `json:"attributed_interactions"`
	AttributedInteractionShare float64             `json:"attributed_interaction_share,omitempty"`
	UniqueActors               int                 `json:"unique_actors,omitempty"`
	TopActorShare              float64             `json:"top_actor_share,omitempty"`
	ConcentrationHHI           float64             `json:"concentration_hhi,omitempty"`
	DirectRiskyActorCount      int                 `json:"direct_risky_actor_count,omitempty"`
	DirectContextualActorCount int                 `json:"direct_contextual_actor_count,omitempty"`
	NearRiskyActorCount        int                 `json:"near_risky_actor_count,omitempty"`
	TopActors                  []GraphActorSummary `json:"top_actors,omitempty"`
	Motifs                     []GraphMotif        `json:"motifs,omitempty"`
}
