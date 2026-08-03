package agentskill

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"
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
	decisions := RankCandidateDecisions(query, candidates)
	ranked := make([]RankedDocument, 0, len(candidates))
	for _, decision := range decisions {
		if !decision.Signals.PassedGate {
			continue
		}
		ranked = append(ranked, decision)
	}
	return ranked
}

func RankCandidateDecisions(query string, candidates []RankingCandidate) []RankedDocument {
	ranked := make([]RankedDocument, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, RankedDocument{
			Document: candidate.Document,
			Signals:  ScoreRankingSignals(query, candidate.Document, candidate.EmbeddingAvailable),
		})
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
	queryTerms := rankingTerms(normalizedQuery)
	if len(queryTerms) == 0 {
		return 0
	}
	querySet := make(map[string]struct{}, len(queryTerms))
	for _, term := range queryTerms {
		querySet[term] = struct{}{}
	}
	best := 0.0
	for _, value := range normalizedValues {
		if strings.Contains(normalizedQuery, value) || strings.Contains(value, normalizedQuery) {
			return 1
		}
		valueTerms := rankingTerms(value)
		if len(valueTerms) == 0 {
			continue
		}
		matched := 0
		for _, term := range valueTerms {
			if _, ok := querySet[term]; ok {
				matched++
			}
		}
		if score := float64(matched) / float64(len(valueTerms)); score > best {
			best = score
		}
	}
	return clamp(best, 0, 1)
}

func normalizeRankingText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func rankingTerms(value string) []string {
	type tokenKind int
	const (
		tokenNone tokenKind = iota
		tokenWord
		tokenHan
	)
	seen := map[string]struct{}{}
	terms := make([]string, 0)
	var current []rune
	kind := tokenNone
	flush := func() {
		if len(current) == 0 {
			return
		}
		if kind == tokenHan {
			if len(current) == 1 {
				appendRankingTerm(&terms, seen, string(current))
			} else {
				for i := 0; i < len(current)-1; i++ {
					appendRankingTerm(&terms, seen, string(current[i:i+2]))
				}
			}
		} else {
			appendRankingTerm(&terms, seen, string(current))
		}
		current = current[:0]
	}
	for _, r := range []rune(strings.ToLower(value)) {
		nextKind := tokenNone
		switch {
		case unicode.Is(unicode.Han, r):
			nextKind = tokenHan
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			nextKind = tokenWord
		}
		if nextKind == tokenNone {
			flush()
			kind = tokenNone
			continue
		}
		if kind != tokenNone && nextKind != kind {
			flush()
		}
		kind = nextKind
		current = append(current, r)
	}
	flush()
	return terms
}

func appendRankingTerm(terms *[]string, seen map[string]struct{}, term string) {
	term = strings.TrimSpace(term)
	if term == "" {
		return
	}
	if _, ok := seen[term]; ok {
		return
	}
	seen[term] = struct{}{}
	*terms = append(*terms, term)
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}
