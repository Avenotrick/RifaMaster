package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("error abriendo db: %w", err)
	}

	// WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("error setting WAL: %w", err)
	}
	// Busy timeout
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("error setting busy timeout: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creando tablas: %w", err)
	}

	if err := initNumbers(db); err != nil {
		return nil, fmt.Errorf("error inicializando números: %w", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS numbers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			number INTEGER NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'available',
			buyer_name TEXT,
			buyer_email TEXT,
			payment_id TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS payments (
			id TEXT PRIMARY KEY,
			preference_id TEXT,
			number INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			buyer_name TEXT NOT NULL,
			buyer_email TEXT NOT NULL,
			amount REAL NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
	}

	for _, q := range tables {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func initNumbers(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM numbers").Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, err := tx.Prepare("INSERT INTO numbers (number, status) VALUES (?, 'available')")
		if err != nil {
			return err
		}
		defer stmt.Close()

		for n := 1; n <= 100; n++ {
			if _, err := stmt.Exec(n); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}
		log.Println("100 números inicializados")
	}
	return nil
}

func GetAllNumbers(db *sql.DB) ([]Number, error) {
	rows, err := db.Query(`
		SELECT id, number, status, COALESCE(buyer_name,''), created_at, updated_at
		FROM numbers ORDER BY number ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var numbers []Number
	for rows.Next() {
		var n Number
		if err := rows.Scan(&n.ID, &n.Number, &n.Status, &n.BuyerName, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		numbers = append(numbers, n)
	}
	return numbers, rows.Err()
}

func GetNumber(db *sql.DB, num int) (*Number, error) {
	var n Number
	err := db.QueryRow(`
		SELECT id, number, status, COALESCE(buyer_name,''), created_at, updated_at
		FROM numbers WHERE number = ?
	`, num).Scan(&n.ID, &n.Number, &n.Status, &n.BuyerName, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func ReserveNumber(db *sql.DB, num int, buyerName, buyerEmail, paymentID string) error {
	result, err := db.Exec(`
		UPDATE numbers SET status = 'reserved', buyer_name = ?, buyer_email = ?, payment_id = ?, updated_at = datetime('now')
		WHERE number = ? AND status = 'available'
	`, buyerName, buyerEmail, paymentID, num)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("número %d no disponible", num)
	}
	return nil
}

func ConfirmPayment(db *sql.DB, externalRef string) error {
	result, err := db.Exec(`
		UPDATE numbers SET status = 'sold', updated_at = datetime('now')
		WHERE payment_id = ? AND status = 'reserved'
	`, externalRef)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no se encontró reserva para el pago %s", externalRef)
	}

	_, err = db.Exec("UPDATE payments SET status = 'approved', updated_at = datetime('now') WHERE id = ?", externalRef)
	return err
}

func RejectPayment(db *sql.DB, externalRef string) error {
	_, err := db.Exec("UPDATE payments SET status = 'rejected', updated_at = datetime('now') WHERE id = ?", externalRef)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE numbers SET status = 'available', buyer_name = NULL, buyer_email = NULL, payment_id = NULL, updated_at = datetime('now')
		WHERE payment_id = ? AND status = 'reserved'
	`, externalRef)
	return err
}

func CreatePayment(db *sql.DB, p *Payment) error {
	_, err := db.Exec(`
		INSERT INTO payments (id, preference_id, number, status, buyer_name, buyer_email, amount)
		VALUES (?, ?, ?, 'pending', ?, ?, ?)
	`, p.ID, p.PreferenceID, p.Number, p.BuyerName, p.BuyerEmail, p.Amount)
	return err
}

func GetPayment(db *sql.DB, id string) (*Payment, error) {
	var p Payment
	err := db.QueryRow(`
		SELECT id, COALESCE(preference_id,''), number, status, buyer_name, buyer_email, amount, created_at
		FROM payments WHERE id = ?
	`, id).Scan(&p.ID, &p.PreferenceID, &p.Number, &p.Status, &p.BuyerName, &p.BuyerEmail, &p.Amount, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
