package util

import (
	"encoding/json"
	// "fmt"
	"srun-auth/sruncrypto"
)

// InfoData 对应 info 函数的对象参数
type InfoData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IP       string `json:"ip"`
	ACID     string `json:"acid"`
	EncVer   string `json:"enc_ver"`
}

func Info(data InfoData, token string) (string, error) {
	json_bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	json_str := string(json_bytes)
	x := sruncrypto.XEncode(json_str, token)

	return "{SRBX1}" + sruncrypto.Base64Encode(x), nil
}
