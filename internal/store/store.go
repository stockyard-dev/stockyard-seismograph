package store

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type ErrorEvent struct {
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	Level       string `json:"level"` // debug, info, warning, error, fatal
	Source      string `json:"source"`
	Stack       string `json:"stack"`
	Metadata    string `json:"metadata"` // JSON blob
	Status      string `json:"status"`   // open, acknowledged, resolved, ignored
	Count       int    `json:"count"`
	FirstSeen   string `json:"first_seen"`
	LastSeen    string `json:"last_seen"`
}

type ErrorOccurrence struct {
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Message     string `json:"message"`
	Stack       string `json:"stack"`
	Metadata    string `json:"metadata"`
	CreatedAt   string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(d, "seismograph.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS error_groups(
		id TEXT PRIMARY KEY, fingerprint TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL, message TEXT DEFAULT '',
		level TEXT DEFAULT 'error', source TEXT DEFAULT '',
		stack TEXT DEFAULT '', metadata TEXT DEFAULT '{}',
		status TEXT DEFAULT 'open', count INTEGER DEFAULT 1,
		first_seen TEXT DEFAULT(datetime('now')),
		last_seen TEXT DEFAULT(datetime('now')))`)

	db.Exec(`CREATE TABLE IF NOT EXISTS error_occurrences(
		id TEXT PRIMARY KEY, fingerprint TEXT NOT NULL,
		message TEXT DEFAULT '', stack TEXT DEFAULT '',
		metadata TEXT DEFAULT '{}',
		created_at TEXT DEFAULT(datetime('now')))`)

	db.Exec(`CREATE INDEX IF NOT EXISTS idx_occ_fp ON error_occurrences(fingerprint)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS extras(resource TEXT NOT NULL,record_id TEXT NOT NULL,data TEXT NOT NULL DEFAULT '{}',PRIMARY KEY(resource, record_id))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }

func fingerprint(title, source string) string {
	h := sha256.Sum256([]byte(title + "|" + source))
	return fmt.Sprintf("%x", h[:8])
}

// Ingest captures a new error, creating or updating the group
func (d *DB) Ingest(title, message, level, source, stack, metadata string) (*ErrorEvent, error) {
	fp := fingerprint(title, source)
	ts := now()

	if level == "" {
		level = "error"
	}
	if metadata == "" {
		metadata = "{}"
	}

	// Try to update existing group
	res, _ := d.db.Exec(`UPDATE error_groups SET count=count+1, last_seen=?, message=?, stack=? WHERE fingerprint=?`,
		ts, message, stack, fp)
	rows, _ := res.RowsAffected()

	if rows == 0 {
		// New group
		id := genID()
		d.db.Exec(`INSERT INTO error_groups(id,fingerprint,title,message,level,source,stack,metadata,status,count,first_seen,last_seen)VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, fp, title, message, level, source, stack, metadata, "open", 1, ts, ts)
	}

	// Record occurrence
	occID := genID()
	d.db.Exec(`INSERT INTO error_occurrences(id,fingerprint,message,stack,metadata,created_at)VALUES(?,?,?,?,?,?)`,
		occID, fp, message, stack, metadata, ts)

	return d.GetByFingerprint(fp), nil
}

func (d *DB) GetByFingerprint(fp string) *ErrorEvent {
	var e ErrorEvent
	if d.db.QueryRow(`SELECT id,fingerprint,title,message,level,source,stack,metadata,status,count,first_seen,last_seen FROM error_groups WHERE fingerprint=?`, fp).
		Scan(&e.ID, &e.Fingerprint, &e.Title, &e.Message, &e.Level, &e.Source, &e.Stack, &e.Metadata, &e.Status, &e.Count, &e.FirstSeen, &e.LastSeen) != nil {
		return nil
	}
	return &e
}

func (d *DB) Get(id string) *ErrorEvent {
	var e ErrorEvent
	if d.db.QueryRow(`SELECT id,fingerprint,title,message,level,source,stack,metadata,status,count,first_seen,last_seen FROM error_groups WHERE id=?`, id).
		Scan(&e.ID, &e.Fingerprint, &e.Title, &e.Message, &e.Level, &e.Source, &e.Stack, &e.Metadata, &e.Status, &e.Count, &e.FirstSeen, &e.LastSeen) != nil {
		return nil
	}
	return &e
}

func (d *DB) List(level, status, source string) []ErrorEvent {
	where := "1=1"
	args := []any{}
	if level != "" {
		where += " AND level=?"
		args = append(args, level)
	}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if source != "" {
		where += " AND source=?"
		args = append(args, source)
	}
	rows, _ := d.db.Query(`SELECT id,fingerprint,title,message,level,source,stack,metadata,status,count,first_seen,last_seen FROM error_groups WHERE `+where+` ORDER BY last_seen DESC`, args...)
	if rows == nil {
		return []ErrorEvent{}
	}
	defer rows.Close()
	var out []ErrorEvent
	for rows.Next() {
		var e ErrorEvent
		rows.Scan(&e.ID, &e.Fingerprint, &e.Title, &e.Message, &e.Level, &e.Source, &e.Stack, &e.Metadata, &e.Status, &e.Count, &e.FirstSeen, &e.LastSeen)
		out = append(out, e)
	}
	if out == nil {
		return []ErrorEvent{}
	}
	return out
}

func (d *DB) SetStatus(id, status string) error {
	_, err := d.db.Exec(`UPDATE error_groups SET status=? WHERE id=?`, status, id)
	return err
}

func (d *DB) Delete(id string) error {
	var fp string
	d.db.QueryRow(`SELECT fingerprint FROM error_groups WHERE id=?`, id).Scan(&fp)
	if fp != "" {
		d.db.Exec(`DELETE FROM error_occurrences WHERE fingerprint=?`, fp)
	}
	_, err := d.db.Exec(`DELETE FROM error_groups WHERE id=?`, id)
	return err
}

func (d *DB) Occurrences(fp string) []ErrorOccurrence {
	rows, _ := d.db.Query(`SELECT id,fingerprint,message,stack,metadata,created_at FROM error_occurrences WHERE fingerprint=? ORDER BY created_at DESC LIMIT 50`, fp)
	if rows == nil {
		return []ErrorOccurrence{}
	}
	defer rows.Close()
	var out []ErrorOccurrence
	for rows.Next() {
		var o ErrorOccurrence
		rows.Scan(&o.ID, &o.Fingerprint, &o.Message, &o.Stack, &o.Metadata, &o.CreatedAt)
		out = append(out, o)
	}
	if out == nil {
		return []ErrorOccurrence{}
	}
	return out
}

func (d *DB) Sources() []string {
	rows, _ := d.db.Query(`SELECT DISTINCT source FROM error_groups WHERE source!='' ORDER BY source`)
	if rows == nil {
		return []string{}
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

func (d *DB) Stats() map[string]any {
	var total, open, acked, resolved int
	d.db.QueryRow(`SELECT COUNT(*) FROM error_groups`).Scan(&total)
	d.db.QueryRow(`SELECT COUNT(*) FROM error_groups WHERE status='open'`).Scan(&open)
	d.db.QueryRow(`SELECT COUNT(*) FROM error_groups WHERE status='acknowledged'`).Scan(&acked)
	d.db.QueryRow(`SELECT COUNT(*) FROM error_groups WHERE status='resolved'`).Scan(&resolved)

	byLevel := map[string]int{}
	rows, _ := d.db.Query(`SELECT level, COUNT(*) FROM error_groups GROUP BY level`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l string
			var c int
			rows.Scan(&l, &c)
			byLevel[l] = c
		}
	}

	return map[string]any{
		"total":        total,
		"open":         open,
		"acknowledged": acked,
		"resolved":     resolved,
		"by_level":     byLevel,
	}
}

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
