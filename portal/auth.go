package portal

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"srun-auth/config"
	"srun-auth/sruncrypto"
	"srun-auth/util"
)

func Auth(cfg *config.Config) error {
	if cfg.Token == "" || cfg.ACID == "" || cfg.OnlineIP == "" {
		return fmt.Errorf("认证缺少 challenge、ac_id 或客户端 IP")
	}
	username := cfg.LoginUsername()
	info, err := util.Info(util.InfoData{
		Username: username,
		Password: cfg.Password,
		IP:       cfg.OnlineIP,
		ACID:     cfg.ACID,
		EncVer:   "srun_bx1",
	}, cfg.Token)
	if err != nil {
		return err
	}

	// 校验字符串使用裸 HMAC-MD5；只有请求的 password 字段添加 {MD5}。
	hmd5 := sruncrypto.HMD5(cfg.Password, cfg.Token)
	chkstr := cfg.Token + username + cfg.Token + hmd5 +
		cfg.Token + cfg.ACID + cfg.Token + cfg.OnlineIP +
		cfg.Token + "200" + cfg.Token + "1" + cfg.Token + info
	digest := sha1.Sum([]byte(chkstr))
	timestamp := util.GetCurrentTimeMillis()
	params := url.Values{
		"callback":     {"jQuery_" + timestamp},
		"action":       {"login"},
		"username":     {username},
		"password":     {"{MD5}" + hmd5},
		"ac_id":        {cfg.ACID},
		"ip":           {cfg.OnlineIP},
		"chksum":       {hex.EncodeToString(digest[:])},
		"info":         {info},
		"n":            {"200"},
		"type":         {"1"},
		"os":           {"Mac OS"},
		"name":         {"Macintosh"},
		"double_stack": {"0"},
		"_":            {timestamp},
	}
	_, err = getJSONP(cfg.AuthURL, params, cfg.UserAgent)
	return err
}
