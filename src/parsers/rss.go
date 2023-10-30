package parsers

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Function Extracts all of the RSS entries from an xml rss document. Parses the xml according to a
// predefined schema:
func ExtractFieldsFromRssFeed(file *os.File) (err error) {

	fp := gofeed.NewParser()
	feed, _ := fp.Parse(file)

	fmt.Println(feed.UpdatedParsed.Format(time.RFC3339Nano))

	for _, element := range feed.Items {

		//fmt.Println(element.Title)
		// fmt.Println(element.Link)

		for _, author := range element.Authors {
			fmt.Println(author.Name)
			fmt.Println(author.Email)
		}

		//fmt.Println(element.Description)

		fmt.Println("------------------------")

	}

	// Step 1: Create the Rss Feed entry node with all of the associated fields
	// Step 2: Create the connection from the root rss feed to this newly created article w/ downloaded time
	// Step 3: Create the Author nodes containing information.
	// Step 4: Create the connection from the Author to the Rss feed.
	// Step 5: Update the Rss Feed Source

	return nil

}

// Querying the database for a specific rss feed source given a name:
func GetRssSourceFromDatabase(name string, ctx context.Context, driver neo4j.DriverWithContext) (rssSource RssFeed, err error) {

	// Querying the node from the graph database:
	results, err := neo4j.ExecuteQuery(
		ctx,
		driver,
		"MATCH (source:Rss_Feed:Source {name: $name}) RETURN source",
		map[string]any{"name": name},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		log.Fatal(err)
		return rssSource, err
	}

	if len(results.Records) == 0 {
		err = fmt.Errorf("unable to find Rss feed node in the database for rss feed %s", name)
		return rssSource, err
	}

	if len(results.Records) > 1 {
		err = fmt.Errorf("more than one rss source returned. %d returned", len(results.Records))
		log.Fatal(err)
		return rssSource, err
	}

	extractedNodeDict := results.Records[0].AsMap()
	// A really bad way of doing this. Creating an array that will always be length 1 and appending it
	// Find a better way of parsing an arbitrary key/value pair from a dict that will always be len 1.
	var rssSourceArray []RssFeed

	for _, nodeKey := range extractedNodeDict {
		switch node := nodeKey.(type) {
		case neo4j.Node:

			nodeProps := node.GetProperties()

			rssSource := RssFeed{Id: node.ElementId}

			if url, ok := nodeProps["url"].(string); ok {
				rssSource.Url = url
			} else {
				log.Fatal("No Url for Node", node.ElementId)
			}
			if title, ok := nodeProps["name"].(string); ok {
				rssSource.Title = title
			} else {
				log.Fatal("No name for Node", node.ElementId)
			}

			if scheduledTime, ok := nodeProps["scheduled_time"].(string); ok {
				rssSource.ExecuteTime = scheduledTime
			} else {
				rssSource.ExecuteTime = ""
			}

			if etag, ok := nodeProps["etag"].(string); ok {
				rssSource.Etag = etag
			} else {
				rssSource.Etag = ""
			}

			if lastUpdated, ok := nodeProps["last_updated"].(string); ok {
				rssSource.LastUpdate = lastUpdated
			} else {
				rssSource.LastUpdate = ""
			}

			rssSourceArray = append(rssSourceArray, rssSource)
		}
	}

	return rssSourceArray[0], err

}

// Querying the database for a specific rss feed entry given a title and a url:
func GetRssArticleFromDatabase(name string, url string, ctx context.Context, driver neo4j.DriverWithContext) (insertedEntry RssEntry, err error) {
	// Querying the node from the graph database:
	results, err := neo4j.ExecuteQuery(
		ctx,
		driver,
		"MATCH (article:Rss_Feed:Article {name: $name, url: $url}) RETURN article",
		map[string]any{"name": name, "url": url},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		log.Fatal(err)
		return insertedEntry, err
	}

	if len(results.Records) == 0 {
		return insertedEntry, nil
	}

	if len(results.Records) > 1 {
		err = fmt.Errorf("more than one rss entry returned. %d returned", len(results.Records))
		log.Fatal(err)
		return insertedEntry, err
	}

	var rssEntryArray []RssEntry

	extractedNodeDict := results.Records[0].AsMap()

	for _, nodeKey := range extractedNodeDict {
		switch node := nodeKey.(type) {
		case neo4j.Node:

			nodeProps := node.GetProperties()

			rssEntry := RssEntry{Id: node.ElementId}

			if url, ok := nodeProps["url"].(string); ok {
				rssEntry.Url = url
			} else {
				log.Fatal("No Url for Node", node.ElementId)
			}
			if title, ok := nodeProps["name"].(string); ok {
				rssEntry.Title = title
			} else {
				log.Fatal("No name for Node", node.ElementId)
			}

			if datePosted, ok := nodeProps["date_posted"].(string); ok {
				rssEntry.DatePosted = datePosted
			} else {
				log.Fatal("No Date Posted for Node", node.ElementId)
			}

			if StaticFileUrl, ok := nodeProps["static_file_url"].(string); ok {
				rssEntry.StorageUrl = StaticFileUrl
			} else {
				rssEntry.StorageUrl = ""
			}

			if InStorage, ok := nodeProps["in_static_file_storage"].(int); ok {
				rssEntry.InStorage = InStorage
			} else {
				rssEntry.InStorage = 0
			}

			rssEntryArray = append(rssEntryArray, rssEntry)
		}
	}
	return rssEntryArray[0], err
}

// Function that checks the Database for an Author:
func GetAuthorFromDatabase(name string, ctx context.Context, driver neo4j.DriverWithContext) (author RssAuthor, err error) {
	// Querying the node from the graph database:
	results, err := neo4j.ExecuteQuery(
		ctx,
		driver,
		"MATCH (author:Rss_Feed:Author:Person {name: $name}) RETURN author",
		map[string]any{"name": name},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		log.Fatal(err)
		return author, err
	}

	if len(results.Records) == 0 {
		return author, nil
	}

	if len(results.Records) > 1 {
		err = fmt.Errorf("more than one author returned. %d returned", len(results.Records))
		log.Fatal(err)
		return author, err
	}

	extractedNodeDict := results.Records[0].AsMap()
	// A really bad way of doing this. Creating an array that will always be length 1 and appending it
	// Find a better way of parsing an arbitrary key/value pair from a dict that will always be len 1.
	var personArray []RssAuthor

	for _, nodeKey := range extractedNodeDict {
		switch node := nodeKey.(type) {
		case neo4j.Node:

			nodeProps := node.GetProperties()

			author := RssAuthor{Id: node.ElementId}

			if name, ok := nodeProps["name"].(string); ok {
				author.Name = name
			} else {
				err = fmt.Errorf("no name found for extracted author node")
				return author, err
			}
			if email, ok := nodeProps["email"].(string); ok {
				author.Email = email
			} else {
				author.Email = ""
			}

			personArray = append(personArray, author)
		}
	}

	return personArray[0], err

}

/*
func IngestRssFeedItem(item *gofeed.Item, feed *RssFeed, ctx context.Context, driver neo4j.DriverWithContext) (ingestedEntry RssEntry, err error) {

}
*/
