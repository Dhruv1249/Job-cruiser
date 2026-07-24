package utils

import (
	"regexp"
	"strconv"
	"strings"
)

var salaryRangeRe = regexp.MustCompile(`(?i)\$([0-9]{2,3}(?:,[0-9]{3})?k?)\s*(?:-|to)\s*\$?([0-9]{2,3}(?:,[0-9]{3})?k?)`)

func parseSalaryVal(valStr string) int {
	valStr = strings.ToLower(strings.TrimSpace(valStr))
	isK := strings.HasSuffix(valStr, "k")
	clean := strings.ReplaceAll(strings.ReplaceAll(valStr, "k", ""), ",", "")
	num, err := strconv.Atoi(clean)
	if err != nil {
		return 0
	}
	if isK || (num >= 20 && num <= 500) {
		num = num * 1000
	}
	return num
}

func ExtractSalaryFromText(text string) (salMin *int, salMax *int) {
	if text == "" {
		return nil, nil
	}
	match := salaryRangeRe.FindStringSubmatch(text)
	if len(match) >= 3 {
		minVal := parseSalaryVal(strings.TrimPrefix(match[1], "$"))
		maxVal := parseSalaryVal(strings.TrimPrefix(match[2], "$"))
		if minVal > 10000 && maxVal >= minVal {
			return &minVal, &maxVal
		}
	}
	return nil, nil
}
