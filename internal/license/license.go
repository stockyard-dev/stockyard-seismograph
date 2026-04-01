package license
import ("crypto/ed25519";"encoding/hex";"strings";"time")
const pubKeyHex = "3af8f9593b3331c27994f1eeacf111c727ff6015016b0af44ed3ca6934d40b13"
func Validate(key string) bool {
    parts := strings.SplitN(key, ".", 3)
    if len(parts) != 3 { return false }
    payload := parts[0] + "." + parts[1]
    sig, err := hex.DecodeString(parts[2])
    if err != nil { return false }
    pub, err := hex.DecodeString(pubKeyHex)
    if err != nil { return false }
    if !ed25519.Verify(pub, []byte(payload), sig) { return false }
    expStr, err := hex.DecodeString(parts[1])
    if err != nil { return false }
    exp, err := time.Parse("2006-01-02", string(expStr))
    if err != nil { return false }
    return time.Now().Before(exp)
}
func Tier(key string) string { if Validate(key) { return "pro" }; return "free" }
