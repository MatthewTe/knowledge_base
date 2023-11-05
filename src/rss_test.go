package main

import (
	"context"
	"encoding/json"
	"fmt"
	"knowledge_base/parsers"
	"log"
	"os"
	"time"

	"testing"

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

/*
func TestRssFeedXMLIngestion(t *testing.T) {

	testXMLPath := "../data/rss/38_north_test.rss"
	xmlFile, err := os.Open(testXMLPath)
	if err != nil {
		log.Fatal(err)
	}

	parsers.ExtractFieldsFromRssFeed(xmlFile)

}
*/

func TestRssFeedExtraction(t *testing.T) {

	fmt.Println("------------------------ TestRssFeedExtraction ------------------------ ")
	fmt.Println("RssFeed")

	err := godotenv.Load("../data/test.env")
	if err != nil {
		log.Fatal("Unable to load environment variable for tests", err)
	}

	ctx := context.Background()
	dbUri := os.Getenv("dbUri")
	dbUser := os.Getenv("dbUser")
	dbPassword := os.Getenv("dbPassword")
	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))
	if err != nil {
		log.Fatal(nil)
	}
	defer driver.Close(ctx)

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		log.Fatal(err)
	}

	name := "38 North"

	// Create the 38 North entry if it doesn't exist:
	result, err := neo4j.ExecuteQuery(
		ctx,
		driver,
		`MERGE (rss_feed:Rss_Feed:Source {name: $name})
		ON CREATE SET
			rss_feed.url = $url,
			rss_feed.scheduled_time = $scheduled_time,
			rss_feed.etag = $etag,
			rss_feed.last_updated = $last_updated,
			rss_feed.created = timestamp()
		RETURN rss_feed`,
		map[string]any{
			"name":           "38 North",
			"url":            "http://0.0.0.0:8000/test/rss_feed",
			"scheduled_time": "18:00",
			"etag":           "",
			"last_updated":   "",
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		log.Fatal("Error in creating 38 North node for testing", err)
	}

	fmt.Printf(
		"Created %v nodes in %+v. \n",
		result.Summary.Counters().NodesCreated(),
		result.Summary.ResultAvailableAfter(),
	)

	for _, record := range result.Records {

		nodeDict := record.AsMap()

		for _, nodeKey := range nodeDict {

			switch node := nodeKey.(type) {
			case neo4j.Node:

				nodeProps := node.GetProperties()

				rssFeed := parsers.RssFeed{Id: node.ElementId}

				if url, ok := nodeProps["url"].(string); ok {
					rssFeed.Url = url
				} else {
					log.Fatal("No Url for Node", node.ElementId)
				}
				if title, ok := nodeProps["name"].(string); ok {
					rssFeed.Title = title
				} else {
					log.Fatal("No name for Node", node.ElementId)
				}

				if scheduledTime, ok := nodeProps["scheduled_time"].(string); ok {
					rssFeed.ExecuteTime = scheduledTime
				} else {
					rssFeed.ExecuteTime = ""
				}

				if etag, ok := nodeProps["etag"].(string); ok {
					rssFeed.Etag = etag
				} else {
					rssFeed.Etag = ""
				}

				if lastUpdated, ok := nodeProps["last_updated"].(string); ok {
					rssFeed.LastUpdate = lastUpdated
				} else {
					rssFeed.LastUpdate = ""
				}

				fmt.Println("Id", node.ElementId)
				fmt.Println("Url", rssFeed.Url)
				fmt.Println("Title", rssFeed.Title)
				fmt.Println("Schedule Time", rssFeed.ExecuteTime)
				fmt.Println("Etag", rssFeed.Etag)
				fmt.Println("Last Updated", rssFeed.LastUpdate)
			}
		}
	}

	_, err = parsers.GetRssSourceFromDatabase(name, ctx, driver)
	if err != nil {
		log.Fatal("Error in Extracting Feeds:", err)
	}
	fmt.Println("")
}

func TestRssFeedEntryExtraction(t *testing.T) {

	fmt.Println("---------------------- TestRssFeedEntryExtraction ----------------------")
	fmt.Println("RssEntry")

	err := godotenv.Load("../data/test.env")
	if err != nil {
		log.Fatal("Unable to load environment variable for tests", err)
	}

	ctx := context.Background()
	dbUri := os.Getenv("dbUri")
	dbUser := os.Getenv("dbUser")
	dbPassword := os.Getenv("dbPassword")
	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))
	if err != nil {
		log.Fatal(nil)
	}
	defer driver.Close(ctx)

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		log.Fatal(err)
	}

	name := "Example Title hello world"
	url := "www.google.com"

	// Creating the article if it does not exist:
	result, err := neo4j.ExecuteQuery(
		ctx,
		driver,
		`
		MERGE (article:Rss_Feed:Article {name: $name, url: $url})
		ON CREATE SET
			article.description = $description,
			article.date_posted = $date_posted,
			article.in_static_file_storage = $in_static_file_storage,
			article.created = timestamp()
		WITH article
		
		MATCH (source:Rss_Feed:Source {name: $rss_source_name})

		CREATE (source)-[rel:CONTAINS_ARTICLE]->(article)
		SET rel.date_downloaded = $downloaded_date
		return article
		`,
		map[string]any{
			"name":                   name,
			"url":                    url,
			"description":            "This is a description of a test article",
			"date_posted":            "2006-01-02",
			"in_static_file_storage": 0,
			"downloaded_date":        time.Now().Format("2006-01-02"),
			"rss_source_name":        "38 North",
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		log.Fatal("Error in creating the article with rss source connection:", err)
	}

	article, err := parsers.GetRssArticleFromDatabase(name, url, ctx, driver)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Created %v nodes in %+v. \n",
		result.Summary.Counters().NodesCreated(),
		result.Summary.ResultAvailableAfter(),
	)

	if (parsers.RssEntry{}) == article {
		fmt.Println("Empty data returned. No match found")
	} else {
		fmt.Println("Id: ", article.Id)
		fmt.Println("Title: ", article.Title)
		fmt.Println("Url: ", article.Url)
		fmt.Println("Description: ", article.Description)
		fmt.Println("DatePosted: ", article.DatePosted)
		fmt.Println("DateExtracted: ", article.DateExtracted)
		fmt.Println("InStorage: ", article.InStorage)
		fmt.Println("StorageUrl: ", article.StorageUrl)
	}
	fmt.Println("")
}

func TestRssAuthorExtraction(t *testing.T) {

	fmt.Println("---------------------- TestRssAuthorExtraction ----------------------")

	err := godotenv.Load("../data/test.env")
	if err != nil {
		log.Fatal("Unable to load environment variable for tests", err)
	}

	ctx := context.Background()
	dbUri := os.Getenv("dbUri")
	dbUser := os.Getenv("dbUser")
	dbPassword := os.Getenv("dbPassword")

	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))
	if err != nil {
		log.Fatal(nil)
	}
	defer driver.Close(ctx)

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		log.Fatal(err)
	}

	name := "Martyn Williams"
	email := "martynwilliams@gmail.com"

	// Creating the author if it doesn't exist:
	connectionResult, err := neo4j.ExecuteQuery(
		ctx,
		driver,
		`
		MATCH (article:Rss_Feed:Article {name: $article_name, url: $article_url})
		
		MERGE (author:Rss_Feed:Author:Person {name: $author_name})
		ON CREATE SET author.name = $author_name, author.email = $author_email

		CREATE (author)-[:WROTE]->(article)

		RETURN article, author
		`,
		map[string]any{
			"article_name": "Example Title hello world",
			"article_url":  "www.google.com",
			"author_name":  name,
			"author_email": email,
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		fmt.Println("Error in creating the article author and connection to the article", err)
	}

	fmt.Printf(
		"Created %v nodes in %+v. \n",
		connectionResult.Summary.Counters().NodesCreated(),
		connectionResult.Summary.ResultAvailableAfter(),
	)

	author, err := parsers.GetAuthorFromDatabase(name, ctx, driver)
	if err != nil {
		log.Fatal(err)
	}
	if (parsers.RssAuthor{}) == author {
		fmt.Println("Empty data returned. No match found")
	} else {
		fmt.Println("Id: ", author.Id)
		fmt.Println("Name: ", author.Name)
		fmt.Println("Email: ", author.Email)
	}
	fmt.Println("")
}

// Testing how the function handler for the endpoint that triggers the ingestion of the Rss Entries and
// Authors:
func TestRssEntryIngestionHandler(t *testing.T) {

	fmt.Println("-------------------- TestRssEntryIngestionHandler --------------------")

	err := godotenv.Load("../data/test.env")
	if err != nil {
		log.Fatal("Unable to load environment variable for tests", err)
	}

	ctx := context.Background()
	dbUri := os.Getenv("dbUri")
	dbUser := os.Getenv("dbUser")
	dbPassword := os.Getenv("dbPassword")
	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))
	if err != nil {
		log.Fatal(nil)
	}
	defer driver.Close(ctx)

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		log.Fatal(err)
	}

	ingestionResponse, err := parsers.IngestAllRssItems("38 North", ctx, driver)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	j, _ := json.MarshalIndent(ingestionResponse, "", "    ")
	fmt.Println(string(j))

}
