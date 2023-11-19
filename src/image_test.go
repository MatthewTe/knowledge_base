package main

import (
	"context"
	"fmt"
	"knowledge_base/parsers"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestHTMLImageStaticIngestion(t *testing.T) {
	fmt.Println("------------------------ TestHTMLImageStaticIngestion ----------------------- ")

	fmt.Println("HTML Data Ingestion")

	testComponent := parsers.HtmlContent{Url: "http://localhost:8000/test/html_page"}
	testComponent.LoadHtmlPage()

	err := godotenv.Load("../data/test.env")
	if err != nil {
		log.Fatal("Unable to load environment variable for tests", err)
	}

	minioClient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("test_user", "test_password", ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal("Unable to create minio client", err)
	}

	for i := 0; i < len(testComponent.Images); i++ {
		imgPath := testComponent.Images[i]
		baseImgName := filepath.Base(imgPath)

		log.Println("Testing upload image for img", imgPath)

		object, err := os.Open(imgPath)
		if err != nil {
			log.Fatal("Error in opening temp test files:", err)
		}

		_, err = parsers.UploadImageFileToStatic(
			context.Background(),
			minioClient,
			"test-bucket",
			filepath.Join("image", "38_North", baseImgName),
			object,
		)
		if err != nil {
			log.Fatal("Error in inserting html page into the storage bucket:", err)
		}

	}
}
