package tests

import (
	"knowledge_base/parsers"
	"log"
	"os"
	"testing"
)

func TestRssFeedXMLIngestion(t *testing.T) {

	testXMLPath := "../../data/rss/38_north_test.rss"
	xmlFile, err := os.Open(testXMLPath)
	if err != nil {
		log.Fatal(err)
	}

	parsers.ExtractFieldsFromRssFeed(xmlFile)

}
