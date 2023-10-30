package main

import (
	"context"
	"database/sql"
	"fmt"
	"knowledge_base/parsers"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func setupDatabase(db *sql.DB, dbPath string, rebuild bool) (*sql.DB, error) {

	// Replaces the database connection after removing the database:
	if rebuild {
		db.Close()
		err := os.Remove(dbPath)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Fatal("Error in re-creating a database", err)
			return nil, err
		}
	}

	// Creating schema Transactions:
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer tx.Rollback()

	CreateRssFeedStmt, err := tx.Prepare(`CREATE TABLE IF NOT EXISTS rss_feeds (
		pk INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT,
		title TEXT,
		e_tag TEXT,
		last_updated TEXT,
		execute_time TEXT);
	`)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	createRssFeedEntryStmt, err := tx.Prepare(`CREATE TABLE IF NOT EXISTS rss_entries (
		pk INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT,
		title TEXT,
		description TEXT,
		rss_feed_id INTEGER,
		date_posted TEXT,
		date_extracted TEXT,
		in_storage INTEGER,
		storage_inserted_on TEXT,
		FOREIGN KEY (rss_feed_id)
			REFERENCES rss_feeds (id));
	`)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	_, err = CreateRssFeedStmt.Exec()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	_, err = createRssFeedEntryStmt.Exec()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	return db, nil
}

// func setupGraphDb()

type Env struct {
	db          *sql.DB
	Neo4jDriver neo4j.DriverWithContext
	Ctx         context.Context
}

type ErrorMsg struct {
	Error string
}

func (e *Env) getRssFeeds(c *gin.Context) {

	results, err := neo4j.ExecuteQuery(
		e.Ctx,
		e.Neo4jDriver,
		"MATCH (feeds:Rss_Feed:Source) return feeds",
		nil,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		log.Fatal(err)
		c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
	}

	var rssFeedArray = []parsers.RssFeed{}
	for _, record := range results.Records {

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

				rssFeedArray = append(rssFeedArray, rssFeed)
			}
		}
	}
	c.IndentedJSON(http.StatusOK, rssFeedArray)
}
func (e *Env) postRssFeeds(c *gin.Context) {

	var newRssFeeds parsers.RssFeeds
	err := c.BindJSON(&newRssFeeds)
	if err != nil {
		log.Fatal("Error in creating newRssFeed object", err)
		c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
	}

	var insertedRssFeeds []parsers.RssFeed

	for i := 0; i < len(newRssFeeds.Entries); i++ {

		rssFeed := newRssFeeds.Entries[i]

		result, err := neo4j.ExecuteQuery(
			e.Ctx,
			e.Neo4jDriver,
			`MERGE (rss_feed:Rss_Feed:Source {name: $name})
			ON CREATE SET
				rss_feed.url = $url,
				rss_feed.scheduled_time = $scheduled_time,
				rss_feed.etag = $etag,
				rss_feed.last_updated = $last_updated,
				rss_feed.created = timestamp()
			RETURN rss_feed`,
			map[string]any{
				"name":           rssFeed.Title,
				"url":            rssFeed.Url,
				"scheduled_time": rssFeed.ExecuteTime,
				"etag":           rssFeed.Etag,
				"last_updated":   rssFeed.LastUpdate,
			},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase("neo4j"))
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
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

					insertedRssFeeds = append(insertedRssFeeds, rssFeed)
				}
			}
		}
	}

	c.IndentedJSON(http.StatusOK, insertedRssFeeds)
}

func (e *Env) extractRssFeedEntries(c *gin.Context) {

	type RssFeedTitle struct {
		Title string `json:"title"`
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
		RssFeed    parsers.RssFeed             `json:"source_feed"`
		RssEntries []RssEntryExtractionSummary `json:"entries"`
	}

	var providedTitle RssFeedTitle
	err := c.BindJSON(&providedTitle)
	if err != nil {
		log.Fatal(err)
		c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
	}

	var SummaryResponse RssFeedExtractionSummary

	SummaryResponse.Title = providedTitle.Title

	// 1) Query the database for the graph node of rss feed source based on title.
	extractedRssFeed, err := parsers.GetRssSourceFromDatabase(providedTitle.Title, e.Ctx, e.Neo4jDriver)
	if err != nil {
		SummaryResponse.Error = err.Error()
		SummaryResponse.Status = "Error in extracting an Rss Source from database"
		c.JSON(http.StatusInternalServerError, SummaryResponse)
	}
	SummaryResponse.Id = extractedRssFeed.Id
	SummaryResponse.RssFeed = extractedRssFeed

	// 2) Make a post request to the rss feed endpoint based on the field extracted by the node.
	resp, err := http.Get(extractedRssFeed.Url)
	if err != nil {
		log.Fatal(err)
		c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
	}
	defer resp.Body.Close()

	if resp.StatusCode > 300 {
		err = fmt.Errorf("request to rss feed %s returned status code: %d", extractedRssFeed.Url, resp.StatusCode)
		SummaryResponse.Error = err.Error()
		SummaryResponse.Status = "Error in extracting an Rss Source from database"
		SummaryResponse.RssFeed = extractedRssFeed
		c.JSON(http.StatusInternalServerError, SummaryResponse)
	}

	fp := gofeed.NewParser()
	feed, _ := fp.Parse(resp.Body)

	// -- If I am going to refactor this to make this modular I would do this here and make a function that takes -- // in an io.Reader that represents the rss feed.  // Creating Rss feed struct from the response body io.Reader: fp := gofeed.NewParser() feed, _ := fp.Parse(resp.Body) // Determining if the Etag or the Last Updated Date value exists in the rss feed:
	// TODO: Support Etags in addition to the Last Updated value
	if extractedRssFeed.LastUpdate == feed.Updated {

		noUpdatedRssFeedMsg := fmt.Sprintf(`
		No new Rss Feed found for %s rss feed. DatabaseLastUpdated: %s, RequestLastUpdated %s`,
			extractedRssFeed.Title,
			extractedRssFeed.LastUpdate,
			feed.Updated,
		)

		SummaryResponse.Status = noUpdatedRssFeedMsg

		c.JSON(http.StatusFound, SummaryResponse)
	}

	var EntrySummaryArray []RssEntryExtractionSummary

	for _, item := range feed.Items {

		var EntrySummary RssEntryExtractionSummary

		existingEntry, err := parsers.GetRssArticleFromDatabase(item.Title, item.Link, e.Ctx, e.Neo4jDriver)
		EntrySummary.Title = item.Title
		EntrySummary.Url = item.Link
		if err != nil {
			EntrySummary.Error = err.Error()
			EntrySummary.Status = "Error in querying articles from the database"
			EntrySummaryArray = append(EntrySummaryArray, EntrySummary)
			continue
		}
		// If the article already exists then we are done we don't have to continue to process this entry.
		if (parsers.RssEntry{}) != existingEntry {
			EntrySummary.Id = existingEntry.Id
			EntrySummary.Error = ""
			EntrySummary.Status = "Article already exists in the database. Skipped all functions assocaited with this Entry"
			EntrySummaryArray = append(EntrySummaryArray, EntrySummary)
			fmt.Println(err)
			continue
		}

		// Inserting the Entry into the Graph database:
		result, err := neo4j.ExecuteQuery(
			e.Ctx,
			e.Neo4jDriver,
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

			extractedAuthor, err := parsers.GetAuthorFromDatabase(author.Name, e.Ctx, e.Neo4jDriver)
			if err != nil {

				AuthorSummary.Name = author.Name
				AuthorSummary.Error = err.Error()
				AuthorSummary.Status = "Error in querying authors from the database"
				AuthorSummaryArray = append(AuthorSummaryArray, AuthorSummary)
				continue
			}

			// Ingestion logic if Author is not unique in db:
			if (parsers.RssAuthor{}) != extractedAuthor {

				AuthorSummary.Name = extractedAuthor.Name
				AuthorSummary.Status = "Existing author detected - adding connection to an existing Author"
				AuthorSummary.Id = extractedAuthor.Id

				// Article already exists so we just create a connection between it and the Article:
				connectionResult, err := neo4j.ExecuteQuery(
					e.Ctx,
					e.Neo4jDriver,
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
					e.Ctx,
					e.Neo4jDriver,
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
	c.JSON(http.StatusCreated, SummaryResponse)
}

type RssUrlEntry struct {
	Id string `uri:"id"`
}

func (e *Env) getRssEntry(c *gin.Context) {
	var urlEntry RssUrlEntry
	err := c.ShouldBindUri(&urlEntry)
	if err != nil {
		log.Fatal(err)
		c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
	}

	fmt.Println(urlEntry.Id)

}

func main() {

	dbPath := "./test.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Error opening the database", err)
	}

	// Setting up Database:
	db, err = setupDatabase(db, dbPath, true)
	if err != nil {
		log.Fatal("Error in setting up the database", err)
	}

	ctx := context.Background()
	dbUri := "neo4j://localhost"
	dbUser := "neo4j"
	dbPassword := "Entropy_Investments"
	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))
	if err != nil {
		log.Fatal(nil)
	}
	defer driver.Close(ctx)

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		log.Fatal(err)
	}

	env := &Env{db: db, Neo4jDriver: driver, Ctx: ctx}
	router := gin.Default()

	router.GET("/rss_feeds", env.getRssFeeds)
	router.POST("/rss_feeds", env.postRssFeeds)
	router.POST("/rss_feeds/ingest/", env.extractRssFeedEntries)

	router.GET("/rss_entries/:id", env.getRssEntry)

	router.Run("localhost:8080")

}
