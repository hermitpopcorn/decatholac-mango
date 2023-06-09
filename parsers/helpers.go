package parsers

import "strings"

// Make full URL by prepending a base URL if it's a relative URL.
func makeFullUrl(url string, baseUrl string) string {
	if strings.HasPrefix(url, "/") && baseUrl != "" {
		url = baseUrl + url
	}
	return url
}
