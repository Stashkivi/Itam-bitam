// Package cache provides a durable local store for scan payloads that cannot
// be delivered to the server immediately. It uses SQLite (pure-Go, no CGo)
// with per-record AES-256-GCM encryption so data at rest is protected even
// if the host filesystem is read by an unauthorized party.
package cache

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS pending_payloads (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    collected_at INTEGER NOT NULL,
    payload_type TEXT    NOT NULL CHECK(payload_type IN ('full_scan','diff','heartbeat','alert')),
    payload_blob BLOB    NOT NULL,
    attempts     INTEGER NOT NULL DEFAULT 0,
    last_attempt INTEGER,
    checksum     TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS dead_letter (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    original_id INTEGER,
    failed_at   INTEGER NOT NULL,
    reason      TEXT,
    payload_blob BLOB   NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pending_fifo     ON pending_payloads(collected_at ASC);
CREATE INDEX IF NOT EXISTS idx_pending_attempts ON pending_payloads(attempts);
`

// Store is a thread-safe SQLite-backed payload queue.
type Store struct {
	db        *sql.DB
	gcmKey    []byte // 32-byte AES-256 key derived at construction time
	maxRows   int
	maxTries  int
	batchSize int
}

// PendingRow is one row read back from the queue, already decrypted.
type PendingRow struct {
	ID          int64
	CollectedAt time.Time
	PayloadType string
	Payload     json.RawMessage
}

// Open opens (or creates) the SQLite cache at path. key is the 32-byte
// AES-256 encryption key; derive it with DeriveKey before calling Open.
func Open(path string, key []byte, maxRows, maxTries, batchSize int) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("cache open: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite writer serialisation
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("cache schema: %w", err)
	}
	return &Store{
		db:        db,
		gcmKey:    key,
		maxRows:   maxRows,
		maxTries:  maxTries,
		batchSize: batchSize,
	}, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

// Push encrypts and enqueues a payload. If the queue is at capacity, the
// oldest row is evicted to maintain the maxRows bound.
func (s *Store) Push(ctx context.Context, payloadType string, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	checksum := sha256hex(raw)
	blob, err := s.encrypt(raw)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Evict oldest rows when at capacity.
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM pending_payloads
		WHERE id IN (
			SELECT id FROM pending_payloads
			ORDER BY collected_at ASC
			LIMIT MAX(0, (SELECT COUNT(*) FROM pending_payloads) - ? + 1)
		)`, s.maxRows-1); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pending_payloads (collected_at, payload_type, payload_blob, checksum)
		VALUES (?, ?, ?, ?)`,
		time.Now().Unix(), payloadType, blob, checksum,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// Peek returns up to batchSize rows in FIFO order without removing them.
func (s *Store) Peek(ctx context.Context) ([]PendingRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, collected_at, payload_type, payload_blob
		FROM pending_payloads
		ORDER BY collected_at ASC
		LIMIT ?`, s.batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PendingRow
	for rows.Next() {
		var r PendingRow
		var ts int64
		var blob []byte
		if err := rows.Scan(&r.ID, &ts, &r.PayloadType, &blob); err != nil {
			continue
		}
		r.CollectedAt = time.Unix(ts, 0)
		plain, err := s.decrypt(blob)
		if err != nil {
			continue
		}
		r.Payload = json.RawMessage(plain)
		result = append(result, r)
	}
	return result, rows.Err()
}

// Ack removes a successfully delivered row.
func (s *Store) Ack(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pending_payloads WHERE id = ?`, id)
	return err
}

// Nack increments the attempt counter and, when maxTries is reached, moves
// the row to the dead_letter table.
func (s *Store) Nack(ctx context.Context, id int64, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE pending_payloads
		SET attempts = attempts + 1, last_attempt = ?
		WHERE id = ?`, time.Now().Unix(), id,
	); err != nil {
		return err
	}

	// Move to dead_letter when exhausted.
	var attempts int
	var blob []byte
	row := tx.QueryRowContext(ctx, `SELECT attempts, payload_blob FROM pending_payloads WHERE id = ?`, id)
	if err := row.Scan(&attempts, &blob); err == nil && attempts >= s.maxTries {
		tx.ExecContext(ctx, `
			INSERT INTO dead_letter (original_id, failed_at, reason, payload_blob)
			VALUES (?, ?, ?, ?)`, id, time.Now().Unix(), reason, blob)
		tx.ExecContext(ctx, `DELETE FROM pending_payloads WHERE id = ?`, id)
	}

	return tx.Commit()
}

// PendingCount returns the number of rows waiting for delivery.
func (s *Store) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pending_payloads`).Scan(&n)
	return n, err
}

// DeriveKey produces a 32-byte AES key from an arbitrary secret using SHA-256.
// For production use HKDF; this simplified derivation is fine for the cache
// because the secret is the agent's enrollment credential (already high-entropy).
func DeriveKey(secret string) []byte {
	h := sha256.Sum256([]byte("itam-cache-v1:" + secret))
	return h[:]
}

// encrypt seals plain with AES-256-GCM. The nonce is prepended to the ciphertext.
func (s *Store) encrypt(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.gcmKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

// decrypt reverses encrypt. The first NonceSize bytes are the nonce.
func (s *Store) decrypt(blob []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.gcmKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, fmt.Errorf("blob too short")
	}
	return gcm.Open(nil, blob[:ns], blob[ns:], nil)
}

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
