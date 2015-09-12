package main

import (
	"database/sql"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

const BEACON = "R0lGODlhAQABAIAAANvf7wAAACH5BAEAAAAALAAAAAABAAEAAAICRAEAOw=="

var db *sql.DB

type PageView struct {
	Id        int
	Timestamp time.Time
	URL       string
	Referrer  string
	Ip        string
	Domain    string
	Title     string
	Headers   []byte
	UA        string
	Locale    string
}

func (p *PageView) Save() error {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
		return err
	}

	_, err = db.Exec(`INSERT INTO pageview (
		url, referrer, ip, domain,
		title, headers, ua, locale
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
		p.URL, p.Referrer, p.Ip, p.Domain,
		p.Title, p.Headers, p.UA, p.Locale,
	)
	if err != nil {
		log.Fatal(err)
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func init() {
	var err error
	db, err = sql.Open("sqlite3", path.Join("db", "analytics.sqlite3"))
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err = schema(); err != nil {
		log.Fatal(err)
	}
}

func schema() error {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS pageview (
		id			INTEGER NOT NULL,
		timestamp 	TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
		url			TEXT NOT NULL,
		referrer	TEXT NULL,
		ip			VARCHAR(255) NOT NULL,
		domain		VARCHAR(255) NOT NULL,
		title		TEXT NULL,
		headers		TEXT NULL,
		ua			TEXT NULL,
		locale		TEXT NULL,
		PRIMARY KEY(id)
	);`)
	if err != nil {
		log.Fatal(err)
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func main() {
	http.HandleFunc("/b.js", Script)
	http.HandleFunc("/b.gif", Analyze)
	http.HandleFunc("/", NotFound)

	log.Println("Running...")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

func Script(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	w.Header().Set("Content-Type", "text/javascript")
	w.Header().Set("Cache-Control", "private, no-cache")

	fp := path.Join("templates", "b.js")
	tmpl, err := template.ParseFiles(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func Analyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	Params, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Fatal(err)
	}

	RawURL := Params.Get("u")
	if RawURL == "" {
		log.Fatal(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ParsedURL, err := url.Parse(RawURL)
	if err != nil {
		log.Fatal(err)
	}

	Ip := r.Header.Get("X-Forwarded-For")
	if Ip == "" {
		Ip = r.RemoteAddr
	}

	Headers, err := json.Marshal(r.Header)
	if err != nil {
		log.Fatal(err)
	}

	p := &PageView{
		URL:     ParsedURL.String(),
		Domain:  ParsedURL.Host,
		Title:   Params.Get("t"),
		UA:      r.UserAgent(),
		Locale:  r.Header.Get("Accept-Language"),
		Ip:      Ip,
		Headers: Headers,
	}

	if err := p.Save(); err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "private, no-cache")

	beacon, _ := b64.StdEncoding.DecodeString(BEACON)
	fmt.Fprintln(w, string(beacon))
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)

	lp := path.Join("templates", "layout.html")
	fp := path.Join("templates", "404.html")
	tmpl, err := template.ParseFiles(lp, fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
