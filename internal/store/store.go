package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Event struct{
	ID string `json:"id"`
	Type string `json:"type"`
	Magnitude float64 `json:"magnitude"`
	Source string `json:"source"`
	Data string `json:"data"`
	Severity string `json:"severity"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"seismograph.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS events(id TEXT PRIMARY KEY,type TEXT NOT NULL,magnitude REAL DEFAULT 0,source TEXT DEFAULT '',data TEXT DEFAULT '',severity TEXT DEFAULT 'info',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Event)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO events(id,type,magnitude,source,data,severity,created_at)VALUES(?,?,?,?,?,?,?)`,e.ID,e.Type,e.Magnitude,e.Source,e.Data,e.Severity,e.CreatedAt);return err}
func(d *DB)Get(id string)*Event{var e Event;if d.db.QueryRow(`SELECT id,type,magnitude,source,data,severity,created_at FROM events WHERE id=?`,id).Scan(&e.ID,&e.Type,&e.Magnitude,&e.Source,&e.Data,&e.Severity,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Event{rows,_:=d.db.Query(`SELECT id,type,magnitude,source,data,severity,created_at FROM events ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Event;for rows.Next(){var e Event;rows.Scan(&e.ID,&e.Type,&e.Magnitude,&e.Source,&e.Data,&e.Severity,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM events WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n);return n}
