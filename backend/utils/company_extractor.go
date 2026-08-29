package utils

import (
	"net/url"
	"strings"
)

/*
ExtractCompanyDomain extracts the genuine root domain name of a company from a direct job URL,
ignoring generic third-party job boards and ATS aggregator platforms.
*/
func ExtractCompanyDomain(jobURL string) string {
	cleanURL := strings.TrimSpace(jobURL)
	if cleanURL == "" {
		return ""
	}

	parsedURL, err := url.Parse(cleanURL)
	if err != nil || parsedURL.Host == "" {
		return ""
	}

	host := strings.ToLower(parsedURL.Host)
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	aggregatorDomains := []string{
		"greenhouse.io", "lever.co", "ashbyhq.com", "smartrecruiters.com",
		"workdaysite.com", "myworkdaysite.com", "indeed.com", "linkedin.com",
		"dice.com", "amazon.jobs", "ycombinator.com", "remoteok.com",
		"weworkremotely.com", "themuse.com", "himalayas.app", "jobspresso.co",
		"workingnomads.com", "news.ycombinator.com", "web3.career", "cryptojobslist.com",
		"rust.careers", "simplyhired.com", "glassdoor.com", "ziprecruiter.com",
	}

	for _, aggregator := range aggregatorDomains {
		if strings.Contains(host, aggregator) {
			return ""
		}
	}

	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		mainDomain := parts[len(parts)-2] + "." + parts[len(parts)-1]
		if parts[len(parts)-2] == "co" || parts[len(parts)-2] == "com" || parts[len(parts)-2] == "org" || parts[len(parts)-2] == "gov" || parts[len(parts)-2] == "edu" {
			if len(parts) >= 3 {
				mainDomain = parts[len(parts)-3] + "." + parts[len(parts)-2] + "." + parts[len(parts)-1]
			}
		}
		return mainDomain
	}

	return host
}

/*
ExtractCompanyName infers a clean, human-readable company name from raw company input,
ATS job board URLs (Greenhouse, Lever, Ashby, Workday), domain names, or title separators.
*/
func ExtractCompanyName(rawCompany string, jobURL string, rawTitle string) string {
	cleanCompany := strings.TrimSpace(rawCompany)
	if cleanCompany != "" && !strings.EqualFold(cleanCompany, "unknown") && !strings.EqualFold(cleanCompany, "null") {
		return cleanCompany
	}

	if cleanURL := strings.TrimSpace(jobURL); cleanURL != "" {
		if parsedURL, err := url.Parse(cleanURL); err == nil && parsedURL.Host != "" {
			host := strings.ToLower(parsedURL.Host)
			pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")

			if strings.Contains(host, "greenhouse.io") && len(pathParts) > 0 && pathParts[0] != "" {
				return CapitalizeSlug(pathParts[0])
			}
			if strings.Contains(host, "lever.co") && len(pathParts) > 0 && pathParts[0] != "" {
				return CapitalizeSlug(pathParts[0])
			}
			if strings.Contains(host, "ashbyhq.com") && len(pathParts) > 0 && pathParts[0] != "" {
				return CapitalizeSlug(pathParts[0])
			}
			if strings.Contains(host, "workdaysite.com") {
				subdomains := strings.Split(host, ".")
				if len(subdomains) > 0 && subdomains[0] != "www" {
					return CapitalizeSlug(subdomains[0])
				}
			}

			domainParts := strings.Split(host, ".")
			if len(domainParts) >= 2 {
				mainName := domainParts[len(domainParts)-2]
				if mainName != "greenhouse" && mainName != "lever" && mainName != "ashbyhq" && mainName != "workdaysite" && mainName != "myworkdaysite" && mainName != "jobspy" && mainName != "google" {
					return CapitalizeSlug(mainName)
				}
			}
		}
	}

	if strings.Contains(rawTitle, " at ") {
		parts := strings.Split(rawTitle, " at ")
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			return strings.TrimSpace(parts[1])
		}
	}

	return "Unknown"
}

/*
CapitalizeSlug converts a hyphenated or underscored slug into a clean title.
*/
func CapitalizeSlug(slug string) string {
	clean := strings.ReplaceAll(slug, "-", " ")
	clean = strings.ReplaceAll(clean, "_", " ")
	words := strings.Fields(clean)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
