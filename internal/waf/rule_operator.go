package waf

type RuleOperator = string
type RuleCaseInsensitive = string

const (
	RuleOperatorGt                           RuleOperator = "gt"
	RuleOperatorGte                          RuleOperator = "gte"
	RuleOperatorLt                           RuleOperator = "lt"
	RuleOperatorLte                          RuleOperator = "lte"
	RuleOperatorEq                           RuleOperator = "eq"
	RuleOperatorNeq                          RuleOperator = "neq"
	RuleOperatorEqString                     RuleOperator = "eq string"
	RuleOperatorNeqString                    RuleOperator = "neq string"
	RuleOperatorMatch                        RuleOperator = "match"
	RuleOperatorNotMatch                     RuleOperator = "not match"
	RuleOperatorWildcardMatch                RuleOperator = "wildcard match"
	RuleOperatorWildcardNotMatch             RuleOperator = "wildcard not match"
	RuleOperatorContains                     RuleOperator = "contains"
	RuleOperatorNotContains                  RuleOperator = "not contains"
	RuleOperatorPrefix                       RuleOperator = "prefix"
	RuleOperatorSuffix                       RuleOperator = "suffix"
	RuleOperatorContainsAny                  RuleOperator = "contains any"
	RuleOperatorContainsAll                  RuleOperator = "contains all"
	RuleOperatorContainsAnyWord              RuleOperator = "contains any word"
	RuleOperatorContainsAllWords             RuleOperator = "contains all words"
	RuleOperatorNotContainsAnyWord           RuleOperator = "not contains any word"
	RuleOperatorContainsSQLInjection         RuleOperator = "contains sql injection"
	RuleOperatorContainsSQLInjectionStrictly RuleOperator = "contains sql injection strictly"
	RuleOperatorContainsXSS                  RuleOperator = "contains xss"
	RuleOperatorContainsXSSStrictly          RuleOperator = "contains xss strictly"
	RuleOperatorInIPList                     RuleOperator = "in ip list"
	RuleOperatorHasKey                       RuleOperator = "has key" // has key in slice or map
	RuleOperatorVersionGt                    RuleOperator = "version gt"
	RuleOperatorVersionLt                    RuleOperator = "version lt"
	RuleOperatorVersionRange                 RuleOperator = "version range"

	RuleOperatorContainsBinary    RuleOperator = "contains binary"     // contains binary
	RuleOperatorNotContainsBinary RuleOperator = "not contains binary" // not contains binary

	// ip

	RuleOperatorEqIP       RuleOperator = "eq ip"
	RuleOperatorGtIP       RuleOperator = "gt ip"
	RuleOperatorGteIP      RuleOperator = "gte ip"
	RuleOperatorLtIP       RuleOperator = "lt ip"
	RuleOperatorLteIP      RuleOperator = "lte ip"
	RuleOperatorIPRange    RuleOperator = "ip range"
	RuleOperatorNotIPRange RuleOperator = "not ip range"
	RuleOperatorIPMod10    RuleOperator = "ip mod 10"
	RuleOperatorIPMod100   RuleOperator = "ip mod 100"
	RuleOperatorIPMod      RuleOperator = "ip mod"
)

type RuleOperatorDefinition struct {
	Name            string
	Code            string
	Description     string
	CaseInsensitive RuleCaseInsensitive // default caseInsensitive setting
}
