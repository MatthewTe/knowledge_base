package main

import (
	"context"
	"database/sql"
	"fmt"
	"knowledge_base/parsers"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
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

	var providedTitle RssFeedTitle
	err := c.BindJSON(&providedTitle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
	}

	SummaryResponse, _ := parsers.IngestAllRssItems(providedTitle.Title, e.Ctx, e.Neo4jDriver)
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

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current working directory path", err)
	}
	parentDir := filepath.Dir(wd)
	tempDirPath := filepath.Join(parentDir, "temp")

	if _, err := os.Stat(tempDirPath); os.IsNotExist(err) {
		err := os.Mkdir(tempDirPath, os.ModePerm)
		if err != nil {
			fmt.Println("Error creating temp directory:", err)
		} else {
			fmt.Println("Temp Directory created:", tempDirPath)
		}
	} else {
		fmt.Println("Temp Directory already exists:", tempDirPath)
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
