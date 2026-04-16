package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
	tele "gopkg.in/telebot.v4"

	"github.com/chivta/spotify-community-updates/internal/migrations"
)

func main() {
	b, err := tele.NewBot(tele.Settings{
		Token:  os.Getenv("BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		panic(err)
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		panic(err)
	}

	err = RunMigrations(context.Background(), conn)
	if err != nil {
		panic(err)
	}

	b.Handle("/start", func(c tele.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := conn.Exec(ctx, `INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING`, c.Sender().ID)
		if err != nil {
			log.Printf("Failed to register user %d: %v", c.Sender().ID, err)
			return c.Send("Something went wrong.")
		}
		return c.Send("Hello, I'm a bot that will send you updates about Spotify Community.")
	})

	go b.Start()

	http.HandleFunc("/broadcast", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var updates []Post
		err := json.NewDecoder(r.Body).Decode(&updates)
		if err != nil {
			log.Printf("Failed to decode request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		err = BroadcastCommunityUpdate(conn, updates, b)
		if err != nil {
			log.Printf("Failed to broadcast update: %v", err)
			http.Error(w, "Failed to broadcast update", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Update broadcasted successfully"))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("Bot is running...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Post struct {
	Title   string `json:"title"`
	Date    string `json:"date"`
	Summary string `json:"summary"`
	Slug    string `json:"slug"`
}

func BroadcastCommunityUpdate(conn *pgx.Conn, update []Post, b *tele.Bot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := conn.Query(ctx, `SELECT id FROM users`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			log.Printf("Failed to scan user ID: %v", err)
			continue
		}
		recipient := &tele.User{ID: userID}
		for _, post := range update {
			postMessage := fmt.Sprintf(
				"%s\n%s\n%s\nhttps://developer.spotify.com/community",
				post.Title, post.Summary, post.Date)
			if _, err := b.Send(recipient, postMessage); err != nil {
				log.Printf("Failed to send update to user %d: %v", userID, err)
			}
		}
	}

	return rows.Err()
}

func RunMigrations(ctx context.Context, conn *pgx.Conn) error {
	m, err := migrate.NewMigrator(ctx, conn, "public.schema_version")
	if err != nil {
		return err
	}
	err = m.LoadMigrations(migrations.FS)
	if err != nil {
		return err
	}
	return m.Migrate(ctx)
}
