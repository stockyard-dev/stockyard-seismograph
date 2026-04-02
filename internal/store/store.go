package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Event struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Source string `json:"source"`
	Severity string `json:"severity"`
	Payload string `json:"payload"`
	Tags string `json:"tags"`
	Acknowledged int `json:"acknowledged"`
	Status string `json:"status"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"seismograph.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS events(id TEXT PRIMARY KEY,name TEXT NOT NULL,source TEXT DEFAULT '',severity TEXT DEFAULT 'info',payload TEXT DEFAULT '{}',tags TEXT DEFAULT '',acknowledged INTEGER DEFAULT 0,status TEXT DEFAULT 'active',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Event)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO events(id,name,source,severity,payload,tags,acknowledged,status,created_at)VALUES(?,?,?,?,?,?,?,?,?)`,e.ID,e.Name,e.Source,e.Severity,e.Payload,e.Tags,e.Acknowledged,e.Status,e.CreatedAt);return err}
func(d *DB)Get(id string)*Event{var e Event;if d.db.QueryRow(`SELECT id,name,source,severity,payload,tags,acknowledged,status,created_at FROM events WHERE id=?`,id).Scan(&e.ID,&e.Name,&e.Source,&e.Severity,&e.Payload,&e.Tags,&e.Acknowledged,&e.Status,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Event{rows,_:=d.db.Query(`SELECT id,name,source,severity,payload,tags,acknowledged,status,created_at FROM events ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Event;for rows.Next(){var e Event;rows.Scan(&e.ID,&e.Name,&e.Source,&e.Severity,&e.Payload,&e.Tags,&e.Acknowledged,&e.Status,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Update(e *Event)error{_,err:=d.db.Exec(`UPDATE events SET name=?,source=?,severity=?,payload=?,tags=?,acknowledged=?,status=? WHERE id=?`,e.Name,e.Source,e.Severity,e.Payload,e.Tags,e.Acknowledged,e.Status,e.ID);return err}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM events WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n);return n}

func(d *DB)Search(q string, filters map[string]string)[]Event{
    where:="1=1"
    args:=[]any{}
    if q!=""{
        where+=" AND (name LIKE ?)"
        args=append(args,"%"+q+"%");
    }
    if v,ok:=filters["source"];ok&&v!=""{where+=" AND source=?";args=append(args,v)}
    if v,ok:=filters["severity"];ok&&v!=""{where+=" AND severity=?";args=append(args,v)}
    if v,ok:=filters["status"];ok&&v!=""{where+=" AND status=?";args=append(args,v)}
    rows,_:=d.db.Query(`SELECT id,name,source,severity,payload,tags,acknowledged,status,created_at FROM events WHERE `+where+` ORDER BY created_at DESC`,args...)
    if rows==nil{return nil};defer rows.Close()
    var o []Event;for rows.Next(){var e Event;rows.Scan(&e.ID,&e.Name,&e.Source,&e.Severity,&e.Payload,&e.Tags,&e.Acknowledged,&e.Status,&e.CreatedAt);o=append(o,e)};return o
}

func(d *DB)Stats()map[string]any{
    m:=map[string]any{"total":d.Count()}
    rows,_:=d.db.Query(`SELECT status,COUNT(*) FROM events GROUP BY status`)
    if rows!=nil{defer rows.Close();by:=map[string]int{};for rows.Next(){var s string;var c int;rows.Scan(&s,&c);by[s]=c};m["by_status"]=by}
    return m
}
