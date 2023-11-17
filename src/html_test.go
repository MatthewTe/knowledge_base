package main

import (
	"knowledge_base/parsers"

	"testing"
)

func TestHTMLContentExtraction(t *testing.T) {

	// Creating HTML core component to test all processes:
	testComponent := parsers.HtmlContent{Url: "http://localhost:8000/test/html_page"}

	testComponent.LoadHtmlPage()

}
