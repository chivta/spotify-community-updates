package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	spotifyCommunityURL = "https://developer.spotify.com/_next/data/B5Xg_Lj2Q5kwbhVosO17X/community.json"
)

type data struct {
	PageProps struct {
		Posts []Post
	} `json:"pageProps"`
}
type Post struct {
	Title   string `json:"title"`
	Date    string `json:"date"`
	Summary string `json:"summary"`
	Slug    string `json:"slug"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, os.Getenv("DB_URL"))
	if err != nil {
		panic(err)
	}

	resp, err := http.Get(spotifyCommunityURL)
	if err != nil {
		log.Fatalf("Failed to fetch Spotify Community page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	var result data
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
	}

	unsentPosts := filterUnsentUpdates(result.PageProps.Posts, conn)
	body, err := json.Marshal(unsentPosts)
	if err != nil {
		log.Fatalf("Failed to marshal updates: %v", err)
	}
	resp, err = http.Post(os.Getenv("BOT_BROADCAST_URL"), "application/json",bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Failed to send updates to bot: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Bot responded with status code: %d", resp.StatusCode)
	}

	markUpdatesAsSent(unsentPosts, conn)

	log.Printf("Successfully sent %d updates to bot", len(unsentPosts))
}

func filterUnsentUpdates(posts []Post, conn *pgx.Conn) []Post {
	var unsent []Post

	for _, post := range posts {
		var exists bool
		err := conn.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM updates WHERE slug = $1)`, post.Slug).Scan(&exists)
		if err != nil {
			log.Printf("Failed to check update %s: %v", post.Slug, err)
			continue
		}
		if !exists {
			unsent = append(unsent, post)
		}
	}

	return unsent
}

func markUpdatesAsSent(posts []Post, conn *pgx.Conn) {
	for _, post := range posts {
		_, err := conn.Exec(context.Background(), `INSERT INTO updates (slug, title) VALUES ($1, $2) ON CONFLICT DO NOTHING`, post.Slug, post.Title)
		if err != nil {
			log.Printf("Failed to mark update %s as sent: %v", post.Slug, err)
		}
	}
}
