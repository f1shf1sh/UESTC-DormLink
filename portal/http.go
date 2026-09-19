package portal

import (
	"context"
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
	PloyMsg   string          `json:"ploy_msg"`
	ECode     json.RawMessage `json:"ecode"`
	Challenge string          `json:"challenge"`
	ClientIP  string          `json:"client_ip"`
}

func fetch(ctx context.Context, address, userAgent string) ([]byte, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
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

func requestJSONP(ctx context.Context, endpoint string, params url.Values, userAgent string) (*portalResponse, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("无效的认证接口地址")
	}
	u.RawQuery = params.Encode()
	body, _, err := fetch(ctx, u.String(), userAgent)
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
	return &response, nil
}

func getJSONP(ctx context.Context, endpoint string, params url.Values, userAgent string) (*portalResponse, error) {
	response, err := requestJSONP(ctx, endpoint, params, userAgent)
	if err != nil {
		return nil, err
	}
	if response.Error != "ok" {
		return nil, response.failure()
	}
	return response, nil
}

// ResponseError 保留业务错误码，调用方无需匹配拼接后的日志文本。
type ResponseError struct {
	Code          string
	ECode         string
	Message       string
	PolicyMessage string
}

func (err *ResponseError) Error() string {
	message := fmt.Sprintf("服务器拒绝请求: error=%q, ecode=%q, error_msg=%q", err.Code, err.ECode, err.Message)
	if err.PolicyMessage != "" {
		message += fmt.Sprintf(", ploy_msg=%q", err.PolicyMessage)
	}
	return message
}

func (response *portalResponse) failure() error {
	var code string
	if json.Unmarshal(response.ECode, &code) != nil {
		code = string(response.ECode)
	}
	return &ResponseError{Code: response.Error, ECode: code, Message: response.ErrorMsg, PolicyMessage: response.PloyMsg}
}

// IsPermanent 仅识别门户明确要求人工处理的账号错误。
func IsPermanent(err error) bool {
	var response *ResponseError
	if !errors.As(err, &response) {
		return false
	}
	for _, code := range []string{response.Code, response.ECode, response.Message, response.PolicyMessage} {
		switch code {
		case "account_locked", "user_must_modify_password", "E2531", "E2553":
			return true
		}
	}
	// 门户也会把错误码作为 error_msg / ploy_msg 的前缀返回。
	for _, message := range []string{response.Message, response.PolicyMessage} {
		if strings.HasPrefix(message, "E2531:") || strings.HasPrefix(message, "E2553:") {
			return true
		}
	}
	return false
}
