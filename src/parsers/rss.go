package parsers

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"github.com/mmcdole/gofeed"
)

func IngestRssEntry() {

}

type XmlSchema struct {
	XmlCategory xml.Attr `xml:"channel"`
}

type RssEntry struct {
	Id              int    `json:"id"`
	Url             string `json:"url"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	RssFeedId       int    `json:"rss_feed_id"`
	DatePosted      int    `json:"date_posted"`
	DateExtracted   int    `json:"date_extracted"`
	InStorage       int    `json:"in_storage"`
	StorageInserted string `json:"storage_inserted"`
}
type RssEntries struct {
	Entries []RssEntry
}

// Function Extracts all of the RSS entries from an xml rss document. Parses the xml according to a
// predefined schema:
func ExtractFieldsFromRssFeed(file *os.File) (err error) {

	fp := gofeed.NewParser()
	feed, _ := fp.Parse(file)

	fmt.Println(feed.UpdatedParsed.Format(time.RFC3339Nano))

	for _, element := range feed.Items {

		fmt.Println(element.Title)
		fmt.Println(element.Link)

		for _, author := range element.Authors {
			fmt.Println(author.Name)
		}

		fmt.Println(element.Description)

		fmt.Println("------------------------")

	}

	return nil

}
