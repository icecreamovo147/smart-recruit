package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"smart-recruit-proto/recruitment/pb"
)

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

func TestEmbeddingServiceBackfillDebugAndSemanticScores(t *testing.T) {
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
	if len(debug.GetSkills()) != 2 || debug.GetSkills()[0].GetName() != "resume_screen" || debug.GetSkills()[0].GetVectorScore() <= debug.GetSkills()[1].GetVectorScore() {
		t.Fatalf("skills = %+v", debug.GetSkills())
	}
	scores, fallback := service.SemanticScores(context.Background(), "请帮我筛选简历", 5)
	if fallback != "" || scores[1] <= scores[2] {
		t.Fatalf("scores=%v fallback=%q", scores, fallback)
	}

	store.docs[0].Description = "旧简历向量应被失效"
	runner.vectors["旧简历向量应被失效"] = []float64{0, 1}
	if err := service.UpsertAgentSkill(context.Background(), 1); err != nil {
		t.Fatalf("UpsertAgentSkill returned %v", err)
	}
	debug, err = service.SearchAgentSkills(context.Background(), "请帮我筛选简历", 5)
	if err != nil {
		t.Fatalf("SearchAgentSkills after update returned %v", err)
	}
	var skillOneCount int
	for _, item := range debug.GetSkills() {
		if item.GetId() == 1 {
			skillOneCount++
		}
	}
	if skillOneCount != 1 {
		t.Fatalf("stale skill embeddings should be invalidated, got %d entries: %+v", skillOneCount, debug.GetSkills())
	}
}

func TestEmbeddingServiceExplicitFallbackWhenRunnerFails(t *testing.T) {
	store := newFakeEmbeddingStore()
	service := NewEmbeddingService(store, &fakeEmbeddingRunner{errText: "provider unavailable"})
	resp, err := service.SearchAgentSkills(context.Background(), "query", 5)
	if err != nil {
		t.Fatalf("SearchAgentSkills returned %v", err)
	}
	if resp.GetEmbeddingAvailable() || !strings.Contains(resp.GetFallbackReason(), "provider unavailable") {
		t.Fatalf("resp = %+v", resp)
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
	cfg        EmbeddingConfig
	docs       []AgentSkillEmbeddingDocument
	embeddings []AIEmbeddingRecord
	lastStatus string
}

func newFakeEmbeddingStore() *fakeEmbeddingStore {
	return &fakeEmbeddingStore{
		cfg: EmbeddingConfig{ProviderID: 1, ProviderName: "local", Endpoint: "http://example.test/embeddings", APIKey: "secret", ModelID: 2, ModelName: "text-embedding", Dimension: 2, TimeoutSeconds: 2},
		docs: []AgentSkillEmbeddingDocument{
			{ID: 1, Name: "resume_screen", DisplayName: "Resume Screen", Description: "筛选简历", Category: "candidate", Priority: 10, SemanticTags: []string{"简历"}, BodyMarkdown: "简历"},
			{ID: 2, Name: "interview_plan", DisplayName: "Interview Plan", Description: "安排面试", Category: "interview", Priority: 1, SemanticTags: []string{"面试"}, BodyMarkdown: "面试"},
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

func (f *fakeEmbeddingStore) ListAgentSkillEmbeddingDocuments(_ context.Context, objectID int64, _ int) ([]AgentSkillEmbeddingDocument, error) {
	if objectID == 0 {
		return f.docs, nil
	}
	for _, doc := range f.docs {
		if doc.ID == objectID {
			return []AgentSkillEmbeddingDocument{doc}, nil
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

func (f *fakeEmbeddingStore) ListAIEmbeddings(context.Context, string, string, int) ([]AIEmbeddingRecord, error) {
	return f.embeddings, nil
}

var testModelRequest = pb.TestEmbeddingModelRequest{
	ProviderId: 1,
	ModelId:    2,
	TestText:   defaultEmbeddingTestText,
}

var backfillRequest = pb.BackfillEmbeddingsRequest{ObjectType: "agent_skill", Limit: 10}
