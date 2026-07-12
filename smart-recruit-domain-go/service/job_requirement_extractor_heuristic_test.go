package service

import (
	"context"
	"testing"
)

func TestHeuristicExtractorChineseJD(t *testing.T) {
	extractor := NewJobRequirementExtractorHeuristic()

	t.Run("Chinese Java backend JD", func(t *testing.T) {
		profile, hash, err := extractor.Extract(context.Background(),
			"Java 后端开发工程师",
			"技术部",
			"负责公司核心业务系统的后端服务设计与开发，包括微服务架构、分布式系统、数据库设计等。",
			"精通 Java 编程，熟悉 Spring Boot、MySQL、Redis，熟悉分布式系统设计，有微服务架构经验。3 年以上相关工作经验，本科及以上学历。",
		)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}
		if profile.ProfileVersion != RequirementProfileVersionV1 {
			t.Fatalf("expected profile version %s, got %s", RequirementProfileVersionV1, profile.ProfileVersion)
		}
		if len(profile.Requirements) == 0 {
			t.Fatal("expected at least one requirement")
		}
		if hash == "" {
			t.Fatal("expected non-empty input hash")
		}

		hasMustHave := false
		hasJava := false
		for _, req := range profile.Requirements {
			if req.Priority == RequirementPriorityMustHave {
				hasMustHave = true
			}
			if req.ID == "java" {
				hasJava = true
			}
			if req.ID == "education-本科" {
				if !req.Knockout {
					t.Fatal("expected education requirement to be knockout")
				}
			}
			if req.ID == "experience-years" {
				if !req.Knockout {
					t.Fatal("expected experience requirement to be knockout")
				}
			}
		}
		if !hasMustHave {
			t.Fatal("expected must_have requirements")
		}
		if !hasJava {
			t.Fatal("expected Java requirement")
		}
	})

	t.Run("Chinese JD with soft skills", func(t *testing.T) {
		profile, _, err := extractor.Extract(context.Background(),
			"产品经理",
			"产品部",
			"负责产品规划和迭代，需要良好的沟通能力和团队合作精神。",
			"有产品经验优先，沟通能力强，具备团队合作能力。",
		)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}
		hasSoftSkill := false
		for _, req := range profile.Requirements {
			if req.Priority == RequirementPrioritySoftSkill {
				hasSoftSkill = true
				break
			}
		}
		if !hasSoftSkill {
			t.Fatal("expected soft_skill priority requirements from Chinese soft skill mentions")
		}
	})

	t.Run("English JD", func(t *testing.T) {
		profile, hash, err := extractor.Extract(context.Background(),
			"Senior Backend Engineer",
			"Platform",
			"Build backend services and APIs.",
			"Go, PostgreSQL, Kubernetes, React. Bachelor degree required.",
		)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}
		if len(profile.Requirements) == 0 {
			t.Fatal("expected requirements")
		}
		if hash == "" {
			t.Fatal("expected non-empty hash")
		}

		hasGo := false
		hasPostgres := false
		for _, req := range profile.Requirements {
			if req.ID == "go" {
				hasGo = true
				if req.Priority != RequirementPriorityMustHave {
					t.Fatal("expected Go to be must_have")
				}
			}
			if req.ID == "postgresql" || req.ID == "postgres" {
				hasPostgres = true
			}
		}
		if !hasGo {
			t.Fatal("expected Go requirement")
		}
		if !hasPostgres {
			t.Fatal("expected PostgreSQL requirement")
		}
	})

	t.Run("bare minimum JD with no requirements", func(t *testing.T) {
		profile, hash, err := extractor.Extract(context.Background(),
			"Intern",
			"General",
			"",
			"",
		)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}
		if len(profile.Requirements) == 0 {
			t.Fatal("expected at least fallback requirement")
		}
		if hash == "" {
			t.Fatal("expected non-empty hash")
		}
	})

	t.Run("duplicate requirements are merged", func(t *testing.T) {
		profile, _, err := extractor.Extract(context.Background(),
			"Full Stack Developer",
			"Engineering",
			"Java development with Spring Boot",
			"Java, Spring, Java, Spring", // duplicates
		)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}
		javaCount := 0
		for _, req := range profile.Requirements {
			if req.ID == "java" {
				javaCount++
			}
		}
		if javaCount > 1 {
			t.Fatalf("expected unique Java requirement, got %d", javaCount)
		}
	})

	t.Run("weights sum to 1.0", func(t *testing.T) {
		profile, _, err := extractor.Extract(context.Background(),
			"Data Engineer",
			"Data",
			"Build data pipelines with Python and SQL",
			"Python, SQL, Spark, Kafka, 3+ years experience",
		)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}
		total := 0.0
		for _, req := range profile.Requirements {
			total += req.Weight
		}
		if total < 0.99 || total > 1.01 {
			t.Fatalf("expected weights to sum to ~1.0, got %f", total)
		}
	})

	t.Run("Chinese skill phrases", func(t *testing.T) {
		profile, _, err := extractor.Extract(context.Background(),
			"架构师",
			"架构部",
			"负责微服务架构设计和分布式系统开发",
			"熟悉微服务架构、分布式系统、大数据处理",
		)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}
		hasMicroservice := false
		hasDistributed := false
		for _, req := range profile.Requirements {
			if req.ID == "phrase-微服务" {
				hasMicroservice = true
			}
			if req.ID == "phrase-分布式" {
				hasDistributed = true
			}
		}
		if !hasMicroservice {
			t.Fatal("expected microservice requirement from Chinese phrase")
		}
		if !hasDistributed {
			t.Fatal("expected distributed systems requirement from Chinese phrase")
		}
	})
}

func TestComputeInputHash(t *testing.T) {
	h1 := computeInputHash("test", "v1")
	h2 := computeInputHash("test", "v1")
	h3 := computeInputHash("test", "v2")

	if h1 != h2 {
		t.Fatal("expected same hash for same input")
	}
	if h1 == h3 {
		t.Fatal("expected different hash for different input")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 char hex hash, got %d", len(h1))
	}
}

func TestGenerateRequirementAliases(t *testing.T) {
	req := &JobRequirementItem{
		ID:      "java",
		Label:   "Java 后端开发经验",
		Aliases: []string{"Java", "Spring Boot"},
	}
	aliases := GenerateRequirementAliases(req)
	if len(aliases) < 2 {
		t.Fatalf("expected at least 2 aliases, got %d", len(aliases))
	}
	hasID := false
	for _, alias := range aliases {
		if alias == "java" {
			hasID = true
			break
		}
	}
	if !hasID {
		t.Fatal("expected ID to be included in aliases")
	}
}

func TestNormalizeWeights(t *testing.T) {
	items := []JobRequirementItem{
		{ID: "a", Weight: 0.5},
		{ID: "b", Weight: 0.3},
		{ID: "c", Weight: 0.2},
	}
	normalizeWeights(items)
	total := 0.0
	for _, item := range items {
		total += item.Weight
	}
	if total < 0.99 || total > 1.01 {
		t.Fatalf("expected weights to sum to ~1.0, got %f", total)
	}
}
