package agentskill

import (
	"cmp"
	"math"
	"slices"
	"strings"
)

const (
	VectorWeight           = 0.70
	LexicalWeight          = 0.20
	MetadataWeight         = 0.10
	FallbackLexicalWeight  = 0.70
	FallbackMetadataWeight = 0.30
	RelevanceGate          = 0.15
	MaxBusinessBoost       = 0.10
)

type RelevanceMode string

const (
	RelevanceModeHybrid          RelevanceMode = "hybrid"
	RelevanceModeLexicalMetadata RelevanceMode = "lexical_metadata"
)

type RankingDocument struct {
	ObjectID      int64
	SkillID       int64
	VersionID     int64
	SectionID     int64
	LexicalText   []string
	MetadataTerms []string
	Priority      int
	VectorScore   float64
}

type RankingSignals struct {
	VectorScore    float64
	LexicalScore   float64
	MetadataScore  float64
	RelevanceScore float64
	BusinessBoost  float64
	FinalRankScore float64
	Mode           RelevanceMode
	Reason         string
	PassedGate     bool
}

type RankedDocument struct {
	Document RankingDocument
	Signals  RankingSignals
}

type RankingCandidate struct {
	Document           RankingDocument
	EmbeddingAvailable bool
}

func ScoreRankingSignals(query string, document RankingDocument, embeddingAvailable bool) RankingSignals {
	lexical := termScore(query, document.LexicalText)
	metadata := termScore(query, document.MetadataTerms)
	vector := clamp(document.VectorScore, 0, 1)
	mode := RelevanceModeHybrid
	reason := "vector, lexical, and metadata ranking"
	relevance := vector*VectorWeight + lexical*LexicalWeight + metadata*MetadataWeight
	if !embeddingAvailable {
		vector = 0
		mode = RelevanceModeLexicalMetadata
		reason = "lexical and metadata fallback"
		relevance = lexical*FallbackLexicalWeight + metadata*FallbackMetadataWeight
	}
	passed := relevance >= RelevanceGate
	boost := 0.0
	if passed {
		boost = clamp(float64(document.Priority)/1000, -MaxBusinessBoost, MaxBusinessBoost)
	}
	return RankingSignals{
		VectorScore:    vector,
		LexicalScore:   lexical,
		MetadataScore:  metadata,
		RelevanceScore: relevance,
		BusinessBoost:  boost,
		FinalRankScore: relevance + boost,
		Mode:           mode,
		Reason:         reason,
		PassedGate:     passed,
	}
}

func RankDocuments(query string, documents []RankingDocument, embeddingAvailable bool) []RankedDocument {
	candidates := make([]RankingCandidate, 0, len(documents))
	for _, document := range documents {
		candidates = append(candidates, RankingCandidate{Document: document, EmbeddingAvailable: embeddingAvailable})
	}
	return RankCandidates(query, candidates)
}

func RankCandidates(query string, candidates []RankingCandidate) []RankedDocument {
	ranked := make([]RankedDocument, 0, len(candidates))
	for _, candidate := range candidates {
		signals := ScoreRankingSignals(query, candidate.Document, candidate.EmbeddingAvailable)
		if !signals.PassedGate {
			continue
		}
		ranked = append(ranked, RankedDocument{Document: candidate.Document, Signals: signals})
	}
	slices.SortFunc(ranked, func(a, b RankedDocument) int {
		if a.Signals.FinalRankScore != b.Signals.FinalRankScore {
			if a.Signals.FinalRankScore > b.Signals.FinalRankScore {
				return -1
			}
			return 1
		}
		if a.Document.VersionID != b.Document.VersionID {
			return cmp.Compare(a.Document.VersionID, b.Document.VersionID)
		}
		if a.Document.SectionID != b.Document.SectionID {
			return cmp.Compare(a.Document.SectionID, b.Document.SectionID)
		}
		return cmp.Compare(a.Document.ObjectID, b.Document.ObjectID)
	})
	return ranked
}

func termScore(query string, values []string) float64 {
	normalizedQuery := normalizeRankingText(query)
	if normalizedQuery == "" {
		return 0
	}
	normalizedValues := make([]string, 0, len(values))
	for _, value := range values {
		if normalized := normalizeRankingText(value); normalized != "" {
			normalizedValues = append(normalizedValues, normalized)
		}
	}
	if len(normalizedValues) == 0 {
		return 0
	}
	haystack := strings.Join(normalizedValues, " ")
	if strings.Contains(haystack, normalizedQuery) {
		return 1
	}
	queryTerms := strings.Fields(normalizedQuery)
	if len(queryTerms) == 0 {
		return 0
	}
	matched := 0
	for _, term := range queryTerms {
		for _, value := range normalizedValues {
			if strings.Contains(value, term) || strings.Contains(term, value) {
				matched++
				break
			}
		}
	}
	return clamp(float64(matched)/float64(len(queryTerms)), 0, 1)
}

func normalizeRankingText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}
