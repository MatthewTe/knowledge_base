package main

import (
	"context"
	"fmt"
	"knowledge_base/parsers"
	"log"
	"os"

	"testing"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

func TestHTMLContentExtraction(t *testing.T) {

	fmt.Println("------------------------ TestHTMLContentExtraction ----------------------- ")
	fmt.Println("Data Extraction from HTML Page")

	// Creating HTML core component to test all processes:
	testComponent := parsers.HtmlContent{Url: "http://localhost:8000/test/html_page"}

	fmt.Println("\nInital Struct arrays before html processing: ")
	fmt.Println("Image Paths:", testComponent.Images)
	fmt.Println("Table Paths", testComponent.Tables)
	fmt.Println("Snapshot Paths:", testComponent.Snapshot)

	testComponent.LoadHtmlPage()

	fmt.Println("\nStruct arrays after html processing: ")
	fmt.Println("Image Paths:", testComponent.Images)
	fmt.Println("Table Paths", testComponent.Tables)
	fmt.Println("Snapshot Paths:", testComponent.Snapshot)

	assert.Equal(t, len(testComponent.Images), 4)

}

func TestHTMLPageS3Ingestion(t *testing.T) {
	fmt.Println("------------------------- TestHTMLPageS3Ingestion ------------------------ ")
	fmt.Println("Inserting the HTML Page to Storage Bucket")

	// Dev client connection to storage bucket:

	// First load the html content into the struct so we can pass the page to the insertion storage bucket:
	testComponent := parsers.HtmlContent{Url: "http://localhost:8000/test/html_page"}
	testComponent.LoadHtmlPage()

	minioClient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("test_user", "test_password", ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal("Unable to create minio client", err)
	}

	// Extracting the file object:
	object, err := os.Open(testComponent.HtmlPage)
	if err != nil {
		log.Fatal("Error in opening temp test files:", err)
	}

	_, err = parsers.UploadHtmlFileToStatic(
		context.Background(),
		minioClient,
		"test-bucket",
		"html/38_North/test_article.html",
		object,
	)
	if err != nil {
		log.Fatal("Error in inserting html page into the storage bucket:", err)
	}
}

func TestHTMLPageStaticIngestion(t *testing.T) {
	fmt.Println("------------------------ TestHTMLPageStaticIngestion ----------------------- ")

	fmt.Println("HTML Data Ingestion")

	testComponent := parsers.HtmlContent{Url: "http://localhost:8000/test/html_page"}
	testComponent.LoadHtmlPage()

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

}
