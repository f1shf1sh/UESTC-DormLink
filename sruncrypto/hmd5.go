package sruncrypto

import (
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
)

func HMD5(pwd, salt string) string {
	data := []byte(pwd)
	key := []byte(salt)
	h := hmac.New(md5.New, key)
	_, _ = h.Write(data)

	binary := h.Sum(nil)

	return hex.EncodeToString(binary)
}
