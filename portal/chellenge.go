package portal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"srun-auth/config"
	"srun-auth/util"
	"strings"
)

func GetChallenge(cfg *config.Config) error {
	timestamp_str := util.GetCurrentTimeMillis()
	callback_name := "jQuery_" + timestamp_str

	var username string
	if cfg.Carrier != "" {
		username = cfg.Username + "@" + cfg.Carrier
	} else {
		username = cfg.Username
	}
	params := &url.Values{}
	params.Add("callback", callback_name)
	params.Add("username", username)
	params.Add("ip", cfg.OnlineIP)
	params.Add("_", timestamp_str)

	full_url := cfg.ChallengeURL + "?" + params.Encode()

	resp, err := http.Get(full_url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	start := strings.Index(string(body), "(")
	end := strings.LastIndex(string(body), ")")
	if start == -1 || end == -1 || start >= end {
		return fmt.Errorf("非法 JSONP 响应")
	}
	jsonStr := body[start+1 : end]
	// 解析为 map
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return err
	}

	challenge, _ := data["challenge"].(string)
	cfg.Token = challenge

	return nil
}
