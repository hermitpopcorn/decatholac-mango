package types

type Target struct {
	Name            string
	Source          string
	AscendingSource bool // Whether the source lists item A->Z instead of Z->A like normal
	Mode            string
	BaseUrl         string
	RequestHeaders  map[string]string

	// JSON mode
	Keys Keys

	// HTML mode
	Tags Tags
}

type Keys struct {
	Chapters   string
	Number     string
	Title      string
	Date       string
	DateFormat string
	Url        string
	Skip       map[string]any
}

type Tags struct {
	ChaptersTag     string
	NumberTag       string
	NumberAttribute string
	TitleTag        string
	TitleAttribute  string
	DateTag         string
	DateAttribute   string
	DateFormat      string
	UrlTag          string
	UrlAttribute    string
}
