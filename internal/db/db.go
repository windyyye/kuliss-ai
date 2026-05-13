package db

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Connect(dbType string, dsn string) {
	var err error

	if dbType == "sqlite" {
		DB, err = sql.Open("sqlite3", dsn)
	} else {
		DB, err = sql.Open("postgres", dsn)
	}

	if err != nil {
		log.Fatalf("db açılmadı: %v", err)
	}

	const maxAttempts = 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = DB.Ping(); err == nil {
			break
		}
		if attempt == maxAttempts {
			log.Fatalf("db ye ulaşılamıyor: %v", err)
		}
		log.Printf("db hazır değil (deneme %d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(time.Second)
	}

	log.Printf("Veritabanı bağlandı. Yüklendi (%s)\n", dbType)

	if dbType == "sqlite" {
		initSQLite()
	}
}

func initSQLite() {
	query := `
	CREATE TABLE IF NOT EXISTS contacts (
		phone TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		phone TEXT,
		role TEXT,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(phone) REFERENCES contacts(phone)
	);`
	_, err := DB.Exec(query)
	if err != nil {
		log.Fatalf("SQLite tabloları oluşturulamadı: %v", err)
	}

	DB.Exec(`ALTER TABLE contacts ADD COLUMN profile_pic TEXT DEFAULT ''`)
	DB.Exec(`ALTER TABLE contacts ADD COLUMN contact_name TEXT DEFAULT ''`)
}

func SaveMessage(phone, pushName, role, content string) error {

	_, err := DB.Exec(`INSERT INTO contacts (phone, contact_name) VALUES (?, ?) 
	ON CONFLICT (phone) DO UPDATE SET contact_name = CASE WHEN excluded.contact_name != '' THEN excluded.contact_name ELSE contact_name END`, phone, pushName)
	if err != nil {
		return err
	}

	_, err = DB.Exec(
		`INSERT INTO conversations (phone, role, content) VALUES (?, ?, ?)`,
		phone, role, content,
	)
	return err
}

func SaveContactProfilePic(phone, url string) error {
	if DB == nil {
		return nil
	}
	_, err := DB.Exec(`UPDATE contacts SET profile_pic = ? WHERE phone = ?`, url, phone)
	return err
}

func GetHistory(phone string, limit int) ([]map[string]string, error) {
	rows, err := DB.Query(`
	SELECT role, content FROM conversations
	WHERE phone = ?
	ORDER BY created_at ASC
	LIMIT ?
	`, phone, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []map[string]string
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, err
		}
		messages = append(messages, map[string]string{
			"role":    role,
			"content": content,
		})
	}

	return messages, nil
}

func IsBlocked(phone string) bool {
	if DB == nil {
		return false
	}
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM blocked_contacts WHERE phone = ?)`
	err := DB.QueryRow(query, phone).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}
