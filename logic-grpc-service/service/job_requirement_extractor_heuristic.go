package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"regexp"
	"sort"
	"strings"
)

const heuristicExtractorVersion = "job-requirement-heuristic-v1"

type JobRequirementExtractorHeuristic struct{}

func NewJobRequirementExtractorHeuristic() *JobRequirementExtractorHeuristic {
	return &JobRequirementExtractorHeuristic{}
}

var (
	categoryByToken = map[string]string{
		"java":          RequirementCategoryCoreSkill,
		"golang":        RequirementCategoryCoreSkill,
		"go":            RequirementCategoryCoreSkill,
		"python":        RequirementCategoryCoreSkill,
		"vue":           RequirementCategoryCoreSkill,
		"react":         RequirementCategoryCoreSkill,
		"mysql":         RequirementCategoryCoreSkill,
		"redis":         RequirementCategoryCoreSkill,
		"kubernetes":    RequirementCategoryCoreSkill,
		"docker":        RequirementCategoryCoreSkill,
		"linux":         RequirementCategoryCoreSkill,
		"git":           RequirementCategoryCoreSkill,
		"grpc":          RequirementCategoryCoreSkill,
		"kafka":         RequirementCategoryCoreSkill,
		"elasticsearch": RequirementCategoryCoreSkill,
		"postgresql":    RequirementCategoryCoreSkill,
		"postgres":      RequirementCategoryCoreSkill,
		"typescript":    RequirementCategoryCoreSkill,
		"javascript":    RequirementCategoryCoreSkill,
		"css":           RequirementCategoryCoreSkill,
		"html":          RequirementCategoryCoreSkill,
		"node":          RequirementCategoryCoreSkill,
		"c++":           RequirementCategoryCoreSkill,
		"c#":            RequirementCategoryCoreSkill,
		"aws":           RequirementCategoryCoreSkill,
		"azure":         RequirementCategoryCoreSkill,
		"rabbitmq":      RequirementCategoryCoreSkill,
	}

	educationMarkers = map[string]string{
		"博士":       "博士",
		"phd":      "博士",
		"doctor":   "博士",
		"硕士":       "硕士",
		"master":   "硕士",
		"本科":       "本科",
		"bachelor": "本科",
		"大专":       "大专",
	}

	experiencePattern = regexp.MustCompile(`(\d+)\s*[年 years]+`)
)

func (e *JobRequirementExtractorHeuristic) Extract(_ context.Context, jobTitle, department, description, requirements string) (*JobRequirementProfile, string, error) {
	fullText := strings.Join([]string{jobTitle, department, description, requirements}, " ")

	items := e.extractRequirements(fullText, requirements)
	if len(items) == 0 {
		items = append(items, JobRequirementItem{
			ID:       "general-requirement",
			Category: RequirementCategoryOther,
			Label:    "基本岗位要求",
			Priority: RequirementPriorityMustHave,
			Weight:   1.0,
		})
	}

	normalizeWeights(items)

	inputHash := computeInputHash(fullText, heuristicExtractorVersion)

	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements:   items,
	}

	return profile, inputHash, nil
}

func (e *JobRequirementExtractorHeuristic) ExtractWithMetadata(ctx context.Context, jobTitle, department, description, requirements string) (*JobRequirementExtractResult, error) {
	profile, inputHash, err := e.Extract(ctx, jobTitle, department, description, requirements)
	if err != nil {
		return nil, err
	}
	return &JobRequirementExtractResult{
		Profile:   profile,
		InputHash: inputHash,
		Metadata: ExtractorMetadata{
			ParserVersion: heuristicExtractorVersion,
			ExtractorType: "heuristic_fallback",
			FallbackUsed:  false,
		},
	}, nil
}

func (e *JobRequirementExtractorHeuristic) extractRequirements(fullText, requirements string) []JobRequirementItem {
	text := strings.ToLower(fullText)
	var items []JobRequirementItem
	seen := make(map[string]bool)
	score := 1.0

	skillTokens := uniqueSortedTokens(requirements)
	if len(skillTokens) == 0 {
		skillTokens = uniqueSortedTokens(fullText)
	}

	for _, token := range skillTokens {
		if candidateMatchStopwords[token] || seen[token] {
			continue
		}
		seen[token] = true

		cat, isKnown := categoryByToken[token]
		if !isKnown {
			cat = RequirementCategoryOther
		}

		priority := RequirementPriorityNiceToHave
		if isKnown {
			priority = RequirementPriorityMustHave
		}

		var chineseLabel string
		if label, ok := knownSkillChineseLabels[token]; ok {
			chineseLabel = label
		} else {
			chineseLabel = token
		}

		items = append(items, JobRequirementItem{
			ID:          token,
			Category:    cat,
			Label:       chineseLabel,
			Description: chineseLabel,
			Priority:    priority,
			Weight:      score,
			Aliases:     []string{token},
		})
		score = math.Max(0.3, score-0.05)
	}

	items = e.extractEducationRequirements(text, items, seen)
	items = e.extractExperienceRequirements(text, items, seen)
	items = e.extractSkillPhrases(fullText, items, seen)
	_ = seen

	return items
}

var knownSkillChineseLabels = map[string]string{
	"java":          "Java",
	"golang":        "Go / Golang",
	"go":            "Go / Golang",
	"python":        "Python",
	"vue":           "Vue.js",
	"react":         "React",
	"mysql":         "MySQL 数据库",
	"redis":         "Redis 缓存",
	"kubernetes":    "Kubernetes 容器编排",
	"docker":        "Docker 容器化",
	"linux":         "Linux 系统",
	"git":           "Git 版本控制",
	"grpc":          "gRPC 通信",
	"kafka":         "Kafka 消息队列",
	"elasticsearch": "Elasticsearch 搜索引擎",
	"postgresql":    "PostgreSQL 数据库",
	"postgres":      "PostgreSQL 数据库",
	"typescript":    "TypeScript",
	"javascript":    "JavaScript",
	"css":           "CSS 样式",
	"html":          "HTML",
	"node":          "Node.js",
	"c++":           "C++",
	"c#":            "C#",
	"aws":           "AWS 云服务",
	"azure":         "Azure 云服务",
	"rabbitmq":      "RabbitMQ 消息队列",
}

var chineseSkillPatterns = []struct {
	pattern  string
	category string
	label    string
	priority string
}{
	{pattern: "java", category: RequirementCategoryCoreSkill, label: "Java", priority: RequirementPriorityMustHave},
	{pattern: "spring", category: RequirementCategoryCoreSkill, label: "Spring 框架", priority: RequirementPriorityMustHave},
	{pattern: "微服务", category: RequirementCategoryCoreSkill, label: "微服务架构", priority: RequirementPriorityMustHave},
	{pattern: "分布式", category: RequirementCategoryCoreSkill, label: "分布式系统", priority: RequirementPriorityMustHave},
	{pattern: "数据库", category: RequirementCategoryCoreSkill, label: "数据库开发", priority: RequirementPriorityMustHave},
	{pattern: "前端", category: RequirementCategoryCoreSkill, label: "前端开发", priority: RequirementPriorityNiceToHave},
	{pattern: "后端", category: RequirementCategoryCoreSkill, label: "后端开发", priority: RequirementPriorityMustHave},
	{pattern: "全栈", category: RequirementCategoryCoreSkill, label: "全栈开发", priority: RequirementPriorityNiceToHave},
	{pattern: "算法", category: RequirementCategoryCoreSkill, label: "算法能力", priority: RequirementPriorityMustHave},
	{pattern: "机器学", category: RequirementCategoryCoreSkill, label: "机器学习", priority: RequirementPriorityNiceToHave},
	{pattern: "深度学", category: RequirementCategoryCoreSkill, label: "深度学习", priority: RequirementPriorityNiceToHave},
	{pattern: "测试", category: RequirementCategoryCoreSkill, label: "测试能力", priority: RequirementPriorityNiceToHave},
	{pattern: "运维", category: RequirementCategoryCoreSkill, label: "运维能力", priority: RequirementPriorityNiceToHave},
	{pattern: "安全", category: RequirementCategoryCoreSkill, label: "安全能力", priority: RequirementPriorityNiceToHave},
	{pattern: "大数据", category: RequirementCategoryCoreSkill, label: "大数据", priority: RequirementPriorityNiceToHave},
	{pattern: "云原生", category: RequirementCategoryCoreSkill, label: "云原生", priority: RequirementPriorityNiceToHave},
	{pattern: "敏捷开发", category: RequirementCategorySoftSkill, label: "敏捷开发能力", priority: RequirementPrioritySoftSkill},
	{pattern: "团队合作", category: RequirementCategorySoftSkill, label: "团队合作", priority: RequirementPrioritySoftSkill},
	{pattern: "沟通", category: RequirementCategorySoftSkill, label: "沟通能力", priority: RequirementPrioritySoftSkill},
	{pattern: "英语", category: RequirementCategoryLanguage, label: "英语能力", priority: RequirementPriorityMustHave},
	{pattern: "英文", category: RequirementCategoryLanguage, label: "英语能力", priority: RequirementPriorityMustHave},
}

func (e *JobRequirementExtractorHeuristic) extractEducationRequirements(text string, items []JobRequirementItem, seen map[string]bool) []JobRequirementItem {
	lower := strings.ToLower(text)
	for marker, degree := range educationMarkers {
		if strings.Contains(lower, marker) {
			id := "education-" + degree
			if seen[id] {
				continue
			}
			seen[id] = true
			items = append(items, JobRequirementItem{
				ID:          id,
				Category:    RequirementCategoryEducation,
				Label:       degree + "及以上学历",
				Description: degree + "及以上学历要求",
				Priority:    RequirementPriorityMustHave,
				Weight:      0.5,
				Knockout:    true,
				Aliases:     nil,
			})
		}
	}
	return items
}

func (e *JobRequirementExtractorHeuristic) extractExperienceRequirements(text string, items []JobRequirementItem, seen map[string]bool) []JobRequirementItem {
	matches := experiencePattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		id := "experience-years"
		if !seen[id] {
			seen[id] = true
			years := matches[1]
			items = append(items, JobRequirementItem{
				ID:          id,
				Category:    RequirementCategoryExperience,
				Label:       years + "年以上工作经验",
				Description: years + "年以上工作经验要求",
				Priority:    RequirementPriorityMustHave,
				Weight:      0.4,
				Knockout:    true,
			})
		}
	} else {
		for _, marker := range []string{"经验", "experience"} {
			if strings.Contains(text, marker) {
				id := "experience-required"
				if !seen[id] {
					seen[id] = true
					items = append(items, JobRequirementItem{
						ID:          id,
						Category:    RequirementCategoryExperience,
						Label:       "相关工作经验",
						Description: "具备相关工作经验",
						Priority:    RequirementPriorityMustHave,
						Weight:      0.3,
					})
				}
				break
			}
		}
	}
	return items
}

func (e *JobRequirementExtractorHeuristic) extractSkillPhrases(fullText string, items []JobRequirementItem, seen map[string]bool) []JobRequirementItem {
	lower := strings.ToLower(fullText)
	for _, sp := range chineseSkillPatterns {
		if strings.Contains(lower, sp.pattern) {
			id := "phrase-" + sp.pattern
			if seen[id] {
				continue
			}
			seen[id] = true

			duplicate := false
			for _, existing := range items {
				if existing.Label == sp.label || existing.ID == id {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}

			items = append(items, JobRequirementItem{
				ID:          id,
				Category:    sp.category,
				Label:       sp.label,
				Description: sp.label,
				Priority:    sp.priority,
				Weight:      0.3,
			})
		}
	}

	if seen["沟通"] || seen["phrase-沟通"] {
		return items
	}
	return items
}

func normalizeWeights(items []JobRequirementItem) {
	total := 0.0
	for _, item := range items {
		total += item.Weight
	}
	if total == 0 {
		return
	}
	for i := range items {
		items[i].Weight = math.Round(items[i].Weight/total*100) / 100
	}

	adjusted := 0.0
	for i := range items {
		items[i].Weight = math.Round(items[i].Weight*100) / 100
		adjusted += items[i].Weight
	}
	if diff := math.Round((1.0-adjusted)*100) / 100; diff != 0 {
		items[len(items)-1].Weight = math.Round((items[len(items)-1].Weight+diff)*100) / 100
	}
}

func computeInputHash(texts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(texts, "\n")))
	return hex.EncodeToString(sum[:])
}

func GenerateRequirementAliases(req *JobRequirementItem) []string {
	aliases := make([]string, 0, len(req.Aliases)+2)
	aliases = append(aliases, req.Aliases...)
	aliases = append(aliases, req.ID)

	lower := strings.ToLower(req.Label)
	for _, sep := range []string{"、", "/", "，", ","} {
		if strings.Contains(lower, sep) {
			for _, part := range strings.Split(lower, sep) {
				part = strings.TrimSpace(part)
				if part != "" && part != req.Label {
					aliases = append(aliases, part)
				}
			}
		}
	}
	sort.Strings(aliases)

	unique := make([]string, 0, len(aliases))
	seen := make(map[string]bool)
	for _, a := range aliases {
		if !seen[a] {
			seen[a] = true
			unique = append(unique, a)
		}
	}
	return unique
}
