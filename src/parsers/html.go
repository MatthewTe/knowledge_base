package parsers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gocolly/colly/v2"
	"golang.org/x/net/html"
)

type HtmlContent struct {
	Url      string
	HtmlPage *html.Node
	Tables   [][]byte
	Images   [][]byte
	Snapshot []byte
}

func (htmlContent *HtmlContent) LoadHtmlPage() {
	c := colly.NewCollector()

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
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Making Request to ", r.URL)
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})
	c.Visit(htmlContent.Url)
}

// Refactor this to use Colly. I can save the whole html page to a temp dir by
// accessing the string param of the Colly callback. Add a callback for the main <html> body
// that extracts the full html page as a string. Load it and save it to a temp dir and then
// upload to minio.
// https://okanexe.medium.com/colly-web-scraping-in-golang-a-practical-tutorial-for-beginners-6e35cb3bd608
