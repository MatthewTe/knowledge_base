package parsers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/minio/minio-go/v7"
	"golang.org/x/net/context"
)

type HtmlContent struct {
	Url      string
	HtmlPage string
	Tables   []string
	Images   []string
	Snapshot []string
}

func (htmlContent *HtmlContent) LoadHtmlPage() {
	c := colly.NewCollector()

	// Load the whole HTML page:
	c.OnHTML("html", func(e *colly.HTMLElement) {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Could not get current working directory path", err)
		}

		parentDir := filepath.Dir(wd)
		tempFileName := filepath.Join(parentDir, "temp", "test_file.html")

		err = os.WriteFile(tempFileName, e.Response.Body, 0666)
		if err != nil {
			log.Fatal("Unable to write extracted text to html file in temp dir", err)
		} else {
			fmt.Println("Wrote ", tempFileName, "to temporary file system storage")

			// Finally add the uploaded html to the struct array:
			htmlContent.HtmlPage = tempFileName

		}

	})

	// Extract images from an html:
	c.OnHTML("body img", func(e *colly.HTMLElement) {
		imagePath := e.Attr("src")

		fmt.Println(imagePath)

		resp, err := http.Get(imagePath)
		if err != nil {
			log.Fatal("Error downloading image:", err)
		}
		defer resp.Body.Close()
		imageData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("Could not extract image data into byte array", err)
		}

		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Could not get current working directory path", err)
		}

		// Read the image data
		fileName := extractFileName(imagePath)
		parentDir := filepath.Dir(wd)
		tempFileName := filepath.Join(parentDir, "temp", fileName)
		err = os.WriteFile(tempFileName, imageData, 0666)
		if err != nil {
			log.Fatal("Error reading image data into temp directory", err)
		} else {
			fmt.Println("Wrote", tempFileName, "to temporary file system storage.")

			// Finally add the uploaded image to the struct array:
			htmlContent.Images = append(htmlContent.Images, tempFileName)
		}

	})

	// Extracting tables from html page:

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Making Request to ", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})
	c.Visit(htmlContent.Url)
}

func extractFileName(url string) string {
	// Remove any query parameters or fragments from the URL
	url = strings.Split(url, "?")[0]
	url = strings.Split(url, "#")[0]

	// Use filepath.Base to get the base component of the path
	return filepath.Base(url)
}

func UploadHtmlFileToStatic(ctx context.Context, minioClient *minio.Client, bucketName string, bucketFilePath string, reader *os.File) (string, error) {

	// Creating the client object:
	err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})

	// First we create the bucket if it doens't exist:
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("Bucket already exists, skipping bucket creation")
		} else {
			log.Println("Error in creating bucket in Minio", bucketName, err)
			return bucketFilePath, err
		}
	} else {
		log.Println("Successfully created: ", bucketName)
	}

	// Calculating the size of the byte array to be uploaded:
	objectStat, err := reader.Stat()
	if err != nil {
		log.Println("Error in calculating the statistics for the file", err)
		return bucketFilePath, err
	}

	info, err := minioClient.PutObject(
		ctx,
		bucketName,
		bucketFilePath,
		reader,
		objectStat.Size(),
		minio.PutObjectOptions{ContentType: "text/html"},
	)
	if err != nil {
		log.Println("Unable to insert the html page into the s3 bucket", err)
	} else {
		log.Println("Inserted html", bucketFilePath, "of size:", info.Size, "bytes", "successfully into bucket", bucketName)
	}

	return bucketFilePath, err
}

func UploadHtmlFileToGraph() {

}

// Refactor this to use Colly. I can save the whole html page to a temp dir by
// accessing the string param of the Colly callback. Add a callback for the main <html> body
// that extracts the full html page as a string. Load it and save it to a temp dir and then
// upload to minio.
// https://okanexe.medium.com/colly-web-scraping-in-golang-a-practical-tutorial-for-beginners-6e35cb3bd608
