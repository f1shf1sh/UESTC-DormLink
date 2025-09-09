package portal

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"srun-auth/config"
	"srun-auth/sruncrypto"
	"srun-auth/util"
)

func Auth(cfg *config.Config) error {
	data := util.InfoData{
		Username: cfg.Username,
		Password: cfg.Password,
		IP:       cfg.OnlineIP,
		ACID:     cfg.ACID,
		EncVer:   "srun_bx1",
	}

	info, err := util.Info(data, cfg.Token)
	if err != nil {
		return err
	}

	var username string
	if cfg.Carrier != "" {
		username = cfg.Username + "@" + cfg.Carrier
	} else {
		username = cfg.Username
	}
	password := "{MD5}" + sruncrypto.HMD5(cfg.Password, cfg.Token)

	chkstr := cfg.Token + cfg.Username
	chkstr += cfg.Token + password
	chkstr += cfg.Token + cfg.ACID
	chkstr += cfg.Token + cfg.OnlineIP
	chkstr += cfg.Token + "200"
	chkstr += cfg.Token + "1"
	chkstr += cfg.Token + info
	h := sha1.New()
	h.Write([]byte(chkstr))
	hash_bytes := h.Sum(nil)
	chksum := hex.EncodeToString(hash_bytes)

	timestamp_str := util.GetCurrentTimeMillis()
	callback_name := "jQuery_" + timestamp_str

	params := &url.Values{}
	params.Add("callback", callback_name)
	params.Add("action", "login")
	params.Add("username", username)
	params.Add("password", password)
	params.Add("ac_id", cfg.ACID)
	params.Add("ip", cfg.OnlineIP)
	params.Add("chksum", chksum)
	params.Add("info", info)
	params.Add("n", "200")
	params.Add("type", "1")
	params.Add("os", "Mac OS")
	params.Add("name", "Macintosh")
	params.Add("double_stack", "0")
	params.Add("_", timestamp_str)

	full_url := cfg.AuthURL + "?" + params.Encode()
	fmt.Println("url:", full_url)
	resp, err := http.Get(full_url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	return nil
}
