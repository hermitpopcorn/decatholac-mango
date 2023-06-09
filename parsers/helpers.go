package parsers

import "strings"

func makeFullUrl(url string, baseUrl string) string {
	if strings.HasPrefix(url, "/") && baseUrl != "" {
		url = baseUrl + url
	}
	return url
}
