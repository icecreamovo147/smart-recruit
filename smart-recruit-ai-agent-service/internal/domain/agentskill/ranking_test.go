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
	decisions := RankCandidateDecisions("resume", []RankingCandidate{{
		Document: RankingDocument{
			ObjectID: 1, VersionID: 1, Priority: 1000, VectorScore: 0.20,
		},
		EmbeddingAvailable: true,
	}})
	if len(decisions) != 1 || decisions[0].Signals.PassedGate {
		t.Fatalf("below-gate decision evidence = %+v, want retained rejection", decisions)
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

func TestChineseLexicalFallbackRecallsCandidateScreeningSkill(t *testing.T) {
	query := `我要筛选一名 Java 后端候选人。

岗位要求：
- 5 年以上 Java 经验
- 熟悉 Spring Boot 和 MySQL
- 有微服务架构经验

候选人信息：
- 4 年 Java 经验
- 熟悉 Spring Boot 和 PostgreSQL
- 参与过两个微服务项目
- 没有提供 MySQL 项目证据

请评估该候选人与岗位的匹配情况。`
	document := RankingDocument{
		ObjectID:  101,
		VersionID: 101,
		LexicalText: []string{
			"候选人筛选方法",
			"评估候选人与岗位要求的匹配情况",
			"Java Spring Boot MySQL 微服务",
		},
		MetadataTerms: []string{"候选人筛选", "candidate screening"},
	}

	signals := ScoreRankingSignals(query, document, false)
	if !signals.PassedGate || signals.Mode != RelevanceModeLexicalMetadata {
		t.Fatalf("Chinese fallback signals = %+v, want candidate screening Skill to pass", signals)
	}
	if signals.LexicalScore < 0.4 || signals.RelevanceScore < RelevanceGate {
		t.Fatalf("Chinese fallback score = %+v, want stable relevance above gate", signals)
	}
}

func TestChineseLexicalFallbackRejectsUnrelatedSkill(t *testing.T) {
	signals := ScoreRankingSignals(
		"安排销售团队年度团建预算并预订场地",
		RankingDocument{
			ObjectID:      101,
			VersionID:     101,
			LexicalText:   []string{"候选人筛选方法", "评估 Java 后端候选人与岗位要求的匹配情况"},
			MetadataTerms: []string{"招聘", "微服务经验"},
		},
		false,
	)
	if signals.PassedGate {
		t.Fatalf("unrelated Chinese fallback signals = %+v, want below relevance gate", signals)
	}
}
