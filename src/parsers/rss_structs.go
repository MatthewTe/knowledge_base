package parsers

import "encoding/xml"

type XmlSchema struct {
	XmlCategory xml.Attr `xml:"channel"`
}

type RssEntry struct {
	Id            string `json:"id"`
	Url           string `json:"url"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	DatePosted    string `json:"date_posted"`
	DateExtracted int    `json:"date_extracted"`
	InStorage     int    `json:"in_storage"`
	StorageUrl    string `json:"storage_inserted"`
}
type RssEntries struct {
	Entries []RssEntry
}
type RssFeed struct {
	Id          string `json:"id"`
	Url         string `json:"url"`
	Title       string `json:"title"`
	Etag        string `json:"etag"`
	LastUpdate  string `json:"last_updated"`
	ExecuteTime string `json:"execute_time"`
}
type RssFeeds struct {
	Entries []RssFeed
}
type RssAuthor struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type RssAuthorExtractionSummary struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

type RssEntryExtractionSummary struct {
	Id      string                       `json:"id"`
	Title   string                       `json:"title"`
	Url     string                       `json:"url"`
	Status  string                       `json:"status"`
	Error   string                       `json:"error"`
	Authors []RssAuthorExtractionSummary `json:"authors"`
}

type RssFeedExtractionSummary struct {
	Id         string                      `json:"id"`
	Title      string                      `json:"title"`
	Status     string                      `json:"status"`
	Error      string                      `json:"error"`
	RssFeed    RssFeed                     `json:"source_feed"`
	RssEntries []RssEntryExtractionSummary `json:"entries"`
}
