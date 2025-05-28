package model

import "html/template"

type PageData struct {
	SiteTitle string
	PageTitle string
	Content   template.HTML
	BaseURL   string
	Date      string
	Params    map[string]interface{}
	Layout    string // New field for specifying the layout
}
