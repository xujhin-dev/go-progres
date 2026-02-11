package security

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Validator 验证器接口
type Validator interface {
	Validate(value interface{}) error
	Sanitize(value string) string
}

// StringValidator 字符串验证器
type StringValidator struct {
	MinLength int
	MaxLength int
	Required  bool
	Pattern   *regexp.Regexp
	AllowHTML bool
}

// NewStringValidator 创建字符串验证器
func NewStringValidator(minLength, maxLength int, required bool) *StringValidator {
	return &StringValidator{
		MinLength: minLength,
		MaxLength: maxLength,
		Required:  required,
		AllowHTML: false,
	}
}

// Validate 验证字符串
func (sv *StringValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if sv.Required && str == "" {
		return fmt.Errorf("value is required")
	}

	if !sv.Required && str == "" {
		return nil
	}

	length := utf8.RuneCountInString(str)
	if length < sv.MinLength {
		return fmt.Errorf("value too short, minimum length is %d", sv.MinLength)
	}

	if length > sv.MaxLength {
		return fmt.Errorf("value too long, maximum length is %d", sv.MaxLength)
	}

	if sv.Pattern != nil && !sv.Pattern.MatchString(str) {
		return fmt.Errorf("value does not match required pattern")
	}

	return nil
}

// Sanitize 清理字符串
func (sv *StringValidator) Sanitize(value string) string {
	if !sv.AllowHTML {
		value = html.EscapeString(value)
	}

	// 移除控制字符
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, value)

	// 标准化空白字符
	value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
	value = strings.TrimSpace(value)

	return value
}

// SetPattern 设置正则表达式模式
func (sv *StringValidator) SetPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	sv.Pattern = re
	return nil
}

// EmailValidator 邮箱验证器
type EmailValidator struct {
	Required bool
}

// NewEmailValidator 创建邮箱验证器
func NewEmailValidator(required bool) *EmailValidator {
	return &EmailValidator{Required: required}
}

// Validate 验证邮箱
func (ev *EmailValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if ev.Required && str == "" {
		return fmt.Errorf("email is required")
	}

	if !ev.Required && str == "" {
		return nil
	}

	// 简单的邮箱验证正则
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// Sanitize 清理邮箱
func (ev *EmailValidator) Sanitize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return value
}

// PhoneValidator 手机号验证器
type PhoneValidator struct {
	Required bool
	Country  string // 国家代码，如 "CN", "US"
}

// NewPhoneValidator 创建手机号验证器
func NewPhoneValidator(required bool, country string) *PhoneValidator {
	return &PhoneValidator{
		Required: required,
		Country:  country,
	}
}

// Validate 验证手机号
func (pv *PhoneValidator) Validate(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if pv.Required && str == "" {
		return fmt.Errorf("phone number is required")
	}

	if !pv.Required && str == "" {
		return nil
	}

	// 移除所有非数字字符
	digits := regexp.MustCompile(`[^\d]`).ReplaceAllString(str, "")

	switch pv.Country {
	case "CN":
		// 中国手机号：11位，以1开头
		if len(digits) != 11 || digits[0] != '1' {
			return fmt.Errorf("invalid Chinese phone number")
		}
	case "US":
		// 美国手机号：10位
		if len(digits) != 10 {
			return fmt.Errorf("invalid US phone number")
		}
	default:
		// 通用验证：至少10位数字
		if len(digits) < 10 {
			return fmt.Errorf("invalid phone number")
		}
	}

	return nil
}

// Sanitize 清理手机号
func (pv *PhoneValidator) Sanitize(value string) string {
	// 移除所有非数字字符
	digits := regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
	return digits
}

// NumberValidator 数字验证器
type NumberValidator struct {
	Min      *float64
	Max      *float64
	Required bool
	Integer  bool
}

// NewNumberValidator 创建数字验证器
func NewNumberValidator(required bool) *NumberValidator {
	return &NumberValidator{Required: required}
}

// Validate 验证数字
func (nv *NumberValidator) Validate(value interface{}) error {
	var num float64
	var err error

	switch v := value.(type) {
	case string:
		num, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("value must be a number")
		}
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case float64:
		num = v
	case float32:
		num = float64(v)
	default:
		return fmt.Errorf("value must be a number")
	}

	if nv.Integer && num != float64(int(num)) {
		return fmt.Errorf("value must be an integer")
	}

	if nv.Min != nil && num < *nv.Min {
		return fmt.Errorf("value must be at least %f", *nv.Min)
	}

	if nv.Max != nil && num > *nv.Max {
		return fmt.Errorf("value must be at most %f", *nv.Max)
	}

	return nil
}

// Sanitize 清理数字
func (nv *NumberValidator) Sanitize(value string) string {
	return strings.TrimSpace(value)
}

// SetMin 设置最小值
func (nv *NumberValidator) SetMin(min float64) {
	nv.Min = &min
}

// SetMax 设置最大值
func (nv *NumberValidator) SetMax(max float64) {
	nv.Max = &max
}

// SetInteger 设置是否为整数
func (nv *NumberValidator) SetInteger(integer bool) {
	nv.Integer = integer
}

// ArrayValidator 数组验证器
type ArrayValidator struct {
	MinLength int
	MaxLength int
	Required  bool
	ItemValidator Validator
}

// NewArrayValidator 创建数组验证器
func NewArrayValidator(minLength, maxLength int, required bool) *ArrayValidator {
	return &ArrayValidator{
		MinLength: minLength,
		MaxLength: maxLength,
		Required:  required,
	}
}

// Validate 验证数组
func (av *ArrayValidator) Validate(value interface{}) error {
	switch v := value.(type) {
	case []interface{}:
		if av.Required && len(v) == 0 {
			return fmt.Errorf("array is required")
		}

		if len(v) < av.MinLength {
			return fmt.Errorf("array too short, minimum length is %d", av.MinLength)
		}

		if av.MaxLength > 0 && len(v) > av.MaxLength {
			return fmt.Errorf("array too long, maximum length is %d", av.MaxLength)
		}

		// 验证每个元素
		if av.ItemValidator != nil {
			for i, item := range v {
				if err := av.ItemValidator.Validate(item); err != nil {
					return fmt.Errorf("item at index %d: %w", i, err)
				}
			}
		}

	case []string:
		if av.Required && len(v) == 0 {
			return fmt.Errorf("array is required")
		}

		if len(v) < av.MinLength {
			return fmt.Errorf("array too short, minimum length is %d", av.MinLength)
		}

		if av.MaxLength > 0 && len(v) > av.MaxLength {
			return fmt.Errorf("array too long, maximum length is %d", av.MaxLength)
		}

		// 验证每个元素
		if av.ItemValidator != nil {
			for i, item := range v {
				if err := av.ItemValidator.Validate(item); err != nil {
					return fmt.Errorf("item at index %d: %w", i, err)
				}
			}
		}

	default:
		return fmt.Errorf("value must be an array")
	}

	return nil
}

// Sanitize 清理数组
func (av *ArrayValidator) Sanitize(value string) string {
	return value
}

// SetItemValidator 设置元素验证器
func (av *ArrayValidator) SetItemValidator(validator Validator) {
	av.ItemValidator = validator
}

// ValidationRule 验证规则
type ValidationRule struct {
	Field     string
	Validator Validator
	Message   string
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool
	Errors map[string]string
}

// NewValidationResult 创建验证结果
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:  true,
		Errors: make(map[string]string),
	}
}

// AddError 添加错误
func (vr *ValidationResult) AddError(field, message string) {
	vr.Valid = false
	vr.Errors[field] = message
}

// ValidatorSet 验证器集合
type ValidatorSet struct {
	rules []ValidationRule
}

// NewValidatorSet 创建验证器集合
func NewValidatorSet() *ValidatorSet {
	return &ValidatorSet{
		rules: make([]ValidationRule, 0),
	}
}

// AddRule 添加验证规则
func (vs *ValidatorSet) AddRule(field string, validator Validator, message string) {
	vs.rules = append(vs.rules, ValidationRule{
		Field:     field,
		Validator: validator,
		Message:   message,
	})
}

// Validate 验证数据
func (vs *ValidatorSet) Validate(data map[string]interface{}) *ValidationResult {
	result := NewValidationResult()

	for _, rule := range vs.rules {
		value, exists := data[rule.Field]
		if !exists {
			if sv, ok := rule.Validator.(*StringValidator); ok && sv.Required {
				result.AddError(rule.Field, rule.Message)
			}
			continue
		}

		if err := rule.Validator.Validate(value); err != nil {
			result.AddError(rule.Field, rule.Message)
		}
	}

	return result
}

// SanitizeData 清理数据
func (vs *ValidatorSet) SanitizeData(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for _, rule := range vs.rules {
		value, exists := data[rule.Field]
		if !exists {
			continue
		}

		if str, ok := value.(string); ok {
			sanitized[rule.Field] = rule.Validator.Sanitize(str)
		} else {
			sanitized[rule.Field] = value
		}
	}

	// 添加未定义的字段
	for field, value := range data {
		if _, exists := sanitized[field]; !exists {
			sanitized[field] = value
		}
	}

	return sanitized
}

// XSSProtection XSS 防护
type XSSProtection struct {
	allowedTags    map[string]bool
	allowedAttrs   map[string]map[string]bool
	removeComments bool
}

// NewXSSProtection 创建 XSS 防护
func NewXSSProtection() *XSSProtection {
	return &XSSProtection{
		allowedTags: map[string]bool{
			"b": true, "i": true, "u": true, "em": true, "strong": true,
			"p": true, "br": true, "div": true, "span": true,
		},
		allowedAttrs: map[string]map[string]bool{
			"a": {"href": true, "title": true},
			"img": {"src": true, "alt": true, "title": true},
		},
		removeComments: true,
	}
}

// SanitizeHTML 清理 HTML
func (xss *XSSProtection) SanitizeHTML(html string) string {
	// 移除脚本标签
	html = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")
	
	// 移除危险属性
	dangerousAttrs := []string{"onload", "onerror", "onclick", "onmouseover", "onfocus", "onblur"}
	for _, attr := range dangerousAttrs {
		html = regexp.MustCompile(`(?i)\s+`+attr+`\s*=\s*["'][^"']*["']`).ReplaceAllString(html, "")
	}

	// 移除注释
	if xss.removeComments {
		html = regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(html, "")
	}

	return html
}

// SQLInjectionProtection SQL 注入防护
type SQLInjectionProtection struct {
	patterns []*regexp.Regexp
}

// NewSQLInjectionProtection 创建 SQL 注入防护
func NewSQLInjectionProtection() *SQLInjectionProtection {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)`),
		regexp.MustCompile(`(?i)(--|#|/\*|\*/|;|'|"|\\|%)`),
		regexp.MustCompile(`(?i)(or|and)\s+\d+\s*=\s*\d+`),
		regexp.MustCompile(`(?i)(or|and)\s+['"][^'"]*['"]\s*=\s*['"][^'"]*['"]`),
	}

	return &SQLInjectionProtection{
		patterns: patterns,
	}
}

// CheckSQLInjection 检查 SQL 注入
func (sip *SQLInjectionProtection) CheckSQLInjection(input string) bool {
	input = strings.ToLower(input)
	
	for _, pattern := range sip.patterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	
	return false
}

// SanitizeSQL 清理 SQL 输入
func (sip *SQLInjectionProtection) SanitizeSQL(input string) string {
	// 移除危险字符
	input = regexp.MustCompile(`['"\\;--#/*]`).ReplaceAllString(input, "")
	
	// 标准化空白字符
	input = regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")
	
	return strings.TrimSpace(input)
}

// InputFilter 输入过滤器
type InputFilter struct {
	xssProtection        *XSSProtection
	sqlInjectionProtection *SQLInjectionProtection
	maxLength           int
	allowEmpty          bool
}

// NewInputFilter 创建输入过滤器
func NewInputFilter(maxLength int, allowEmpty bool) *InputFilter {
	return &InputFilter{
		xssProtection:        NewXSSProtection(),
		sqlInjectionProtection: NewSQLInjectionProtection(),
		maxLength:           maxLength,
		allowEmpty:          allowEmpty,
	}
}

// FilterInput 过滤输入
func (ifilter *InputFilter) FilterInput(input string) (string, error) {
	if !ifilter.allowEmpty && strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("input cannot be empty")
	}

	if len(input) > ifilter.maxLength {
		return "", fmt.Errorf("input too long")
	}

	// 检查 SQL 注入
	if ifilter.sqlInjectionProtection.CheckSQLInjection(input) {
		return "", fmt.Errorf("potentially dangerous input detected")
	}

	// 清理 HTML
	input = ifilter.xssProtection.SanitizeHTML(input)

	// URL 解码
	decoded, err := url.QueryUnescape(input)
	if err != nil {
		decoded = input
	}

	return decoded, nil
}

// FilterJSON 过滤 JSON 输入
func (ifilter *InputFilter) FilterJSON(jsonStr string) (map[string]interface{}, error) {
	var data map[string]interface{}
	
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON format")
	}

	filtered := make(map[string]interface{})
	
	for key, value := range data {
		if str, ok := value.(string); ok {
			filteredStr, err := ifilter.FilterInput(str)
			if err != nil {
				return nil, fmt.Errorf("invalid input for field %s: %w", key, err)
			}
			filtered[key] = filteredStr
		} else {
			filtered[key] = value
		}
	}

	return filtered, nil
}
