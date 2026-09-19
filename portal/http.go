package portal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}

type portalResponse struct {
	Error     string          `json:"error"`
	ErrorMsg  string          `json:"error_msg"`
	ECode     json.RawMessage `json:"ecode"`
	Challenge string          `json:"challenge"`
	ClientIP  string          `json:"client_ip"`
}

func fetch(address, userAgent string) ([]byte, *url.URL, error) {
	req, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("无效的 HTTP 请求地址")
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		// url.Error 含完整请求 URL，其中可能包含认证信息，不能直接输出。
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("服务器返回 HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取 HTTP 响应失败: %w", err)
	}
	return body, resp.Request.URL, nil
}

func getJSONP(endpoint string, params url.Values, userAgent string) (*portalResponse, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("无效的认证接口地址")
	}
	u.RawQuery = params.Encode()
	body, _, err := fetch(u.String(), userAgent)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(body))
	text = strings.TrimSpace(strings.TrimSuffix(text, ";"))
	prefix := params.Get("callback") + "("
	if !strings.HasPrefix(text, prefix) || !strings.HasSuffix(text, ")") {
		return nil, fmt.Errorf("服务器未返回预期的 JSONP；请检查认证接口是否指向登录页面")
	}
	var response portalResponse
	if err := json.Unmarshal([]byte(text[len(prefix):len(text)-1]), &response); err != nil {
		return nil, fmt.Errorf("无法解析认证服务器的 JSON 响应")
	}
	if response.Error != "ok" {
		return nil, fmt.Errorf("服务器拒绝请求: error=%q, ecode=%s, error_msg=%q",
			response.Error, response.ECode, response.ErrorMsg)
	}
	return &response, nil
}
