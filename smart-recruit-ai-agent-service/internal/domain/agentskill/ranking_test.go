package agentskill

import (
	"math"
	"testing"
)

func TestScoreRankingSignalsUsesCanonicalFormulaAndSingleBoost(t *testing.T) {
	signals := ScoreRankingSignals("resume screening", RankingDocument{
		LexicalText:   []string{"resume screening workflow"},
		MetadataTerms: []string{"resume screening"},
		Priority:      100,
		VectorScore:   0.5,
	}, true)

	wantRelevance := 0.5*VectorWeight + 1*LexicalWeight + 1*MetadataWeight
	if math.Abs(signals.RelevanceScore-wantRelevance) > 1e-9 {
		t.Fatalf("relevance = %f, want %f", signals.RelevanceScore, wantRelevance)
	}
	if signals.BusinessBoost != MaxBusinessBoost {
		t.Fatalf("business boost = %f, want %f", signals.BusinessBoost, MaxBusinessBoost)
	}
	if math.Abs(signals.FinalRankScore-(wantRelevance+MaxBusinessBoost)) > 1e-9 {
		t.Fatalf("final = %f, want a single boost over relevance", signals.FinalRankScore)
	}
	if signals.Mode != RelevanceModeHybrid || !signals.PassedGate {
		t.Fatalf("signals = %+v", signals)
	}
}

func TestScoreRankingSignalsFallsBackAndClampsPriority(t *testing.T) {
	signals := ScoreRankingSignals("candidate", RankingDocument{
		LexicalText: []string{"candidate review"},
		Priority:    -5000,
		VectorScore: 1,
	}, false)

	if signals.VectorScore != 0 || signals.Mode != RelevanceModeLexicalMetadata {
		t.Fatalf("fallback signals = %+v", signals)
	}
	if signals.RelevanceScore != FallbackLexicalWeight {
		t.Fatalf("fallback relevance = %f, want %f", signals.RelevanceScore, FallbackLexicalWeight)
	}
	if signals.BusinessBoost != -MaxBusinessBoost {
		t.Fatalf("business boost = %f, want %f", signals.BusinessBoost, -MaxBusinessBoost)
	}
}

func TestRankDocumentsAppliesGateBeforeBusinessBoost(t *testing.T) {
	ranked := RankDocuments("resume", []RankingDocument{{
		ObjectID:    1,
		VersionID:   1,
		Priority:    1000,
		VectorScore: 0.20,
	}}, true)
	if len(ranked) != 0 {
		t.Fatalf("business priority must not rescue an irrelevant document: %+v", ranked)
	}
}

func TestRankDocumentsUsesVersionAndSectionAsDeterministicTieBreakers(t *testing.T) {
	documents := []RankingDocument{
		{ObjectID: 30, VersionID: 2, SectionID: 5, LexicalText: []string{"resume"}},
		{ObjectID: 20, VersionID: 1, SectionID: 9, LexicalText: []string{"resume"}},
		{ObjectID: 10, VersionID: 1, SectionID: 3, LexicalText: []string{"resume"}},
	}
	ranked := RankDocuments("resume", documents, false)
	if len(ranked) != 3 {
		t.Fatalf("ranked = %+v", ranked)
	}
	got := []int64{ranked[0].Document.ObjectID, ranked[1].Document.ObjectID, ranked[2].Document.ObjectID}
	want := []int64{10, 20, 30}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tie order = %v, want %v", got, want)
		}
	}
}
