package main

import (
	"context"
	"fmt"
	"knowledge_base/parsers"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestRssFeedXMLIngestion(t *testing.T) {

	testXMLPath := "../data/rss/38_north_test.rss"
	xmlFile, err := os.Open(testXMLPath)
	if err != nil {
		log.Fatal(err)
	}

	parsers.ExtractFieldsFromRssFeed(xmlFile)

}

func TestRssFeedExtraction(t *testing.T) {

	// TODO: Make this configuration specific to tests using environment params

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
	_, err = parsers.GetRssSourceFromDatabase(name, ctx, driver)
	if err != nil {
		log.Fatal("Error in Extracting Feeds:", err)
	}

}

func TestRssFeedEntryExtraction(t *testing.T) {

	// TODO: Make this configuration specific to tests using environment params
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

	article, err := parsers.GetRssArticleFromDatabase(name, url, ctx, driver)
	if err != nil {
		log.Fatal(err)
	}

	if (parsers.RssEntry{}) == article {
		fmt.Println("Empty data returned. No match found")
	} else {
		fmt.Println(article.Id)
		fmt.Println(article.Title)
		fmt.Println(article.Url)
		fmt.Println("------------------------")
	}
}

func TestRssAuthorExtraction(t *testing.T) {
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

	author, err := parsers.GetAuthorFromDatabase(name, ctx, driver)
	if err != nil {
		log.Fatal(err)
	}

	if (parsers.RssAuthor{}) == author {
		fmt.Println("Empty data returned. No match found")
	} else {
		fmt.Println(author.Id)
		fmt.Println(author.Name)
		fmt.Println(author.Email)
		fmt.Println("------------------------")
	}

}

// Testing how the function handler for the endpoint that triggers the ingestion of the Rss Entries and
// Authors:
func TestRssEntryIngestionHandler(t *testing.T) {
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

	env := &Env{Neo4jDriver: driver, Ctx: ctx}
	router := gin.Default()
	router.POST("/rss_feeds/ingest/", env.extractRssFeedEntries)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/rss_feeds/ingest/", nil)
	router.ServeHTTP(w, req)

}
