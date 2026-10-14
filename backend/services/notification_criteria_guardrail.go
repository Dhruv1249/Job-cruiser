/*
Package services provides business logic for AI job matching, CV parsing, and queue management.
*/
package services

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	ppoRegexPattern               = regexp.MustCompile(`\bppo\b`)
	stipendCriteriaRegex          = regexp.MustCompile(`(?:stipend|internship)\s*(?:with\s*stipend\s*)?<\s*(?:[₹$]|rs\.?|inr)?\s*(\d+(?:,\d+)*)\s*(k)?`)
	fullTimeLpaCriteriaRegex      = regexp.MustCompile(`(?:full-?time\s*(?:job|role)?\s*<\s*|<)\s*(\d+(?:\.\d+)?)\s*lpa`)
	stipendInDescriptionRegex     = regexp.MustCompile(`(?:stipend\s*(?:of|:)?|up\s*to)\s*(?:[₹$]|rs\.?|inr)?\s*(\d+(?:,\d+)*)(?:\s*(k))?`)
	monthlyStipendMentionRegex    = regexp.MustCompile(`(?:[₹$]|rs\.?|inr)\s*(\d+(?:,\d+)*)\s*(?:\/|\s*per\s*)(?:month|mth|pm)`)
	lpaRangeInDescriptionRegex    = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:-|to)\s*(\d+(?:\.\d+)?)\s*lpa`)
	singleLpaInDescriptionRegex   = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*lpa`)
)

/*
EvaluateNotificationCriteriaGuardrails deterministically evaluates whether a job passes candidate alerting criteria.
*/
func EvaluateNotificationCriteriaGuardrails(
	criteriaPrompt string,
	jobTitle string,
	employmentType string,
	salaryMin int,
	salaryMax int,
	salaryPeriod string,
	salaryCurrency string,
	rawDescription string,
) bool {
	trimmedCriteria := strings.TrimSpace(criteriaPrompt)
	if trimmedCriteria == "" {
		return true
	}

	lowercasedCriteria := strings.ToLower(trimmedCriteria)
	lowercasedTitle := strings.ToLower(jobTitle)
	lowercasedEmploymentType := strings.ToLower(strings.TrimSpace(employmentType))
	lowercasedDescription := strings.ToLower(rawDescription)

	isInternshipRole := lowercasedEmploymentType == "intern" ||
		lowercasedEmploymentType == "intern_ppo" ||
		strings.Contains(lowercasedTitle, "intern")

	disqualifiesStandaloneInternships := strings.Contains(lowercasedCriteria, "disqualify standalone internship") ||
		strings.Contains(lowercasedCriteria, "without ppo") ||
		strings.Contains(lowercasedCriteria, "internship with explicit ppo") ||
		strings.Contains(lowercasedCriteria, "internship with ppo") ||
		strings.Contains(lowercasedCriteria, "no standalone internship")

	if isInternshipRole && disqualifiesStandaloneInternships {
		hasExplicitPPO := lowercasedEmploymentType == "intern_ppo" ||
			containsExplicitPPOOffer(lowercasedTitle, lowercasedDescription)
		if !hasExplicitPPO {
			return false
		}
	}

	candidateMinimumStipend := extractCandidateMinimumStipend(lowercasedCriteria)
	if isInternshipRole && candidateMinimumStipend > 0 {
		stipendViolated := isStatedStipendBelowThreshold(
			candidateMinimumStipend,
			salaryMin,
			salaryPeriod,
			lowercasedDescription,
		)
		if stipendViolated {
			return false
		}
	}

	candidateMinimumFullTimeAnnualSalary := extractCandidateMinimumFullTimeAnnualSalary(lowercasedCriteria)
	if !isInternshipRole && candidateMinimumFullTimeAnnualSalary > 0 {
		salaryViolated := isStatedFullTimeSalaryBelowThreshold(
			candidateMinimumFullTimeAnnualSalary,
			salaryMin,
			salaryMax,
			lowercasedDescription,
		)
		if salaryViolated {
			return false
		}
	}

	return true
}

func containsExplicitPPOOffer(lowercasedTitle string, lowercasedDescription string) bool {
	if ppoRegexPattern.MatchString(lowercasedTitle) || ppoRegexPattern.MatchString(lowercasedDescription) {
		return true
	}

	explicitPhrases := []string{
		"pre-placement offer",
		"pre placement offer",
		"full-time conversion",
		"full time conversion",
		"conversion to full-time",
		"conversion to full time",
		"convert to full-time",
		"convert to full time",
		"permanent offer after internship",
		"opportunity for full-time",
		"opportunity for a full-time",
	}

	for _, phrase := range explicitPhrases {
		if strings.Contains(lowercasedTitle, phrase) || strings.Contains(lowercasedDescription, phrase) {
			return true
		}
	}

	return false
}

func extractCandidateMinimumStipend(lowercasedCriteria string) int {
	matches := stipendCriteriaRegex.FindStringSubmatch(lowercasedCriteria)
	if len(matches) < 2 {
		return 0
	}

	cleanedNumericString := strings.ReplaceAll(matches[1], ",", "")
	parsedValue, parseError := strconv.Atoi(cleanedNumericString)
	if parseError != nil {
		return 0
	}

	if len(matches) >= 3 && strings.EqualFold(matches[2], "k") {
		return parsedValue * 1000
	}

	return parsedValue
}

func extractCandidateMinimumFullTimeAnnualSalary(lowercasedCriteria string) int {
	matches := fullTimeLpaCriteriaRegex.FindStringSubmatch(lowercasedCriteria)
	if len(matches) < 2 {
		return 0
	}

	parsedLpa, parseError := strconv.ParseFloat(matches[1], 64)
	if parseError != nil {
		return 0
	}

	return int(parsedLpa * 100000)
}

func isStatedStipendBelowThreshold(
	minimumStipendThreshold int,
	salaryMin int,
	salaryPeriod string,
	lowercasedDescription string,
) bool {
	if salaryMin > 0 && (salaryPeriod == "monthly" || salaryPeriod == "stipend" || salaryMin < 100000) {
		if salaryMin < minimumStipendThreshold {
			return true
		}
	}

	monthlyMatches := monthlyStipendMentionRegex.FindStringSubmatch(lowercasedDescription)
	if len(monthlyMatches) >= 2 {
		cleanedNumber := strings.ReplaceAll(monthlyMatches[1], ",", "")
		if detectedValue, err := strconv.Atoi(cleanedNumber); err == nil && detectedValue > 0 {
			if detectedValue < minimumStipendThreshold {
				return true
			}
		}
	}

	stipendMatches := stipendInDescriptionRegex.FindStringSubmatch(lowercasedDescription)
	if len(stipendMatches) >= 2 {
		cleanedNumber := strings.ReplaceAll(stipendMatches[1], ",", "")
		if detectedValue, err := strconv.Atoi(cleanedNumber); err == nil && detectedValue > 0 {
			multiplier := 1
			if len(stipendMatches) >= 3 && strings.EqualFold(stipendMatches[2], "k") {
				multiplier = 1000
			}
			totalStipend := detectedValue * multiplier
			if totalStipend < minimumStipendThreshold {
				return true
			}
		}
	}

	return false
}

func isStatedFullTimeSalaryBelowThreshold(
	minimumAnnualThreshold int,
	salaryMin int,
	salaryMax int,
	lowercasedDescription string,
) bool {
	if salaryMax > 0 && salaryMax < minimumAnnualThreshold {
		return true
	}

	if salaryMin > 0 && salaryMax == 0 && salaryMin < minimumAnnualThreshold {
		return true
	}

	rangeMatches := lpaRangeInDescriptionRegex.FindStringSubmatch(lowercasedDescription)
	if len(rangeMatches) >= 3 {
		upperLpa, parseError := strconv.ParseFloat(rangeMatches[2], 64)
		if parseError == nil && upperLpa > 0 {
			upperAnnual := int(upperLpa * 100000)
			if upperAnnual < minimumAnnualThreshold {
				return true
			}
		}
	}

	singleMatches := singleLpaInDescriptionRegex.FindStringSubmatch(lowercasedDescription)
	if len(singleMatches) >= 2 {
		singleLpa, parseError := strconv.ParseFloat(singleMatches[1], 64)
		if parseError == nil && singleLpa > 0 {
			singleAnnual := int(singleLpa * 100000)
			if singleAnnual < minimumAnnualThreshold {
				return true
			}
		}
	}

	return false
}
