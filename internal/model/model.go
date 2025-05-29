package model

import (
	"html/template"
	"time"
)

// ContentItem represents a single piece of content (e.g., blog post, project page).
type ContentItem struct {
	Title       string
	Date        time.Time
	Type        string
	SourcePath  string
	Permalink   string
	ContentHTML template.HTML
	Frontmatter map[string]interface{}
	Summary     string
	Layout      string
}

// SiteData holds all site-wide data, including configuration and content.
type SiteData struct {
	Config        map[string]interface{}
	ContentItems  []*ContentItem
	Posts         []*ContentItem
	Projects      []*ContentItem
	ContentByType map[string][]*ContentItem
}
