package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
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

type Env struct {
	db *sql.DB
}

type ErrorMsg struct {
	Error string
}

type RssFeed struct {
	Id          int    `json:"id"`
	Url         string `json:"url"`
	Title       string `json:"title"`
	Etag        string `json:"etag"`
	LastUpdate  string `json:"last_updated"`
	ExecuteTime string `json:"execute_time"`
}
type RssFeeds struct {
	Entries []RssFeed
}

func (e *Env) getRssFeeds(c *gin.Context) {

	rows, err := e.db.Query("SELECT * FROM rss_feeds;")
	if err != nil {
		log.Fatal("Unable to query rss feeds", err)
		c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
	}
	defer rows.Close()

	var rssFeedArray = []RssFeed{}
	for rows.Next() {
		var id int
		var url, title, etag, lastUpdated, executeTime string
		err = rows.Scan(&id, &url, &title, &etag, &lastUpdated, &executeTime)
		if err != nil {
			log.Fatal(err)
		}

		rssFeed := RssFeed{
			Id:          id,
			Url:         url,
			Title:       title,
			Etag:        etag,
			LastUpdate:  lastUpdated,
			ExecuteTime: executeTime,
		}
		rssFeedArray = append(rssFeedArray, rssFeed)
	}

	c.IndentedJSON(http.StatusOK, rssFeedArray)
}
func (e *Env) postRssFeeds(c *gin.Context) {

	var newRssFeeds RssFeeds
	err := c.BindJSON(&newRssFeeds)
	if err != nil {
		log.Fatal("Error in creating newRssFeed object", err)
		c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
	}

	for i := 0; i < len(newRssFeeds.Entries); i++ {

		rssFeed := newRssFeeds.Entries[i]
		insertFeedStmt, err := e.db.Prepare(`INSERT INTO 
			rss_feeds(url, title, e_tag, last_updated, execute_time)
			VALUES(?, ?, ?, ?, ?)
		`)
		if err != nil {
			log.Fatal("Error in creating newRssFeed object", err)
			c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
		}

		res, err := insertFeedStmt.Exec(
			rssFeed.Url,
			rssFeed.Title,
			rssFeed.Etag,
			rssFeed.LastUpdate,
			rssFeed.ExecuteTime,
		)
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
		}

		rowCnt, err := res.RowsAffected()
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusInternalServerError, ErrorMsg{Error: err.Error()})
		}
		fmt.Println("Num rows inserted", rowCnt)

	}

	c.IndentedJSON(http.StatusOK, newRssFeeds)
}

func (e *Env) extractRssFeedEntries(c *gin.Context) {
	/*
		Steps to extract rss entries:
		1) Extract the rss feed from the POST request.
		2) Extract the rss feed item from the database
		3) Make request to the rss feed url to get the etag or the last updated value.
		4) If the etag or the last updated value indicate that the values have change make a request to extract all of the entries
		5) Extract and iterate through each of the rss feeds:
			6) Save the entry in the database.
			7) Make an http request to the url provided by the rss feed to extract the html file and save it as a static file in a minio storage.
			8) Trigger the nlp process?????
	*/

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

func (e *Env) postRssEntry(c *gin.Context) {

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

	env := &Env{db: db}
	router := gin.Default()

	router.GET("/rss_feeds", env.getRssFeeds)
	router.POST("/rss_feeds", env.postRssFeeds)

	router.GET("/rss_entries/:id", env.getRssEntry)

	router.Run("localhost:8080")

}
