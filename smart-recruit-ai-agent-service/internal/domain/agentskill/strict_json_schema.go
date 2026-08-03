package agentskill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	strictJSONSchemaDialect = "https://json-schema.org/draft/2020-12/schema"
	maxStrictJSONDepth      = 128
	maxStrictJSONCount      = 1_000_000
	maxStrictNumberExponent = 10_000
)

var strictJSONSchemaKeywords = map[string]struct{}{
	"$schema": {}, "title": {}, "description": {}, "default": {}, "examples": {},
	"type": {}, "properties": {}, "required": {}, "additionalProperties": {}, "items": {},
	"enum": {}, "const": {}, "allOf": {}, "anyOf": {}, "oneOf": {}, "not": {},
	"minProperties": {}, "maxProperties": {}, "minItems": {}, "maxItems": {},
	"uniqueItems": {}, "minLength": {}, "maxLength": {}, "pattern": {},
	"minimum": {}, "maximum": {}, "exclusiveMinimum": {}, "exclusiveMaximum": {},
	"multipleOf": {},
}

type strictJSONSchemaError struct {
	stage     string
	category  string
	count     int
	pathDepth int
}

func (e *strictJSONSchemaError) Error() string {
	if e == nil {
		return ""
	}
	return strictJSONSummary(e.stage, e.category, e.count, e.pathDepth)
}

func newStrictJSONSchemaError(stage, category string, count, pathDepth int) error {
	return &strictJSONSchemaError{
		stage:     boundedStrictLabel(stage, "unknown"),
		category:  boundedStrictLabel(category, "unknown"),
		count:     boundedStrictCount(count),
		pathDepth: boundedStrictDepth(pathDepth),
	}
}

func strictJSONSummary(stage, category string, count, pathDepth int) string {
	return fmt.Sprintf(
		"stage=%s category=%s count=%d path_depth=%d",
		boundedStrictLabel(stage, "unknown"),
		boundedStrictLabel(category, "unknown"),
		boundedStrictCount(count),
		boundedStrictDepth(pathDepth),
	)
}

func boundedStrictLabel(value, fallback string) string {
	switch value {
	case "compile", "parse", "validation":
		return value
	case "valid", "invalid_json", "duplicate_key", "trailing_value", "depth_limit",
		"root_shape", "unknown_keyword", "invalid_keyword_shape", "unsupported_dialect",
		"unsupported_reference", "invalid_number", "schema_mismatch":
		return value
	default:
		return fallback
	}
}

func boundedStrictCount(value int) int {
	if value < 0 {
		return 0
	}
	if value > maxStrictJSONCount {
		return maxStrictJSONCount
	}
	return value
}

func boundedStrictDepth(value int) int {
	if value < 0 {
		return 0
	}
	if value > maxStrictJSONDepth {
		return maxStrictJSONDepth
	}
	return value
}

// StrictOutputSchema is an immutable, concurrency-safe compiled form of the
// fail-closed JSON Schema dialect accepted for strict Agent Skill output.
type StrictOutputSchema struct {
	root *strictJSONSchemaNode
}

type strictJSONSchemaNode struct {
	hasType bool
	types   strictJSONType

	properties           map[string]*strictJSONSchemaNode
	required             []string
	hasAdditional        bool
	allowAdditional      bool
	additionalProperties *strictJSONSchemaNode
	items                *strictJSONSchemaNode

	enumValues []any
	hasConst   bool
	constValue any
	allOf      []*strictJSONSchemaNode
	anyOf      []*strictJSONSchemaNode
	oneOf      []*strictJSONSchemaNode
	not        *strictJSONSchemaNode

	minProperties *int
	maxProperties *int
	minItems      *int
	maxItems      *int
	uniqueItems   bool
	minLength     *int
	maxLength     *int
	pattern       *regexp.Regexp

	minimum          *big.Rat
	maximum          *big.Rat
	exclusiveMinimum *big.Rat
	exclusiveMaximum *big.Rat
	multipleOf       *big.Rat
}

type strictJSONType uint8

const (
	strictJSONTypeNull strictJSONType = 1 << iota
	strictJSONTypeBoolean
	strictJSONTypeObject
	strictJSONTypeArray
	strictJSONTypeNumber
	strictJSONTypeInteger
	strictJSONTypeString
)

// CompileStrictOutputSchema rejects unsupported dialect features and unknown
// keywords instead of silently weakening their validation semantics.
func CompileStrictOutputSchema(raw []byte) (*StrictOutputSchema, error) {
	_, schema, err := compileStrictOutputSchemaCanonical(raw)
	return schema, err
}

// compileStrictOutputSchemaCanonical parses the original bytes exactly once so
// duplicate keys cannot be hidden by encoding/json's map decoding before the
// immutable validator is compiled. The returned JSON is the canonical schema
// that must be persisted in the manifest.
func compileStrictOutputSchemaCanonical(raw []byte) (json.RawMessage, *StrictOutputSchema, error) {
	value, err := decodeStrictJSON(raw, "compile")
	if err != nil {
		return nil, nil, err
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, nil, newStrictJSONSchemaError("compile", "root_shape", 1, 0)
	}
	root, err := compileStrictJSONSchemaNode(value, 0, true)
	if err != nil {
		return nil, nil, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, nil, newStrictJSONSchemaError("compile", "invalid_json", 1, 0)
	}
	return canonical, &StrictOutputSchema{root: root}, nil
}

// ValidateJSON requires exactly one JSON value, returns its canonical encoding
// on success, and exposes only bounded classification data on failure.
func (s *StrictOutputSchema) ValidateJSON(content string) (canonical string, privacySafeSummary string, err error) {
	if s == nil || s.root == nil {
		err = newStrictJSONSchemaError("validation", "invalid_keyword_shape", 1, 0)
		return "", err.Error(), err
	}
	value, parseErr := decodeStrictJSON([]byte(content), "parse")
	if parseErr != nil {
		return "", parseErr.Error(), parseErr
	}
	stats := validationStats{}
	s.root.validate(value, 0, &stats)
	if stats.violations > 0 {
		err = newStrictJSONSchemaError("validation", "schema_mismatch", stats.violations, stats.maxDepth)
		return "", err.Error(), err
	}
	encoded, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		err = newStrictJSONSchemaError("validation", "invalid_json", 1, stats.maxDepth)
		return "", err.Error(), err
	}
	return string(encoded), strictJSONSummary("validation", "valid", 0, stats.maxDepth), nil
}

func compileStrictJSONSchemaNode(value any, depth int, root bool) (*strictJSONSchemaNode, error) {
	if depth > maxStrictJSONDepth {
		return nil, newStrictJSONSchemaError("compile", "depth_limit", 1, depth)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "$ref" || key == "$dynamicRef" || key == "$recursiveRef" {
			return nil, newStrictJSONSchemaError("compile", "unsupported_reference", 1, depth)
		}
		if _, supported := strictJSONSchemaKeywords[key]; !supported {
			return nil, newStrictJSONSchemaError("compile", "unknown_keyword", 1, depth)
		}
	}

	node := &strictJSONSchemaNode{}
	if dialect, exists := object["$schema"]; exists {
		dialectString, ok := dialect.(string)
		if !root || !ok || dialectString != strictJSONSchemaDialect {
			return nil, newStrictJSONSchemaError("compile", "unsupported_dialect", 1, depth)
		}
	}
	for _, keyword := range []string{"title", "description"} {
		if annotation, exists := object[keyword]; exists {
			if _, ok := annotation.(string); !ok {
				return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
			}
		}
	}
	if examples, exists := object["examples"]; exists {
		if _, ok := examples.([]any); !ok {
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
	}

	if typeValue, exists := object["type"]; exists {
		types, err := compileStrictJSONTypes(typeValue, depth)
		if err != nil {
			return nil, err
		}
		node.hasType = true
		node.types = types
	}
	if propertiesValue, exists := object["properties"]; exists {
		properties, ok := propertiesValue.(map[string]any)
		if !ok {
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
		node.properties = make(map[string]*strictJSONSchemaNode, len(properties))
		propertyNames := make([]string, 0, len(properties))
		for name := range properties {
			propertyNames = append(propertyNames, name)
		}
		sort.Strings(propertyNames)
		for _, name := range propertyNames {
			propertySchema, err := compileStrictJSONSchemaNode(properties[name], depth+1, false)
			if err != nil {
				return nil, err
			}
			node.properties[name] = propertySchema
		}
	}
	if requiredValue, exists := object["required"]; exists {
		required, err := compileStrictStringArray(requiredValue, depth, true)
		if err != nil {
			return nil, err
		}
		node.required = required
	}
	if additionalValue, exists := object["additionalProperties"]; exists {
		node.hasAdditional = true
		switch typed := additionalValue.(type) {
		case bool:
			node.allowAdditional = typed
		case map[string]any:
			additionalSchema, err := compileStrictJSONSchemaNode(typed, depth+1, false)
			if err != nil {
				return nil, err
			}
			node.allowAdditional = true
			node.additionalProperties = additionalSchema
		default:
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
	}
	if itemsValue, exists := object["items"]; exists {
		items, err := compileStrictJSONSchemaNode(itemsValue, depth+1, false)
		if err != nil {
			return nil, err
		}
		node.items = items
	}

	if enumValue, exists := object["enum"]; exists {
		values, ok := enumValue.([]any)
		if !ok || len(values) == 0 || hasStrictJSONDuplicates(values) {
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
		node.enumValues = values
	}
	if constValue, exists := object["const"]; exists {
		node.hasConst = true
		node.constValue = constValue
	}
	var err error
	if node.allOf, err = compileStrictJSONSchemaArray(object, "allOf", depth); err != nil {
		return nil, err
	}
	if node.anyOf, err = compileStrictJSONSchemaArray(object, "anyOf", depth); err != nil {
		return nil, err
	}
	if node.oneOf, err = compileStrictJSONSchemaArray(object, "oneOf", depth); err != nil {
		return nil, err
	}
	if notValue, exists := object["not"]; exists {
		node.not, err = compileStrictJSONSchemaNode(notValue, depth+1, false)
		if err != nil {
			return nil, err
		}
	}

	if node.minProperties, err = compileStrictNonNegativeInteger(object, "minProperties", depth); err != nil {
		return nil, err
	}
	if node.maxProperties, err = compileStrictNonNegativeInteger(object, "maxProperties", depth); err != nil {
		return nil, err
	}
	if node.minItems, err = compileStrictNonNegativeInteger(object, "minItems", depth); err != nil {
		return nil, err
	}
	if node.maxItems, err = compileStrictNonNegativeInteger(object, "maxItems", depth); err != nil {
		return nil, err
	}
	if uniqueValue, exists := object["uniqueItems"]; exists {
		unique, ok := uniqueValue.(bool)
		if !ok {
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
		node.uniqueItems = unique
	}
	if node.minLength, err = compileStrictNonNegativeInteger(object, "minLength", depth); err != nil {
		return nil, err
	}
	if node.maxLength, err = compileStrictNonNegativeInteger(object, "maxLength", depth); err != nil {
		return nil, err
	}
	if patternValue, exists := object["pattern"]; exists {
		patternString, ok := patternValue.(string)
		if !ok {
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
		node.pattern, err = regexp.Compile(patternString)
		if err != nil {
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
	}

	if node.minimum, err = compileStrictNumber(object, "minimum", depth, false); err != nil {
		return nil, err
	}
	if node.maximum, err = compileStrictNumber(object, "maximum", depth, false); err != nil {
		return nil, err
	}
	if node.exclusiveMinimum, err = compileStrictNumber(object, "exclusiveMinimum", depth, false); err != nil {
		return nil, err
	}
	if node.exclusiveMaximum, err = compileStrictNumber(object, "exclusiveMaximum", depth, false); err != nil {
		return nil, err
	}
	if node.multipleOf, err = compileStrictNumber(object, "multipleOf", depth, true); err != nil {
		return nil, err
	}
	return node, nil
}

func compileStrictJSONTypes(value any, depth int) (strictJSONType, error) {
	var names []string
	switch typed := value.(type) {
	case string:
		names = []string{typed}
	case []any:
		if len(typed) == 0 {
			return 0, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
		seen := make(map[string]struct{}, len(typed))
		for _, entry := range typed {
			name, ok := entry.(string)
			if !ok {
				return 0, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
			}
			if _, exists := seen[name]; exists {
				return 0, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	default:
		return 0, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	var types strictJSONType
	for _, name := range names {
		switch name {
		case "null":
			types |= strictJSONTypeNull
		case "boolean":
			types |= strictJSONTypeBoolean
		case "object":
			types |= strictJSONTypeObject
		case "array":
			types |= strictJSONTypeArray
		case "number":
			types |= strictJSONTypeNumber
		case "integer":
			types |= strictJSONTypeInteger
		case "string":
			types |= strictJSONTypeString
		default:
			return 0, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
	}
	return types, nil
}

func compileStrictStringArray(value any, depth int, allowEmpty bool) ([]string, error) {
	array, ok := value.([]any)
	if !ok || (!allowEmpty && len(array) == 0) {
		return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	output := make([]string, 0, len(array))
	seen := make(map[string]struct{}, len(array))
	for _, entry := range array {
		item, ok := entry.(string)
		if !ok {
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
		if _, exists := seen[item]; exists {
			return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
		}
		seen[item] = struct{}{}
		output = append(output, item)
	}
	return output, nil
}

func compileStrictJSONSchemaArray(object map[string]any, keyword string, depth int) ([]*strictJSONSchemaNode, error) {
	value, exists := object[keyword]
	if !exists {
		return nil, nil
	}
	array, ok := value.([]any)
	if !ok || len(array) == 0 {
		return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	output := make([]*strictJSONSchemaNode, 0, len(array))
	for _, entry := range array {
		schema, err := compileStrictJSONSchemaNode(entry, depth+1, false)
		if err != nil {
			return nil, err
		}
		output = append(output, schema)
	}
	return output, nil
}

func compileStrictNonNegativeInteger(object map[string]any, keyword string, depth int) (*int, error) {
	value, exists := object[keyword]
	if !exists {
		return nil, nil
	}
	number, ok := value.(json.Number)
	if !ok {
		return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	rational, ok := strictJSONNumberRat(number)
	if !ok {
		return nil, newStrictJSONSchemaError("compile", "invalid_number", 1, depth)
	}
	if rational.Sign() < 0 || !rational.IsInt() || !rational.Num().IsInt64() {
		return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	parsed := rational.Num().Int64()
	if strconv.IntSize == 32 && parsed > int64(^uint32(0)>>1) {
		return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	output := int(parsed)
	return &output, nil
}

func compileStrictNumber(object map[string]any, keyword string, depth int, positive bool) (*big.Rat, error) {
	value, exists := object[keyword]
	if !exists {
		return nil, nil
	}
	number, ok := value.(json.Number)
	if !ok {
		return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	rational, ok := strictJSONNumberRat(number)
	if !ok {
		return nil, newStrictJSONSchemaError("compile", "invalid_number", 1, depth)
	}
	if positive && rational.Sign() <= 0 {
		return nil, newStrictJSONSchemaError("compile", "invalid_keyword_shape", 1, depth)
	}
	return rational, nil
}

type validationStats struct {
	violations int
	maxDepth   int
}

func (s *validationStats) visit(depth int) {
	if depth > s.maxDepth {
		s.maxDepth = boundedStrictDepth(depth)
	}
}

func (s *validationStats) addViolation(depth int) {
	s.visit(depth)
	if s.violations < maxStrictJSONCount {
		s.violations++
	}
}

func (n *strictJSONSchemaNode) validate(value any, depth int, stats *validationStats) {
	stats.visit(depth)
	if n.hasType && !n.types.matches(value) {
		stats.addViolation(depth)
	}
	if len(n.enumValues) > 0 {
		matched := false
		for _, candidate := range n.enumValues {
			if strictJSONEqual(value, candidate) {
				matched = true
			}
		}
		if !matched {
			stats.addViolation(depth)
		}
	}
	if n.hasConst && !strictJSONEqual(value, n.constValue) {
		stats.addViolation(depth)
	}

	for _, schema := range n.allOf {
		schema.validate(value, depth, stats)
	}
	if len(n.anyOf) > 0 {
		matched := false
		branchDepth := depth
		for _, schema := range n.anyOf {
			branch := validationStats{}
			schema.validate(value, depth, &branch)
			if branch.violations == 0 {
				matched = true
			}
			if branch.maxDepth > branchDepth {
				branchDepth = branch.maxDepth
			}
		}
		stats.visit(branchDepth)
		if !matched {
			stats.addViolation(branchDepth)
		}
	}
	if len(n.oneOf) > 0 {
		matches := 0
		branchDepth := depth
		for _, schema := range n.oneOf {
			branch := validationStats{}
			schema.validate(value, depth, &branch)
			if branch.violations == 0 {
				matches++
			}
			if branch.maxDepth > branchDepth {
				branchDepth = branch.maxDepth
			}
		}
		stats.visit(branchDepth)
		if matches != 1 {
			stats.addViolation(branchDepth)
		}
	}
	if n.not != nil {
		branch := validationStats{}
		n.not.validate(value, depth, &branch)
		stats.visit(branch.maxDepth)
		if branch.violations == 0 {
			stats.addViolation(branch.maxDepth)
		}
	}

	switch typed := value.(type) {
	case map[string]any:
		n.validateObject(typed, depth, stats)
	case []any:
		n.validateArray(typed, depth, stats)
	case string:
		n.validateString(typed, depth, stats)
	case json.Number:
		n.validateNumber(typed, depth, stats)
	}
}

func (n *strictJSONSchemaNode) validateObject(value map[string]any, depth int, stats *validationStats) {
	if n.minProperties != nil && len(value) < *n.minProperties {
		stats.addViolation(depth)
	}
	if n.maxProperties != nil && len(value) > *n.maxProperties {
		stats.addViolation(depth)
	}
	for _, required := range n.required {
		if _, exists := value[required]; !exists {
			stats.addViolation(depth + 1)
		}
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		propertyValue := value[key]
		if propertySchema, exists := n.properties[key]; exists {
			propertySchema.validate(propertyValue, depth+1, stats)
			continue
		}
		if !n.hasAdditional {
			continue
		}
		if !n.allowAdditional {
			stats.addViolation(depth + 1)
			continue
		}
		if n.additionalProperties != nil {
			n.additionalProperties.validate(propertyValue, depth+1, stats)
		}
	}
}

func (n *strictJSONSchemaNode) validateArray(value []any, depth int, stats *validationStats) {
	if n.minItems != nil && len(value) < *n.minItems {
		stats.addViolation(depth)
	}
	if n.maxItems != nil && len(value) > *n.maxItems {
		stats.addViolation(depth)
	}
	if n.uniqueItems && hasStrictJSONDuplicates(value) {
		stats.addViolation(depth + 1)
	}
	if n.items != nil {
		for _, item := range value {
			n.items.validate(item, depth+1, stats)
		}
	}
}

func (n *strictJSONSchemaNode) validateString(value string, depth int, stats *validationStats) {
	length := utf8.RuneCountInString(value)
	if n.minLength != nil && length < *n.minLength {
		stats.addViolation(depth)
	}
	if n.maxLength != nil && length > *n.maxLength {
		stats.addViolation(depth)
	}
	if n.pattern != nil && !n.pattern.MatchString(value) {
		stats.addViolation(depth)
	}
}

func (n *strictJSONSchemaNode) validateNumber(value json.Number, depth int, stats *validationStats) {
	number, ok := strictJSONNumberRat(value)
	if !ok {
		stats.addViolation(depth)
		return
	}
	if n.minimum != nil && number.Cmp(n.minimum) < 0 {
		stats.addViolation(depth)
	}
	if n.maximum != nil && number.Cmp(n.maximum) > 0 {
		stats.addViolation(depth)
	}
	if n.exclusiveMinimum != nil && number.Cmp(n.exclusiveMinimum) <= 0 {
		stats.addViolation(depth)
	}
	if n.exclusiveMaximum != nil && number.Cmp(n.exclusiveMaximum) >= 0 {
		stats.addViolation(depth)
	}
	if n.multipleOf != nil {
		quotient := new(big.Rat).Quo(number, n.multipleOf)
		if !quotient.IsInt() {
			stats.addViolation(depth)
		}
	}
}

func (t strictJSONType) matches(value any) bool {
	switch typed := value.(type) {
	case nil:
		return t&strictJSONTypeNull != 0
	case bool:
		return t&strictJSONTypeBoolean != 0
	case map[string]any:
		return t&strictJSONTypeObject != 0
	case []any:
		return t&strictJSONTypeArray != 0
	case string:
		return t&strictJSONTypeString != 0
	case json.Number:
		if t&strictJSONTypeNumber != 0 {
			return true
		}
		if t&strictJSONTypeInteger == 0 {
			return false
		}
		rational, ok := strictJSONNumberRat(typed)
		return ok && rational.IsInt()
	default:
		return false
	}
}

func decodeStrictJSON(raw []byte, stage string) (any, error) {
	if len(bytes.TrimSpace(raw)) == 0 || !utf8.Valid(raw) {
		return nil, newStrictJSONSchemaError(stage, "invalid_json", 1, 0)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := decodeStrictJSONValue(decoder, stage, 0)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, newStrictJSONSchemaError(stage, "trailing_value", 1, 0)
		}
		return nil, newStrictJSONSchemaError(stage, "invalid_json", 1, 0)
	}
	return value, nil
}

func decodeStrictJSONValue(decoder *json.Decoder, stage string, depth int) (any, error) {
	if depth > maxStrictJSONDepth {
		return nil, newStrictJSONSchemaError(stage, "depth_limit", 1, depth)
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, newStrictJSONSchemaError(stage, "invalid_json", 1, depth)
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		if number, ok := token.(json.Number); ok {
			if _, valid := strictJSONNumberParts(number.String()); !valid {
				return nil, newStrictJSONSchemaError(stage, "invalid_number", 1, depth)
			}
		}
		return token, nil
	}
	switch delim {
	case '{':
		object := make(map[string]any)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, newStrictJSONSchemaError(stage, "invalid_json", 1, depth)
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, newStrictJSONSchemaError(stage, "invalid_json", 1, depth)
			}
			if _, exists := object[key]; exists {
				return nil, newStrictJSONSchemaError(stage, "duplicate_key", 1, depth+1)
			}
			child, err := decodeStrictJSONValue(decoder, stage, depth+1)
			if err != nil {
				return nil, err
			}
			object[key] = child
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return nil, newStrictJSONSchemaError(stage, "invalid_json", 1, depth)
		}
		return object, nil
	case '[':
		array := make([]any, 0)
		for decoder.More() {
			child, err := decodeStrictJSONValue(decoder, stage, depth+1)
			if err != nil {
				return nil, err
			}
			array = append(array, child)
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return nil, newStrictJSONSchemaError(stage, "invalid_json", 1, depth)
		}
		return array, nil
	default:
		return nil, newStrictJSONSchemaError(stage, "invalid_json", 1, depth)
	}
}

type strictJSONNumber struct {
	text     string
	exponent int
}

func strictJSONNumberParts(value string) (strictJSONNumber, bool) {
	exponent := 0
	if index := strings.IndexAny(value, "eE"); index >= 0 {
		parsed, err := strconv.ParseInt(value[index+1:], 10, 32)
		if err != nil || parsed > maxStrictNumberExponent || parsed < -maxStrictNumberExponent {
			return strictJSONNumber{}, false
		}
		exponent = int(parsed)
	}
	return strictJSONNumber{text: value, exponent: exponent}, true
}

func strictJSONNumberRat(value json.Number) (*big.Rat, bool) {
	parts, ok := strictJSONNumberParts(value.String())
	if !ok {
		return nil, false
	}
	rational, ok := new(big.Rat).SetString(parts.text)
	return rational, ok
}

func strictJSONEqual(left, right any) bool {
	switch leftValue := left.(type) {
	case nil:
		return right == nil
	case bool:
		rightValue, ok := right.(bool)
		return ok && leftValue == rightValue
	case string:
		rightValue, ok := right.(string)
		return ok && leftValue == rightValue
	case json.Number:
		rightValue, ok := right.(json.Number)
		if !ok {
			return false
		}
		leftNumber, leftOK := strictJSONNumberRat(leftValue)
		rightNumber, rightOK := strictJSONNumberRat(rightValue)
		return leftOK && rightOK && leftNumber.Cmp(rightNumber) == 0
	case []any:
		rightValue, ok := right.([]any)
		if !ok || len(leftValue) != len(rightValue) {
			return false
		}
		for index := range leftValue {
			if !strictJSONEqual(leftValue[index], rightValue[index]) {
				return false
			}
		}
		return true
	case map[string]any:
		rightValue, ok := right.(map[string]any)
		if !ok || len(leftValue) != len(rightValue) {
			return false
		}
		for key, child := range leftValue {
			rightChild, exists := rightValue[key]
			if !exists || !strictJSONEqual(child, rightChild) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func hasStrictJSONDuplicates(values []any) bool {
	for left := 0; left < len(values); left++ {
		for right := left + 1; right < len(values); right++ {
			if strictJSONEqual(values[left], values[right]) {
				return true
			}
		}
	}
	return false
}
