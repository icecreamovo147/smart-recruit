package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
	"smart-recruit-proto/recruitment/pb"
)

func embeddingUint64Ptr(value uint64) *uint64 {
	return &value
}

func TestHTTPEmbeddingRunnerCallsConfiguredEndpoint(t *testing.T) {
	var seenModel string
	var seenAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		var payload struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		seenModel = payload.Model
		_, _ = w.Write([]byte(`{"id":"req-1","data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer server.Close()

	result, err := NewHTTPEmbeddingRunner().Embed(context.Background(), EmbedRequest{
		Config: EmbeddingConfig{Endpoint: server.URL, APIKey: "secret", ModelName: "text-embedding", TimeoutSeconds: 2},
		Texts:  []string{"candidate matching"},
	})
	if err != nil {
		t.Fatalf("Embed returned %v", err)
	}
	if seenModel != "text-embedding" || seenAuth != "Bearer secret" {
		t.Fatalf("request model/auth = %q %q", seenModel, seenAuth)
	}
	if result.RequestID != "req-1" || len(result.Vectors) != 1 || len(result.Vectors[0]) != 3 {
		t.Fatalf("result = %+v", result)
	}
}

func TestEmbeddingServiceBackfillDebugAndSharedVersionRanking(t *testing.T) {
	store := newFakeEmbeddingStore()
	runner := &fakeEmbeddingRunner{vectors: map[string][]float64{
		"Smart Recruit embedding runtime validation": {1, 0},
		"简历":      {1, 0},
		"面试":      {0, 1},
		"请帮我筛选简历": {1, 0},
	}}
	service := NewEmbeddingService(store, runner)

	testResp, err := service.TestModel(context.Background(), &testModelRequest)
	if err != nil {
		t.Fatalf("TestModel returned %v", err)
	}
	if !testResp.GetSuccess() || testResp.GetDimension() != 2 || store.lastStatus != "success" {
		t.Fatalf("testResp=%+v status=%s", testResp, store.lastStatus)
	}

	backfill, err := service.Backfill(context.Background(), &backfillRequest)
	if err != nil {
		t.Fatalf("Backfill returned %v", err)
	}
	if backfill.GetSuccessCount() != 2 || len(store.embeddings) != 2 {
		t.Fatalf("backfill=%+v embeddings=%d", backfill, len(store.embeddings))
	}

	debug, err := service.SearchAgentSkills(context.Background(), "请帮我筛选简历", 5)
	if err != nil {
		t.Fatalf("SearchAgentSkills returned %v", err)
	}
	if !debug.GetEmbeddingAvailable() || debug.GetEmbeddingModel() != "text-embedding" || debug.GetEmbeddingDim() != 2 {
		t.Fatalf("debug metadata = %+v", debug)
	}
	if len(debug.GetSkills()) != 1 || debug.GetSkills()[0].GetName() != "resume_screen" || debug.GetSkills()[0].GetVersionId() != 101 || debug.GetSkills()[0].GetSkillId() != 1 || debug.GetSkills()[0].GetVectorScore() <= 0 {
		t.Fatalf("skills = %+v", debug.GetSkills())
	}
	ranked, err := service.SearchAgentSkillVersions(context.Background(), "请帮我筛选简历", 5)
	if err != nil || ranked.FallbackReason != "" || len(ranked.Items) != 1 || ranked.Items[0].Document.ID != 101 {
		t.Fatalf("ranked=%+v err=%v", ranked, err)
	}

	store.versionDocs[0].Manifest.Description = "旧简历向量应被失效"
	stale, err := service.SearchAgentSkills(context.Background(), "请帮我筛选简历", 5)
	if err != nil {
		t.Fatalf("SearchAgentSkills with stale hash returned %v", err)
	}
	if len(stale.GetSkills()) != 1 || stale.GetSkills()[0].GetVersionId() != 101 || stale.GetSkills()[0].GetVectorScore() != 0 || stale.GetSkills()[0].GetRelevanceMode() != string(domainagentskill.RelevanceModeLexicalMetadata) {
		t.Fatalf("stale embedding must be ignored by current text hash: %+v", stale)
	}
	if err := service.UpsertAgentSkillVersion(context.Background(), 101); err != nil {
		t.Fatalf("UpsertAgentSkillVersion returned %v", err)
	}
	debug, err = service.SearchAgentSkills(context.Background(), "请帮我筛选简历", 5)
	if err != nil {
		t.Fatalf("SearchAgentSkills after update returned %v", err)
	}
	var skillOneCount int
	for _, item := range debug.GetSkills() {
		if item.GetVersionId() == 101 {
			skillOneCount++
		}
	}
	if skillOneCount != 1 {
		t.Fatalf("stale skill embeddings should be invalidated, got %d entries: %+v", skillOneCount, debug.GetSkills())
	}
	var invalidated int
	for _, row := range store.embeddings {
		if row.ObjectType == agentSkillVersionObjectType && row.ObjectID == 101 && row.Status == "invalidated" {
			invalidated++
		}
	}
	if invalidated == 0 {
		t.Fatalf("expected the previous version embedding to be invalidated: %+v", store.embeddings)
	}
}

func TestEmbeddingServiceMemoryUpsertSearchAndBackfill(t *testing.T) {
	store := newFakeEmbeddingStore()
	store.memoryDocs = []MemoryEmbeddingDocument{
		{ID: 101, TenantID: embeddingUint64Ptr(11), OwnerRole: 2, OwnerID: 7, ScopeType: "hr", MemoryType: "preference", Content: "偏好远程办公", Source: "manual", Confidence: 0.8, Importance: 0.8},
	}
	runner := &fakeEmbeddingRunner{vectors: map[string][]float64{
		"preference manual 偏好远程办公": {1, 0},
		"远程办公":                     {1, 0},
	}}
	service := NewEmbeddingService(store, runner)

	memory := domainmemory.Memory{
		ID: 101, TenantID: embeddingUint64Ptr(11), OwnerRole: domainmemory.OwnerRoleHR, OwnerID: 7,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "preference", Content: "偏好远程办公", Source: "manual", Confidence: 0.8, Importance: 0.8,
	}
	if err := service.UpsertMemoryEmbedding(context.Background(), memory); err != nil {
		t.Fatalf("UpsertMemoryEmbedding() error = %v", err)
	}
	backfill, err := service.Backfill(context.Background(), &pb.BackfillEmbeddingsRequest{ObjectType: "ai_memory", Limit: 10})
	if err != nil {
		t.Fatalf("Backfill() error = %v", err)
	}
	if backfill.GetSuccessCount() < 1 {
		t.Fatalf("backfill=%+v", backfill)
	}
	debug, err := service.DebugSemanticRetrieval(context.Background(), &pb.DebugSemanticRetrievalRequest{
		HrId: 7, OwnerRole: 2, OwnerId: 7, TenantId: 11, Query: "远程办公", Limit: 5,
	})
	if err != nil {
		t.Fatalf("DebugSemanticRetrieval() error = %v", err)
	}
	if len(debug.GetMemories()) == 0 || debug.GetMemoryPoolConfidence() == "none" {
		t.Fatalf("memories debug = %+v", debug)
	}
	scores, fallback := service.SemanticMemoryScores(context.Background(), domainmemory.OwnerKey{TenantID: embeddingUint64Ptr(11), Role: domainmemory.OwnerRoleHR, ID: 7}, "远程办公", []domainmemory.Scope{{Type: domainmemory.ScopeHR, ID: 0}}, domainmemory.ScopeHR, 0, 5)
	if fallback != "" || scores[101] <= 0 {
		t.Fatalf("scores=%v fallback=%q", scores, fallback)
	}
	if err := service.InvalidateMemoryEmbedding(context.Background(), 101); err != nil {
		t.Fatalf("InvalidateMemoryEmbedding() error = %v", err)
	}
}

func TestEmbeddingServiceExplicitFallbackWhenRunnerFails(t *testing.T) {
	store := newFakeEmbeddingStore()
	service := NewEmbeddingService(store, &fakeEmbeddingRunner{errText: "provider unavailable"})
	resp, err := service.SearchAgentSkills(context.Background(), "请帮我筛选简历", 5)
	if err != nil {
		t.Fatalf("SearchAgentSkills returned %v", err)
	}
	if resp.GetEmbeddingAvailable() || !strings.Contains(resp.GetFallbackReason(), "provider unavailable") || len(resp.GetSkills()) == 0 {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.GetSkills()[0].GetName() != "resume_screen" || resp.GetSkills()[0].GetRelevanceMode() != "lexical_metadata" || resp.GetSkills()[0].GetVectorScore() != 0 {
		t.Fatalf("fallback skills = %+v", resp.GetSkills())
	}
	ranked, err := service.SearchAgentSkillVersions(context.Background(), "请帮我筛选简历", 5)
	if err != nil || len(ranked.Items) == 0 || ranked.Items[0].Ranking.Signals.FinalRankScore <= 0 || !strings.Contains(ranked.FallbackReason, "provider unavailable") {
		t.Fatalf("fallback ranked=%+v err=%v", ranked, err)
	}
}

func TestAgentSkillVersionEmbeddingTextBoundsCoreAndExcludesSections(t *testing.T) {
	corePrefix := strings.Repeat("核", maxCatalogCoreSummaryRunes)
	doc := newFakeEmbeddingStore().versionDocs[0]
	doc.CoreMarkdown = corePrefix + "不得进入目录索引的尾部"

	text := AgentSkillVersionEmbeddingText(doc)
	if strings.Contains(text, "不得进入目录索引的尾部") {
		t.Fatalf("catalog embedding leaked core content beyond the bounded summary")
	}
	if !strings.Contains(text, corePrefix) {
		t.Fatalf("catalog embedding omitted the bounded core summary")
	}
	if strings.Contains(text, "简历筛选评分细则") {
		t.Fatalf("catalog embedding must never include reference section content")
	}
}

func TestEmbeddingServiceSectionSearchRequiresAndEnforcesVersionScope(t *testing.T) {
	store := newFakeEmbeddingStore()
	service := NewEmbeddingService(store, &fakeEmbeddingRunner{errText: "provider unavailable"})

	unscoped, err := service.SearchAgentSkillSections(context.Background(), "简历", nil, 5)
	if err != nil {
		t.Fatalf("unscoped section search returned %v", err)
	}
	if len(unscoped.Items) != 0 || !strings.Contains(unscoped.FallbackReason, "version scope is required") {
		t.Fatalf("unscoped section search must fail closed: %+v", unscoped)
	}

	scoped, err := service.SearchAgentSkillSections(context.Background(), "请帮我筛选简历", []int64{101, 101, -1}, 5)
	if err != nil {
		t.Fatalf("scoped section search returned %v", err)
	}
	if scoped.EmbeddingAvailable || !strings.Contains(scoped.FallbackReason, "provider unavailable") {
		t.Fatalf("scoped fallback metadata = %+v", scoped)
	}
	if len(scoped.Items) != 1 || scoped.Items[0].Document.ID != 1001 || scoped.Items[0].Document.VersionID != 101 {
		t.Fatalf("section search escaped selected version scope: %+v", scoped.Items)
	}
	if scoped.Items[0].Ranking.Signals.Mode != domainagentskill.RelevanceModeLexicalMetadata {
		t.Fatalf("section fallback signals = %+v", scoped.Items[0].Ranking.Signals)
	}
}

func TestEmbeddingServiceBackfillsSectionAsIndependentVersionScopedObject(t *testing.T) {
	store := newFakeEmbeddingStore()
	service := NewEmbeddingService(store, &fakeEmbeddingRunner{})
	resp, err := service.Backfill(context.Background(), &pb.BackfillEmbeddingsRequest{
		ObjectType: agentSkillSectionObjectType,
		ObjectId:   1001,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("section backfill returned %v", err)
	}
	if resp.GetSuccessCount() != 1 {
		t.Fatalf("section backfill = %+v", resp)
	}
	if len(store.embeddings) != 1 {
		t.Fatalf("section embeddings = %+v", store.embeddings)
	}
	row := store.embeddings[0]
	if row.ObjectType != agentSkillSectionObjectType || row.ObjectID != 1001 || row.ScopeType != agentSkillVersionScopeType || row.ScopeID != 101 {
		t.Fatalf("section embedding identity/scope = %+v", row)
	}
}

func TestEmbeddingServiceBackfillCountsAgentSkillProviderFailures(t *testing.T) {
	tests := []struct {
		name       string
		objectType string
		objectID   int64
	}{
		{name: "version", objectType: agentSkillVersionObjectType, objectID: 101},
		{name: "section", objectType: agentSkillSectionObjectType, objectID: 1001},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newFakeEmbeddingStore()
			service := NewEmbeddingService(store, &fakeEmbeddingRunner{errText: "provider unavailable"})
			resp, err := service.Backfill(context.Background(), &pb.BackfillEmbeddingsRequest{
				ObjectType: test.objectType,
				ObjectId:   test.objectID,
				Limit:      10,
			})
			if err != nil {
				t.Fatalf("backfill returned transport error %v", err)
			}
			if resp.GetSuccessCount() != 0 || resp.GetFailedCount() != 1 || resp.GetSkippedCount() != 0 {
				t.Fatalf("provider failure counters = %+v", resp)
			}
			if len(store.embeddings) != 1 {
				t.Fatalf("failed diagnostic embeddings = %+v", store.embeddings)
			}
			row := store.embeddings[0]
			if row.ObjectType != test.objectType || row.ObjectID != test.objectID || row.Status != "failed" || !strings.Contains(row.LastError, "provider unavailable") {
				t.Fatalf("failed diagnostic embedding = %+v", row)
			}
		})
	}
}

func TestEmbeddingServiceRejectsMismatchedReadyEmbeddingIdentity(t *testing.T) {
	versionStore := newFakeEmbeddingStore()
	versionDoc := versionStore.versionDocs[0]
	validVersionRow := AIEmbeddingRecord{
		ObjectType:     agentSkillVersionObjectType,
		ObjectID:       versionDoc.ID,
		ScopeType:      agentSkillVersionScopeType,
		ScopeID:        versionDoc.ID,
		TextHash:       hashText(AgentSkillVersionEmbeddingText(versionDoc)),
		EmbeddingModel: versionStore.cfg.ModelName,
		Vector:         []float64{1, 0},
		Status:         "ready",
	}
	versionCases := []struct {
		name   string
		mutate func(*AIEmbeddingRecord)
	}{
		{name: "wrong object type", mutate: func(row *AIEmbeddingRecord) { row.ObjectType = agentSkillSectionObjectType }},
		{name: "wrong scope type", mutate: func(row *AIEmbeddingRecord) { row.ScopeType = "other" }},
		{name: "scope does not equal object", mutate: func(row *AIEmbeddingRecord) { row.ScopeID++ }},
		{name: "unknown object", mutate: func(row *AIEmbeddingRecord) { row.ObjectID++ }},
		{name: "stale hash", mutate: func(row *AIEmbeddingRecord) { row.TextHash = "stale" }},
	}
	for _, test := range versionCases {
		t.Run("version/"+test.name, func(t *testing.T) {
			store := newFakeEmbeddingStore()
			row := validVersionRow
			test.mutate(&row)
			store.catalogRowsOverride = []AIEmbeddingRecord{row}
			result, err := NewEmbeddingService(store, &fakeEmbeddingRunner{}).
				SearchAgentSkillVersions(context.Background(), "请帮我筛选简历", 5)
			if err != nil {
				t.Fatalf("version search returned %v", err)
			}
			assertAgentSkillVersionLexicalFallback(t, result)
		})
	}

	sectionStore := newFakeEmbeddingStore()
	sectionDoc := sectionStore.sectionDocs[0]
	validSectionRow := AIEmbeddingRecord{
		ObjectType:     agentSkillSectionObjectType,
		ObjectID:       sectionDoc.ID,
		ScopeType:      agentSkillVersionScopeType,
		ScopeID:        sectionDoc.VersionID,
		TextHash:       hashText(AgentSkillSectionEmbeddingText(sectionDoc)),
		EmbeddingModel: sectionStore.cfg.ModelName,
		Vector:         []float64{1, 0},
		Status:         "ready",
	}
	sectionCases := []struct {
		name   string
		mutate func(*AIEmbeddingRecord)
	}{
		{name: "wrong object type", mutate: func(row *AIEmbeddingRecord) { row.ObjectType = agentSkillVersionObjectType }},
		{name: "wrong scope type", mutate: func(row *AIEmbeddingRecord) { row.ScopeType = "other" }},
		{name: "scope does not equal version", mutate: func(row *AIEmbeddingRecord) { row.ScopeID++ }},
		{name: "unknown object", mutate: func(row *AIEmbeddingRecord) { row.ObjectID++ }},
		{name: "stale hash", mutate: func(row *AIEmbeddingRecord) { row.TextHash = "stale" }},
	}
	for _, test := range sectionCases {
		t.Run("section/"+test.name, func(t *testing.T) {
			store := newFakeEmbeddingStore()
			row := validSectionRow
			test.mutate(&row)
			store.sectionRowsOverride = []AIEmbeddingRecord{row}
			result, err := NewEmbeddingService(store, &fakeEmbeddingRunner{}).
				SearchAgentSkillSections(context.Background(), "请帮我筛选简历", []int64{sectionDoc.VersionID}, 5)
			if err != nil {
				t.Fatalf("section search returned %v", err)
			}
			assertAgentSkillSectionLexicalFallback(t, result)
		})
	}
}

func assertAgentSkillVersionLexicalFallback(t *testing.T, result *AgentSkillVersionSearchResult) {
	t.Helper()
	if result.EmbeddingAvailable || !strings.Contains(result.FallbackReason, "no ready agent_skill_version embeddings") {
		t.Fatalf("invalid version embedding identity must force lexical fallback: %+v", result)
	}
	if len(result.Items) == 0 || result.Items[0].Ranking.Signals.Mode != domainagentskill.RelevanceModeLexicalMetadata {
		t.Fatalf("version lexical fallback must remain available: %+v", result.Items)
	}
}

func assertAgentSkillSectionLexicalFallback(t *testing.T, result *AgentSkillSectionSearchResult) {
	t.Helper()
	if result.EmbeddingAvailable || !strings.Contains(result.FallbackReason, "no ready agent_skill_section embeddings") {
		t.Fatalf("invalid section embedding identity must force lexical fallback: %+v", result)
	}
	if len(result.Items) == 0 || result.Items[0].Ranking.Signals.Mode != domainagentskill.RelevanceModeLexicalMetadata {
		t.Fatalf("section lexical fallback must remain available: %+v", result.Items)
	}
}

func TestEmbeddingServiceRejectsLegacyAgentSkillObjectType(t *testing.T) {
	service := NewEmbeddingService(newFakeEmbeddingStore(), &fakeEmbeddingRunner{})
	resp, err := service.Backfill(context.Background(), &pb.BackfillEmbeddingsRequest{
		ObjectType: "agent_skill",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("legacy backfill returned transport error %v", err)
	}
	if resp.GetCode() != 400 || resp.GetSuccessCount() != 0 {
		t.Fatalf("legacy object type must be rejected: %+v", resp)
	}
}

type fakeEmbeddingRunner struct {
	vectors map[string][]float64
	errText string
}

func (f *fakeEmbeddingRunner) Embed(_ context.Context, req EmbedRequest) (EmbedResult, error) {
	if f.errText != "" {
		return EmbedResult{}, errString(f.errText)
	}
	out := make([][]float64, 0, len(req.Texts))
	for _, text := range req.Texts {
		if vec, ok := f.vectors[text]; ok {
			out = append(out, vec)
			continue
		}
		if strings.Contains(text, "简历") {
			out = append(out, []float64{1, 0})
		} else {
			out = append(out, []float64{0, 1})
		}
	}
	return EmbedResult{Vectors: out, RequestID: "fake", Latency: time.Millisecond}, nil
}

type errString string

func (e errString) Error() string { return string(e) }

type fakeEmbeddingStore struct {
	cfg                 EmbeddingConfig
	versionDocs         []AgentSkillVersionEmbeddingDocument
	sectionDocs         []AgentSkillSectionEmbeddingDocument
	memoryDocs          []MemoryEmbeddingDocument
	embeddings          []AIEmbeddingRecord
	catalogRowsOverride []AIEmbeddingRecord
	sectionRowsOverride []AIEmbeddingRecord
	lastStatus          string
}

func newFakeEmbeddingStore() *fakeEmbeddingStore {
	return &fakeEmbeddingStore{
		cfg: EmbeddingConfig{ProviderID: 1, ProviderName: "local", Endpoint: "http://example.test/embeddings", APIKey: "secret", ModelID: 2, ModelName: "text-embedding", Dimension: 2, TimeoutSeconds: 2},
		versionDocs: []AgentSkillVersionEmbeddingDocument{
			{
				ID: 101, SkillID: 1, Version: "2.0.0", CompiledHash: strings.Repeat("a", 64), CoreMarkdown: "简历",
				Manifest: domainagentskill.Manifest{
					SchemaVersion: 2, SkillName: "resume_screen", DisplayName: "Resume Screen", Description: "筛选简历",
					AgentType: "hr_recruiting_agent", Category: "candidate", Priority: 10,
					RiskLevel: domainagentskill.RiskLevelMedium, Composition: domainagentskill.Composition{Role: domainagentskill.CompositionRolePrimary},
					SemanticTags: []string{"简历"},
				},
			},
			{
				ID: 102, SkillID: 2, Version: "2.0.0", CompiledHash: strings.Repeat("b", 64), CoreMarkdown: "面试",
				Manifest: domainagentskill.Manifest{
					SchemaVersion: 2, SkillName: "interview_plan", DisplayName: "Interview Plan", Description: "安排面试",
					AgentType: "hr_recruiting_agent", Category: "interview", Priority: 1,
					RiskLevel: domainagentskill.RiskLevelLow, Composition: domainagentskill.Composition{Role: domainagentskill.CompositionRoleSupporting},
					SemanticTags: []string{"面试"},
				},
			},
		},
		sectionDocs: []AgentSkillSectionEmbeddingDocument{
			{ID: 1001, SkillID: 1, VersionID: 101, Version: "2.0.0", SectionKey: "resume-rubric", Title: "简历评分规则", ContentMarkdown: "简历筛选评分细则", TriggerTerms: []string{"简历"}, Priority: 10},
			{ID: 1002, SkillID: 2, VersionID: 102, Version: "2.0.0", SectionKey: "interview-rubric", Title: "面试评分规则", ContentMarkdown: "面试评分细则", TriggerTerms: []string{"面试"}, Priority: 10},
		},
	}
}

func (f *fakeEmbeddingStore) ResolveEmbeddingConfig(context.Context, int64, int64) (EmbeddingConfig, bool, error) {
	return f.cfg, true, nil
}

func (f *fakeEmbeddingStore) UpdateEmbeddingTestStatus(_ context.Context, _ int64, status, _ string, _ time.Time) error {
	f.lastStatus = status
	return nil
}

func (f *fakeEmbeddingStore) ListAgentSkillVersionEmbeddingDocuments(_ context.Context, versionID int64, _ int) ([]AgentSkillVersionEmbeddingDocument, error) {
	if versionID == 0 {
		return f.versionDocs, nil
	}
	for _, doc := range f.versionDocs {
		if doc.ID == versionID {
			return []AgentSkillVersionEmbeddingDocument{doc}, nil
		}
	}
	return nil, nil
}

func (f *fakeEmbeddingStore) ListAgentSkillSectionEmbeddingDocuments(_ context.Context, sectionID int64, versionIDs []int64, _ int) ([]AgentSkillSectionEmbeddingDocument, error) {
	allowed := map[int64]struct{}{}
	for _, versionID := range versionIDs {
		allowed[versionID] = struct{}{}
	}
	out := make([]AgentSkillSectionEmbeddingDocument, 0, len(f.sectionDocs))
	for _, doc := range f.sectionDocs {
		if sectionID > 0 && doc.ID != sectionID {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[doc.VersionID]; !ok {
				continue
			}
		}
		out = append(out, doc)
	}
	return out, nil
}

func (f *fakeEmbeddingStore) ListMemoryEmbeddingDocuments(_ context.Context, objectID int64, _ int) ([]MemoryEmbeddingDocument, error) {
	if len(f.memoryDocs) == 0 {
		return nil, nil
	}
	if objectID == 0 {
		return f.memoryDocs, nil
	}
	for _, doc := range f.memoryDocs {
		if doc.ID == objectID {
			return []MemoryEmbeddingDocument{doc}, nil
		}
	}
	return nil, nil
}

func (f *fakeEmbeddingStore) UpsertAIEmbedding(_ context.Context, row AIEmbeddingRecord) error {
	f.embeddings = append(f.embeddings, row)
	return nil
}

func (f *fakeEmbeddingStore) InvalidateAIEmbedding(_ context.Context, objectType string, objectID int64) error {
	for i := range f.embeddings {
		if f.embeddings[i].ObjectType == objectType && f.embeddings[i].ObjectID == objectID {
			f.embeddings[i].Status = "invalidated"
		}
	}
	return nil
}

func (f *fakeEmbeddingStore) ListAIEmbeddings(_ context.Context, objectType, _ string, _ int) ([]AIEmbeddingRecord, error) {
	if f.catalogRowsOverride != nil {
		return append([]AIEmbeddingRecord(nil), f.catalogRowsOverride...), nil
	}
	out := make([]AIEmbeddingRecord, 0, len(f.embeddings))
	for _, row := range f.embeddings {
		if row.ObjectType == objectType && row.Status == "ready" {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeEmbeddingStore) ListAIEmbeddingsByScopeIDs(_ context.Context, objectType, _ string, scopeType string, scopeIDs []int64, _ int) ([]AIEmbeddingRecord, error) {
	if f.sectionRowsOverride != nil {
		return append([]AIEmbeddingRecord(nil), f.sectionRowsOverride...), nil
	}
	allowed := map[int64]struct{}{}
	for _, scopeID := range scopeIDs {
		allowed[scopeID] = struct{}{}
	}
	out := make([]AIEmbeddingRecord, 0, len(f.embeddings))
	for _, row := range f.embeddings {
		if row.ObjectType != objectType || row.ScopeType != scopeType || row.Status != "ready" {
			continue
		}
		if _, ok := allowed[row.ScopeID]; !ok {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (f *fakeEmbeddingStore) ListAIEmbeddingsForOwner(_ context.Context, objectType, _ string, tenantID *uint64, ownerRole int32, ownerID uint64, _ int) ([]AIEmbeddingRecord, error) {
	out := make([]AIEmbeddingRecord, 0, len(f.embeddings))
	for _, row := range f.embeddings {
		if row.ObjectType != objectType || row.Status != "ready" {
			continue
		}
		if ownerID > 0 && floatMeta(row.Metadata, "owner_id") != float64(ownerID) {
			continue
		}
		if ownerRole > 0 && floatMeta(row.Metadata, "owner_role") != float64(ownerRole) {
			continue
		}
		_, hasTenant := row.Metadata["tenant_id"]
		if tenantID == nil && hasTenant {
			continue
		}
		if tenantID != nil && (!hasTenant || uint64(floatMeta(row.Metadata, "tenant_id")) != *tenantID) {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

var testModelRequest = pb.TestEmbeddingModelRequest{
	ProviderId: 1,
	ModelId:    2,
	TestText:   defaultEmbeddingTestText,
}

var backfillRequest = pb.BackfillEmbeddingsRequest{ObjectType: agentSkillVersionObjectType, Limit: 10}
