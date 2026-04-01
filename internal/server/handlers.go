package server
import("encoding/json";"net/http";"strconv";"github.com/stockyard-dev/stockyard-seismograph/internal/store")
func(s *Server)handleIngest(w http.ResponseWriter,r *http.Request){var req struct{Service string `json:"service"`;Level string `json:"level"`;Message string `json:"message"`;Fingerprint string `json:"fingerprint"`};json.NewDecoder(r.Body).Decode(&req);if req.Message==""{writeError(w,400,"message required");return};if req.Level==""{req.Level="error"};s.db.Ingest(req.Service,req.Level,req.Message,req.Fingerprint);writeJSON(w,202,map[string]string{"status":"ingested"})}
func(s *Server)handleList(w http.ResponseWriter,r *http.Request){resolved:=r.URL.Query().Get("resolved")=="true";list,_:=s.db.List(resolved);if list==nil{list=[]store.ErrorGroup{}};writeJSON(w,200,list)}
func(s *Server)handleResolve(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);s.db.Resolve(id);writeJSON(w,200,map[string]string{"status":"resolved"})}
func(s *Server)handleDelete(w http.ResponseWriter,r *http.Request){id,_:=strconv.ParseInt(r.PathValue("id"),10,64);s.db.Delete(id);writeJSON(w,200,map[string]string{"status":"deleted"})}
func(s *Server)handleOverview(w http.ResponseWriter,r *http.Request){m,_:=s.db.Stats();writeJSON(w,200,m)}
