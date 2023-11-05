package parsers

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

// This is the function that gets called with a RssFeed title and performs all of the ingestion activities in the database:
// It wraps all of the previously existing logic in the rss parser:
func IngestAllRssItems(rssFeedTitle string, ctx context.Context, driver neo4j.DriverWithContext) (SummaryResponse RssFeedExtractionSummary, err error) {

	// Generic JSON response struct that summarizes the status of the rss ingestion:
	SummaryResponse.Title = rssFeedTitle

	// 1) Query the database for the graph node of rss feed source based on title.
	extractedRssFeed, err := GetRssSourceFromDatabase(rssFeedTitle, ctx, driver)
	if err != nil {
		SummaryResponse.Error = err.Error()
		SummaryResponse.Status = "Error in extracting an Rss Source from database"
		return
	}
	SummaryResponse.Id = extractedRssFeed.Id
	SummaryResponse.RssFeed = extractedRssFeed

	// 2) Make a post request to the rss feed endpoint based on the field extracted by the node.
	resp, err := http.Get(extractedRssFeed.Url)
	if err != nil {
		SummaryResponse.Error = err.Error()
		SummaryResponse.Status = "Error in extracting an Rss Source from database"
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode > 300 {
		err = fmt.Errorf("request to rss feed %s returned status code: %d", extractedRssFeed.Url, resp.StatusCode)
		SummaryResponse.Error = err.Error()
		SummaryResponse.Status = "Error in extracting an Rss Source from database"
		SummaryResponse.RssFeed = extractedRssFeed
		return
	}

	fp := gofeed.NewParser()
	feed, _ := fp.Parse(resp.Body)

	if extractedRssFeed.LastUpdate == feed.Updated {

		noUpdatedRssFeedMsg := fmt.Sprintf(`
		No new Rss Feed found for %s rss feed. DatabaseLastUpdated: %s, RequestLastUpdated %s`,
			extractedRssFeed.Title,
			extractedRssFeed.LastUpdate,
			feed.Updated,
		)

		SummaryResponse.Status = noUpdatedRssFeedMsg
		return
	}

	var EntrySummaryArray []RssEntryExtractionSummary

	for _, item := range feed.Items {

		var EntrySummary RssEntryExtractionSummary

		existingEntry, err := GetRssArticleFromDatabase(item.Title, item.Link, ctx, driver)
		EntrySummary.Title = item.Title
		EntrySummary.Url = item.Link
		if err != nil {
			EntrySummary.Error = err.Error()
			EntrySummary.Status = "Error in querying articles from the database"
			EntrySummaryArray = append(EntrySummaryArray, EntrySummary)
			continue
		}
		// If the article already exists then we are done we don't have to continue to process this entry.
		if (RssEntry{}) != existingEntry {
			EntrySummary.Id = existingEntry.Id
			EntrySummary.Error = ""
			EntrySummary.Status = "Article already exists in the database. Skipped all functions assocaited with this Entry"
			EntrySummaryArray = append(EntrySummaryArray, EntrySummary)
			fmt.Println(err)
			continue
		}

		// Inserting the Entry into the Graph database:
		result, err := neo4j.ExecuteQuery(
			ctx,
			driver,
			`
			CREATE (article:Rss_Feed:Article {
				name: $name,
				url: $url,
				description: $description,
				date_posted: $date_posted,
				static_file_url: $static_file_url,
				in_static_file_storage: $in_static_file_storage,
				created: timestamp()
			})
			WITH article

			MATCH (source:Rss_Feed:Source {name: $rss_source_name})

			CREATE (source)-[rel:CONTAINS_ARTICLE]->(article)
			SET rel.date_downloaded = $downloaded_date
			return article
			`,
			map[string]any{
				"name":                   item.Title,
				"url":                    item.Link,
				"description":            item.Description,
				"date_posted":            item.Published,
				"static_file_url":        "",
				"in_static_file_storage": 0,
				"downloaded_date":        time.Now().Format("2006-01-02"),
				"rss_source_name":        extractedRssFeed.Title,
			},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase("neo4j"))
		if err != nil {
			EntrySummary.Error = err.Error()
			EntrySummary.Status = "Unable to execute neo4j query to insert new article into the database. Skipped all functions assocaited with this Entry"
			EntrySummaryArray = append(EntrySummaryArray, EntrySummary)
			fmt.Println(err)
			continue
		}

		insertedNodeDict := result.Records[0].AsMap()

		for _, nodeKey := range insertedNodeDict {
			switch node := nodeKey.(type) {
			case neo4j.Node:
				EntrySummary.Id = node.ElementId
			}
		}

		fmt.Printf(
			"Created %v nodes in %+v. \n",
			result.Summary.Counters().NodesCreated(),
			result.Summary.ResultAvailableAfter(),
		)

		EntrySummary.Status = "Successfully inserted the Article. Check Author for futher information about Author connections."

		// Extracting the article's Author:
		var AuthorSummaryArray []RssAuthorExtractionSummary

		for _, author := range item.Authors {

			var AuthorSummary RssAuthorExtractionSummary

			extractedAuthor, err := GetAuthorFromDatabase(author.Name, ctx, driver)
			if err != nil {

				AuthorSummary.Name = author.Name
				AuthorSummary.Error = err.Error()
				AuthorSummary.Status = "Error in querying authors from the database"
				AuthorSummaryArray = append(AuthorSummaryArray, AuthorSummary)
				continue
			}

			// Ingestion logic if Author is not unique in db:
			if (RssAuthor{}) != extractedAuthor {

				AuthorSummary.Name = extractedAuthor.Name
				AuthorSummary.Status = "Existing author detected - adding connection to an existing Author"
				AuthorSummary.Id = extractedAuthor.Id

				// Article already exists so we just create a connection between it and the Article:
				connectionResult, err := neo4j.ExecuteQuery(
					ctx,
					driver,
					`
					MATCH (article:Rss_Feed:Article {name: $article_name, url: $article_url})
					MATCH (author:Rss_Feed:Author:Person {name: $author_name})

					CREATE (author)-[:WROTE]->(article)

					RETURN article, author
					`,
					map[string]any{
						"article_name": item.Title,
						"article_url":  item.Link,
						"author_name":  extractedAuthor.Name,
					},
					neo4j.EagerResultTransformer,
					neo4j.ExecuteQueryWithDatabase("neo4j"))

				if err != nil {
					AuthorSummary.Error = err.Error()
					AuthorSummary.Status = fmt.Sprintf("Error in connecting existing author to the article. Author: %s. Article: %s. Skipping addition author logic",
						extractedAuthor.Name,
						item.Title,
					)

					AuthorSummaryArray = append(AuthorSummaryArray, AuthorSummary)
					fmt.Println(err)
					continue
				}
				fmt.Printf(
					"Created %v nodes in %+v. \n",
					connectionResult.Summary.Counters().NodesCreated(),
					connectionResult.Summary.ResultAvailableAfter(),
				)
			} else {
				// Ingestion logic if article is unique:
				AuthorSummary.Name = author.Name
				AuthorSummary.Status = "New author detected - Creating a new author and connecting it to article"

				// Inserting the author into the database and creating the connection:
				authorCreationResult, err := neo4j.ExecuteQuery(
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
						"article_name": item.Title,
						"article_url":  item.Link,
						"author_name":  author.Name,
						"author_email": author.Email,
					},
					neo4j.EagerResultTransformer,
					neo4j.ExecuteQueryWithDatabase("neo4j"))
				if err != nil {
					AuthorSummary.Error = err.Error()
					AuthorSummary.Status = fmt.Sprintf(
						`Error in creating and connecting author to the article. Author: %s. Article: %s. Skipping addition author logic`,
						extractedAuthor.Name,
						item.Title,
					)
					AuthorSummaryArray = append(AuthorSummaryArray, AuthorSummary)
					fmt.Println(err)
					continue
				}

				insertedNodeDict := authorCreationResult.Records[0].AsMap()

				for _, nodeKey := range insertedNodeDict {
					switch node := nodeKey.(type) {
					case neo4j.Node:
						AuthorSummary.Id = node.ElementId
					}
				}

				fmt.Printf(
					"Created %v nodes in %+v. \n",
					authorCreationResult.Summary.Counters().NodesCreated(),
					authorCreationResult.Summary.ResultAvailableAfter(),
				)
			}

			AuthorSummaryArray = append(AuthorSummaryArray, AuthorSummary)

		}

		EntrySummary.Authors = AuthorSummaryArray
		EntrySummaryArray = append(EntrySummaryArray, EntrySummary)
	}

	// Update the Rss Entry Item in the database to set the Last Updated or E-Tag date:
	SummaryResponse.Status = "Article and Author Ingestion complete for the feed"
	SummaryResponse.RssEntries = EntrySummaryArray

	// Updating the Rss Feed item in the database with a new last update value:
	_, err = neo4j.ExecuteQuery(
		ctx,
		driver,
		`
		MERGE (rss_feed:Rss_Feed:Source {name: $name})
		SET rss_feed.last_updated = $last_updated
		RETURN rss_feed`,
		map[string]any{
			"name":         extractedRssFeed.Title,
			"last_updated": feed.Updated,
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		SummaryResponse.Error = err.Error()
		SummaryResponse.Status = "Unable to update the Rss Feeds' last_updated value from the extracted rss feed"
		return
	}

	return

}
