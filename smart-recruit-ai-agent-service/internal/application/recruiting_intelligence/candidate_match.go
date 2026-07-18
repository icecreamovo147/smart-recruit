package recruiting_intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	MaxCandidateEvidenceUnits  = 100
	MaxCandidateEvidenceRunes  = 240
	MaxMatchEvidenceReferences = 10

	MatchStatusStrongMatch  = "strong_match"
	MatchStatusMatch        = "match"
	MatchStatusPartialMatch = "partial_match"
	MatchStatusWeakEvidence = "weak_evidence"
	MatchStatusMissing      = "missing"
	MatchStatusConflict     = "conflict"

	MatchEvaluatorDeterministic = "deterministic"
	MatchEvaluatorLLM           = "llm"
)

var (
	ErrCandidateMatchSchema     = errors.New("candidate matcher returned an invalid schema")
	ErrCandidateMatchProvenance = errors.New("candidate matcher returned invalid evidence provenance")
	ErrCandidateMatchPolicy     = errors.New("candidate matching is disabled")

	candidateEvidenceEmailPattern  = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	candidateEvidencePhonePatterns = []*regexp.Regexp{
		// Mainland China mobile numbers, including +86 and separated forms.
		regexp.MustCompile(`(?:\+?86[\s.-]?)?1[3-9]\d(?:[\s.-]?\d){8}`),
		// Mainland China fixed lines require an area-code separator. Requiring
		// it avoids treating ordinary years and long business numbers as phones.
		regexp.MustCompile(`0\d{2,3}[\s.-]\d{7,8}`),
		// Representative North American and other common international forms.
		// A leading country code and grouped subscriber number are mandatory.
		regexp.MustCompile(`\+1[\s.-]?\(?[2-9]\d{2}\)?[\s.-]?\d{3}[\s.-]?\d{4}`),
		regexp.MustCompile(`\+(?:44|49|61|81|82|91)[\s.-]?\d{1,4}(?:[\s.-]\d{2,4}){2,4}`),
	}
)

// CandidateEvidenceSource is the transport-neutral input for a trusted
// evidence index. IDs must be the persisted row IDs; this layer never invents
// synthetic source IDs.
type CandidateEvidenceSource struct {
	Skills           []CandidateEvidenceSkill
	Experiences      []CandidateEvidenceExperience
	Projects         []CandidateEvidenceProject
	Educations       []CandidateEvidenceEducation
	Resume           *CandidateEvidenceResume
	CandidateProfile *CandidateEvidenceProfile
}

type CandidateEvidenceSkill struct {
	ID       uint64
	Name     string
	Category string
	Level    string
	Evidence string
}

type CandidateEvidenceExperience struct {
	ID           uint64
	Company      string
	Title        string
	Description  string
	Achievements []string
}

type CandidateEvidenceProject struct {
	ID           uint64
	Name         string
	Role         string
	Description  string
	Technologies []string
	Highlights   []string
}

type CandidateEvidenceEducation struct {
	ID          uint64
	School      string
	Degree      string
	Major       string
	Description string
}

type CandidateEvidenceResume struct {
	ID         uint64
	ParsedText string
}

// Contact fields are deliberately absent. Only the existing non-contact
// candidate profile fields may enter the index.
type CandidateEvidenceProfile struct {
	ID             uint64
	Education      string
	School         string
	WorkExperience string
	Skills         string
}

type EvidenceUnit struct {
	SourceTable     string   `json:"source_table"`
	SourceID        uint64   `json:"source_id"`
	SourceType      string   `json:"source_type"`
	Snippet         string   `json:"snippet"`
	NormalizedTerms []string `json:"normalized_terms"`
}

type EvidenceIndex struct {
	units []EvidenceUnit
	byKey map[string]EvidenceUnit
}

func BuildEvidenceIndex(source CandidateEvidenceSource) EvidenceIndex {
	units := make([]EvidenceUnit, 0, len(source.Skills)+len(source.Experiences)+len(source.Projects)+len(source.Educations)+2)
	for _, item := range source.Skills {
		units = appendEvidenceUnit(units, "resume_skills", item.ID, "skill", []string{item.Name, item.Evidence, item.Level, item.Category})
	}
	for _, item := range source.Experiences {
		units = appendEvidenceUnit(units, "resume_experiences", item.ID, "experience", append([]string{item.Title, item.Company, item.Description}, item.Achievements...))
	}
	for _, item := range source.Projects {
		parts := []string{item.Name, item.Role, item.Description}
		parts = append(parts, item.Technologies...)
		parts = append(parts, item.Highlights...)
		units = appendEvidenceUnit(units, "resume_projects", item.ID, "project", parts)
	}
	for _, item := range source.Educations {
		units = appendEvidenceUnit(units, "resume_educations", item.ID, "education", []string{item.School, item.Degree, item.Major, item.Description})
	}
	if source.Resume != nil {
		units = appendEvidenceUnit(units, "resumes", source.Resume.ID, "resume_text", []string{source.Resume.ParsedText})
	}
	if source.CandidateProfile != nil {
		item := source.CandidateProfile
		units = appendEvidenceUnit(units, "candidate_profiles", item.ID, "candidate_profile", []string{item.Education, item.School, item.WorkExperience, item.Skills})
	}

	sort.Slice(units, func(i, j int) bool {
		left, right := units[i], units[j]
		leftRank, _ := candidateEvidenceSourceRank(left.SourceTable)
		rightRank, _ := candidateEvidenceSourceRank(right.SourceTable)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if left.SourceID != right.SourceID {
			return left.SourceID < right.SourceID
		}
		return left.Snippet < right.Snippet
	})

	// A database row is one provenance identity. Resolve duplicate caller data
	// deterministically, then apply the global bound.
	deduplicated := units[:0]
	seen := make(map[string]struct{}, len(units))
	for _, unit := range units {
		key := evidenceKey(unit.SourceTable, unit.SourceID)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduplicated = append(deduplicated, unit)
		if len(deduplicated) == MaxCandidateEvidenceUnits {
			break
		}
	}
	index := EvidenceIndex{units: cloneEvidenceUnits(deduplicated)}
	index.rebuildLookup()
	return index
}

// Units returns a defensive snapshot so callers cannot add non-allowlisted
// rows or mutate the trusted snippets used for provenance validation.
func (index EvidenceIndex) Units() []EvidenceUnit {
	return cloneEvidenceUnits(index.units)
}

func cloneEvidenceUnits(units []EvidenceUnit) []EvidenceUnit {
	cloned := make([]EvidenceUnit, len(units))
	copy(cloned, units)
	for unitIndex := range cloned {
		cloned[unitIndex].NormalizedTerms = append([]string(nil), cloned[unitIndex].NormalizedTerms...)
	}
	return cloned
}

func appendEvidenceUnit(units []EvidenceUnit, table string, id uint64, sourceType string, parts []string) []EvidenceUnit {
	if _, allowed := candidateEvidenceSourceRank(table); !allowed || id == 0 {
		return units
	}
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeCandidateEvidenceText(redactCandidateContactData(part))
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return units
	}
	snippet := boundCandidateEvidence(strings.Join(cleaned, " | "), MaxCandidateEvidenceRunes)
	if snippet == "" {
		return units
	}
	return append(units, EvidenceUnit{
		SourceTable:     table,
		SourceID:        id,
		SourceType:      sourceType,
		Snippet:         snippet,
		NormalizedTerms: candidateEvidenceTerms(snippet),
	})
}

func (index *EvidenceIndex) rebuildLookup() {
	index.byKey = make(map[string]EvidenceUnit, len(index.units))
	for _, unit := range index.units {
		if _, allowed := candidateEvidenceSourceRank(unit.SourceTable); !allowed || unit.SourceID == 0 {
			continue
		}
		index.byKey[evidenceKey(unit.SourceTable, unit.SourceID)] = unit
	}
}

func candidateEvidenceSourceRank(table string) (int, bool) {
	switch table {
	case "resume_skills":
		return 0, true
	case "resume_experiences":
		return 1, true
	case "resume_projects":
		return 2, true
	case "resume_educations":
		return 3, true
	case "resumes":
		return 4, true
	case "candidate_profiles":
		return 5, true
	default:
		return 0, false
	}
}

func (index EvidenceIndex) lookup(table string, id uint64) (EvidenceUnit, bool) {
	if index.byKey == nil {
		index.rebuildLookup()
	}
	unit, ok := index.byKey[evidenceKey(table, id)]
	return unit, ok
}

func evidenceKey(table string, id uint64) string {
	return fmt.Sprintf("%s#%d", table, id)
}

func redactCandidateContactData(value string) string {
	value = candidateEvidenceEmailPattern.ReplaceAllString(value, "[EMAIL]")
	return replaceCandidatePhoneNumbers(value)
}

func containsCandidateContactData(value string) bool {
	return candidateEvidenceEmailPattern.MatchString(value) || len(candidatePhoneNumberRanges(value)) > 0
}

type candidateContactRange struct {
	start int
	end   int
}

func replaceCandidatePhoneNumbers(value string) string {
	ranges := candidatePhoneNumberRanges(value)
	if len(ranges) == 0 {
		return value
	}
	var redacted strings.Builder
	previous := 0
	for _, match := range ranges {
		redacted.WriteString(value[previous:match.start])
		redacted.WriteString("[PHONE]")
		previous = match.end
	}
	redacted.WriteString(value[previous:])
	return redacted.String()
}

// candidatePhoneNumberRanges is the single source of truth for both
// redaction and leak detection. Matches embedded in a longer digit sequence
// are rejected so normal identifiers are not partially erased.
func candidatePhoneNumberRanges(value string) []candidateContactRange {
	var matches []candidateContactRange
	for _, pattern := range candidateEvidencePhonePatterns {
		for _, location := range pattern.FindAllStringIndex(value, -1) {
			if hasAdjacentASCIIDigit(value, location[0]-1) || hasAdjacentASCIIDigit(value, location[1]) {
				continue
			}
			matches = append(matches, candidateContactRange{start: location[0], end: location[1]})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start != matches[j].start {
			return matches[i].start < matches[j].start
		}
		return matches[i].end > matches[j].end
	})
	merged := matches[:0]
	for _, match := range matches {
		if len(merged) > 0 && match.start < merged[len(merged)-1].end {
			if match.end > merged[len(merged)-1].end {
				merged[len(merged)-1].end = match.end
			}
			continue
		}
		merged = append(merged, match)
	}
	return merged
}

func hasAdjacentASCIIDigit(value string, index int) bool {
	return index >= 0 && index < len(value) && value[index] >= '0' && value[index] <= '9'
}

func normalizeCandidateEvidenceText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func boundCandidateEvidence(value string, maxRunes int) string {
	value = normalizeCandidateEvidenceText(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func candidateEvidenceTerms(value string) []string {
	value = candidateEvidenceSearchableText(value)
	var terms []string
	seen := make(map[string]struct{})
	var token []rune
	flush := func() {
		if len(token) == 0 {
			return
		}
		term := strings.ToLower(string(token))
		if _, exists := seen[term]; !exists {
			seen[term] = struct{}{}
			terms = append(terms, term)
		}
		token = token[:0]
	}
	for _, current := range value {
		if unicode.IsLetter(current) || unicode.IsDigit(current) || current == '+' || current == '#' || current == '.' {
			token = append(token, current)
		} else {
			flush()
		}
	}
	flush()
	sort.Strings(terms)
	if len(terms) > 40 {
		terms = terms[:40]
	}
	return terms
}

func candidateEvidenceSearchableText(value string) string {
	value = strings.ReplaceAll(value, "[PHONE]", " ")
	return strings.ReplaceAll(value, "[EMAIL]", " ")
}

type MatchEvidenceReference struct {
	SourceTable string `json:"source_table"`
	SourceID    uint64 `json:"source_id"`
	Snippet     string `json:"snippet"`
	Reason      string `json:"reason"`
}

type RequirementMatchResult struct {
	RequirementID string                   `json:"requirement_id"`
	Status        string                   `json:"status"`
	Score         float64                  `json:"score"`
	Confidence    float64                  `json:"confidence"`
	Risk          string                   `json:"risk"`
	Evidence      []MatchEvidenceReference `json:"evidence"`
	EvaluatorType string                   `json:"evaluator_type"`
	ModelName     string                   `json:"model_name,omitempty"`
	Prompt        PromptDescriptor         `json:"-"`
	FallbackUsed  bool                     `json:"fallback_used"`
}

type matcherOutput struct {
	Status     string                   `json:"status"`
	Score      float64                  `json:"score"`
	Confidence float64                  `json:"confidence"`
	Risk       string                   `json:"risk"`
	Evidence   []MatchEvidenceReference `json:"evidence"`
}

var (
	matcherOutputKeys   = []string{"status", "score", "confidence", "risk", "evidence"}
	matcherEvidenceKeys = []string{"source_table", "source_id", "snippet", "reason"}
)

func (value *matcherOutput) UnmarshalJSON(data []byte) error {
	type plain matcherOutput
	return decodeExactMatcherObject(data, matcherOutputKeys, (*plain)(value))
}

func (value *MatchEvidenceReference) UnmarshalJSON(data []byte) error {
	type plain MatchEvidenceReference
	return decodeExactMatcherObject(data, matcherEvidenceKeys, (*plain)(value))
}

func decodeExactMatcherObject(data []byte, required []string, destination any) error {
	if err := rejectDuplicateMatcherObjectKeys(data); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("%w: expected JSON object", ErrCandidateMatchSchema)
	}
	allowed := make(map[string]struct{}, len(required))
	for _, key := range required {
		allowed[key] = struct{}{}
		raw, exists := fields[key]
		if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%w: missing required field", ErrCandidateMatchSchema)
		}
	}
	for key := range fields {
		if _, exists := allowed[key]; !exists {
			return fmt.Errorf("%w: unknown field", ErrCandidateMatchSchema)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func rejectDuplicateMatcherObjectKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := walkMatcherJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("%w: multiple JSON values", ErrCandidateMatchSchema)
		}
		return err
	}
	return nil
}

func walkMatcherJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("%w: invalid JSON object key", ErrCandidateMatchSchema)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("%w: duplicate JSON object key", ErrCandidateMatchSchema)
			}
			seen[key] = struct{}{}
			if err := walkMatcherJSONValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := walkMatcherJSONValue(decoder); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: invalid JSON delimiter", ErrCandidateMatchSchema)
	}
	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	closingDelimiter, ok := closing.(json.Delim)
	if !ok || (delimiter == '{' && closingDelimiter != '}') || (delimiter == '[' && closingDelimiter != ']') {
		return fmt.Errorf("%w: invalid JSON structure", ErrCandidateMatchSchema)
	}
	return nil
}

func decodeMatcherOutput(content, requirementID string, index EvidenceIndex) (RequirementMatchResult, error) {
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(content)))
	decoder.DisallowUnknownFields()
	var output matcherOutput
	if err := decoder.Decode(&output); err != nil {
		return RequirementMatchResult{}, fmt.Errorf("%w", ErrCandidateMatchSchema)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return RequirementMatchResult{}, fmt.Errorf("%w", ErrCandidateMatchSchema)
	}
	result := RequirementMatchResult{
		RequirementID: requirementID,
		Status:        output.Status, Score: output.Score, Confidence: output.Confidence,
		Risk: output.Risk, Evidence: output.Evidence, EvaluatorType: MatchEvaluatorLLM,
	}
	if err := validateRequirementMatchResult(result, index); err != nil {
		return RequirementMatchResult{}, err
	}
	return result, nil
}

func validateRequirementMatchResult(result RequirementMatchResult, index EvidenceIndex) error {
	switch result.Status {
	case MatchStatusStrongMatch, MatchStatusMatch, MatchStatusPartialMatch, MatchStatusWeakEvidence, MatchStatusMissing, MatchStatusConflict:
	default:
		return fmt.Errorf("%w: invalid status", ErrCandidateMatchSchema)
	}
	if math.IsNaN(result.Score) || math.IsInf(result.Score, 0) || math.IsNaN(result.Confidence) || math.IsInf(result.Confidence, 0) ||
		result.Score < 0 || result.Score > 100 || result.Confidence < 0 || result.Confidence > 1 {
		return fmt.Errorf("%w: numeric range", ErrCandidateMatchSchema)
	}
	if len([]rune(result.Risk)) > MaxCandidateEvidenceRunes || containsCandidateContactData(result.Risk) || (result.Risk != "" && !isMatcherSimplifiedChineseNaturalLanguage(result.Risk)) {
		return fmt.Errorf("%w: unsafe risk", ErrCandidateMatchSchema)
	}
	if (result.Status == MatchStatusMatch || result.Status == MatchStatusStrongMatch) && len(result.Evidence) == 0 {
		return fmt.Errorf("%w: matched result requires evidence", ErrCandidateMatchSchema)
	}
	if len(result.Evidence) > MaxMatchEvidenceReferences {
		return fmt.Errorf("%w: too many evidence references", ErrCandidateMatchSchema)
	}
	seen := make(map[string]struct{}, len(result.Evidence))
	for _, reference := range result.Evidence {
		key := evidenceKey(reference.SourceTable, reference.SourceID)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%w: duplicate evidence reference", ErrCandidateMatchProvenance)
		}
		seen[key] = struct{}{}
		unit, exists := index.lookup(reference.SourceTable, reference.SourceID)
		if !exists || reference.Snippet != unit.Snippet {
			return fmt.Errorf("%w", ErrCandidateMatchProvenance)
		}
		if strings.TrimSpace(reference.Reason) == "" || len([]rune(reference.Reason)) > MaxCandidateEvidenceRunes || containsCandidateContactData(reference.Reason) || !isMatcherSimplifiedChineseNaturalLanguage(reference.Reason) {
			return fmt.Errorf("%w: unsafe evidence reason", ErrCandidateMatchSchema)
		}
	}
	return nil
}

var (
	matcherEnglishWordPattern    = regexp.MustCompile(`[A-Za-z]+`)
	matcherTechnicalEnglishTerms = map[string]struct{}{
		"ai": {}, "api": {}, "aws": {}, "azure": {}, "c": {}, "cloud": {}, "css": {}, "docker": {},
		"dotnet": {}, "elasticsearch": {}, "gcp": {}, "git": {}, "go": {}, "golang": {}, "graphql": {},
		"grpc": {}, "html": {}, "java": {}, "javascript": {}, "js": {},
		"k8s": {}, "kafka": {}, "kubernetes": {}, "linux": {}, "machine": {}, "learning": {}, "mysql": {},
		"node": {}, "postgresql": {}, "python": {}, "react": {}, "redis": {}, "rust": {}, "spark": {},
		"spring": {}, "sql": {}, "terraform": {}, "typescript": {}, "vue": {},
	}
)

// Matcher risk/reason fields are Simplified Chinese natural-language fields,
// not generic display labels. Only explicitly reviewed technical terms may
// remain in English. Case shape is never proof that an unknown token is a
// technical name, and evidence text must not be able to extend this allowlist.
func isMatcherSimplifiedChineseNaturalLanguage(value string) bool {
	if !isConservativeSimplifiedChineseDisplayText(value) {
		return false
	}
	englishWords := matcherEnglishWordPattern.FindAllString(value, -1)
	for _, word := range englishWords {
		lower := strings.ToLower(word)
		if _, technical := matcherTechnicalEnglishTerms[lower]; !technical {
			return false
		}
	}
	return true
}

type CandidateRequirementEvaluator struct {
	runtime *Runtime
	policy  RuntimePolicy
}

func NewCandidateRequirementEvaluator(runtime *Runtime, policy RuntimePolicy) *CandidateRequirementEvaluator {
	return &CandidateRequirementEvaluator{runtime: runtime, policy: policy.effective()}
}

// Evaluate returns stable requirement order. On a fallback-disabled matcher
// failure it also returns one deterministic result for every requirement.
func (e *CandidateRequirementEvaluator) Evaluate(ctx context.Context, profile JobRequirementProfile, index EvidenceIndex) ([]RequirementMatchResult, error) {
	started := time.Now()
	if e == nil || !e.policy.CandidateMatchEnabled() {
		return nil, ErrCandidateMatchPolicy
	}
	profileCopy := JobRequirementProfile{ProfileVersion: profile.ProfileVersion, Requirements: StableJobRequirements(profile)}
	if err := normalizeAndValidateJobRequirementProfile(&profileCopy); err != nil {
		return nil, fmt.Errorf("%w: requirement profile", ErrCandidateMatchSchema)
	}
	operationCtx, cancel := e.policy.CandidateMatchContext(ctx)
	defer cancel()
	results := make([]RequirementMatchResult, len(profileCopy.Requirements))
	for requirementIndex, requirement := range profileCopy.Requirements {
		results[requirementIndex] = deterministicRequirementMatch(requirement, index)
	}
	useMatcher := e.policy.CandidateMatchExecutionPlan().StructuredCompletionEnabled()
	if !useMatcher {
		observeRuntimeOutcome(e.runtime, ctx, "candidate_requirement_evaluation", AgentTypeCandidateMatchEvaluator, "success", "deterministic", started)
		return results, nil
	}
	for resultIndex, requirement := range profileCopy.Requirements {
		if results[resultIndex].Status != MatchStatusMissing {
			continue
		}
		matched, err := e.evaluateUnresolved(operationCtx, requirement, index)
		if err == nil {
			results[resultIndex] = matched
			continue
		}
		if !e.policy.FallbacksEnabled() {
			observeRuntimeOutcome(e.runtime, ctx, "candidate_requirement_evaluation", AgentTypeCandidateMatchEvaluator, "error", "disabled", started)
			return results, err
		}
		results[resultIndex].FallbackUsed = true
	}
	observeRuntimeOutcome(e.runtime, ctx, "candidate_requirement_evaluation", AgentTypeCandidateMatchEvaluator, "success", candidateMatcherFallbackOutcome(results), started)
	return results, nil
}

func candidateMatcherFallbackOutcome(results []RequirementMatchResult) string {
	for _, result := range results {
		if result.FallbackUsed {
			return "deterministic_missing"
		}
	}
	return "none"
}

func (e *CandidateRequirementEvaluator) evaluateUnresolved(ctx context.Context, requirement JobRequirementItem, index EvidenceIndex) (RequirementMatchResult, error) {
	if e.runtime == nil {
		return RequirementMatchResult{}, fmt.Errorf("candidate requirement evaluation failed: %w", ErrProvider)
	}
	userMessage, err := candidateMatcherUserMessage(requirement, index)
	if err != nil {
		return RequirementMatchResult{}, fmt.Errorf("candidate requirement evaluation failed: %w", ErrCandidateMatchSchema)
	}
	completion, err := e.runtime.Complete(ctx, AgentTypeCandidateMatchEvaluator, userMessage)
	if err != nil {
		return RequirementMatchResult{}, err
	}
	result, err := decodeMatcherOutput(completion.Content, requirement.ID, index)
	if err != nil {
		return RequirementMatchResult{}, err
	}
	result.ModelName = completion.ModelName
	result.Prompt = completion.Prompt
	return result, nil
}

func deterministicRequirementMatch(requirement JobRequirementItem, index EvidenceIndex) RequirementMatchResult {
	aliases := append([]string(nil), requirement.Aliases...)
	aliases = append(aliases, requirement.Label)
	sort.SliceStable(aliases, func(i, j int) bool { return strings.ToLower(aliases[i]) < strings.ToLower(aliases[j]) })
	for _, unit := range index.units {
		for _, alias := range aliases {
			if !candidateEvidenceContains(unit.Snippet, alias) {
				continue
			}
			status, score, confidence := MatchStatusMatch, 85.0, 0.88
			if requirement.Priority == RequirementPriorityMustHave {
				status, score, confidence = MatchStatusStrongMatch, 95, 0.94
			}
			result := RequirementMatchResult{
				RequirementID: requirement.ID, Status: status, Score: score, Confidence: confidence,
				Risk: "", EvaluatorType: MatchEvaluatorDeterministic,
				Evidence: []MatchEvidenceReference{{SourceTable: unit.SourceTable, SourceID: unit.SourceID, Snippet: unit.Snippet, Reason: "候选人证据直接命中岗位要求关键词"}},
			}
			return result
		}
	}
	return RequirementMatchResult{
		RequirementID: requirement.ID, Status: MatchStatusMissing, Score: 0, Confidence: 0.5,
		Risk: "未找到可直接验证该岗位要求的候选人证据", Evidence: []MatchEvidenceReference{}, EvaluatorType: MatchEvaluatorDeterministic,
	}
}

// EvaluateRequirementsDeterministically evaluates every validated
// requirement without invoking the structured matcher. The stable traversal
// is shared with the enhanced evaluator so fixed inputs remain reproducible.
func EvaluateRequirementsDeterministically(profile JobRequirementProfile, index EvidenceIndex) ([]RequirementMatchResult, error) {
	profileCopy := JobRequirementProfile{ProfileVersion: profile.ProfileVersion, Requirements: StableJobRequirements(profile)}
	if err := normalizeAndValidateJobRequirementProfile(&profileCopy); err != nil {
		return nil, fmt.Errorf("%w: requirement profile", ErrCandidateMatchSchema)
	}
	results := make([]RequirementMatchResult, 0, len(profileCopy.Requirements))
	for _, requirement := range profileCopy.Requirements {
		results = append(results, deterministicRequirementMatch(requirement, index))
	}
	return results, nil
}

func candidateEvidenceContains(snippet, alias string) bool {
	alias = normalizeCandidateEvidenceText(alias)
	if alias == "" {
		return false
	}
	lowerSnippet, lowerAlias := strings.ToLower(candidateEvidenceSearchableText(snippet)), strings.ToLower(alias)
	if containsNonASCII(lowerAlias) {
		return strings.Contains(lowerSnippet, lowerAlias)
	}
	return containsRequirementTerm(lowerSnippet, lowerAlias)
}

func candidateMatcherUserMessage(requirement JobRequirementItem, index EvidenceIndex) (string, error) {
	type matcherRequirement struct {
		ID          string   `json:"id"`
		Category    string   `json:"category"`
		Label       string   `json:"label"`
		Description string   `json:"description"`
		Priority    string   `json:"priority"`
		Knockout    bool     `json:"knockout"`
		Aliases     []string `json:"aliases"`
	}
	payload := struct {
		Requirement matcherRequirement `json:"requirement"`
		Evidence    []EvidenceUnit     `json:"candidate_evidence"`
	}{
		Requirement: matcherRequirement{requirement.ID, requirement.Category, requirement.Label, requirement.Description, requirement.Priority, requirement.Knockout, append([]string(nil), requirement.Aliases...)},
		Evidence:    index.Units(),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
