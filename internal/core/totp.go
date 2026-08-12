package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func newTOTPSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}
func totpURI(secret, email string) string {
	q := url.Values{"secret": {secret}, "issuer": {"FormForge"}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}
	return "otpauth://totp/" + url.PathEscape("FormForge:"+email) + "?" + q.Encode()
}
func validateTOTP(secret, code string, now time.Time) bool {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != 6 {
		return false
	}
	for d := int64(-1); d <= 1; d++ {
		if generateTOTP(secret, uint64(now.Unix()/30+d)) == code {
			return true
		}
	}
	return false
}
func generateTOTP(secret string, counter uint64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return ""
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	h := hmac.New(sha1.New, key)
	_, _ = h.Write(msg[:])
	sum := h.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[off])&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
	return fmt.Sprintf("%06d", bin%1000000)
}
func recoveryHash(code string) string {
	n := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(code), "-", ""), " ", ""))
	h := sha256.Sum256([]byte(n))
	return hex.EncodeToString(h[:])
}
func newRecoveryCodes(n int) []string {
	out := []string{}
	for i := 0; i < n; i++ {
		raw := strings.ToUpper(strings.ReplaceAll(RandomToken(9), "_", "A"))
		raw = strings.ReplaceAll(raw, "-", "B")
		if len(raw) < 12 {
			raw += "ABCDEFGHJKLMNPQRSTUVWXYZ"
		}
		raw = raw[:12]
		out = append(out, raw[:4]+"-"+raw[4:8]+"-"+raw[8:])
	}
	return out
}
