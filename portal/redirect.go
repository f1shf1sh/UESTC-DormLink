package portal

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"srun-auth/config"
)

var (
	metaTag     = regexp.MustCompile("(?is)<meta\\b[^>]*>")
	inputTag    = regexp.MustCompile("(?is)<input\\b[^>]*>")
	attribute   = regexp.MustCompile("(?i)([a-z_][a-z0-9_-]*)\\s*=\\s*(?:\"([^\"]*)\"|'([^']*)'|([^\\s>]+))")
	refreshURL  = regexp.MustCompile("(?i)(?:^|;)\\s*url\\s*=\\s*(.+)$")
	configBlock = regexp.MustCompile("(?s)\\b(?:var|let|const)\\s+CONFIG\\s*=\\s*\\{(.*?)\\}")
	configACID  = regexp.MustCompile("(?:^|,)\\s*[\"']?acid[\"']?\\s*:\\s*[\"']?([0-9]+)[\"']?\\s*(?:,|$)")
	configIP    = regexp.MustCompile("(?:^|,)\\s*[\"']?ip[\"']?\\s*:\\s*[\"']([^\"']*)[\"']")
)

// Redirect 发现门户和网络参数；登录页面只提供上下文，不承担认证。
func Redirect(cfg *config.Config) error {
	address := strings.TrimSpace(cfg.PortalIP)
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	u, err := url.Parse(address)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("portal_ip 必须是门户 IP 或 HTTP(S) 地址")
	}
	body, finalURL, err := fetch(u.String(), cfg.UserAgent)
	if err != nil {
		return err
	}
	// 部分宿舍门户在 HTTP 200 页面里用一次 meta refresh 跳转。
	for _, tag := range metaTag.FindAllString(string(body), -1) {
		attrs := attributes(tag)
		if !strings.EqualFold(attrs["http-equiv"], "refresh") {
			continue
		}
		match := refreshURL.FindStringSubmatch(attrs["content"])
		if len(match) == 0 {
			continue
		}
		next, err := finalURL.Parse(strings.Trim(strings.TrimSpace(match[1]), "'\""))
		if err != nil {
			return fmt.Errorf("门户 meta refresh 地址无效")
		}
		body, finalURL, err = fetch(next.String(), cfg.UserAgent)
		if err != nil {
			return err
		}
		break
	}
	if strings.Contains(strings.ToLower(finalURL.Path), "/eportal") {
		return fmt.Errorf("当前入口指向 ePortal，不是本项目支持的 SRun 认证门户")
	}
	query := finalURL.Query()
	acid, ip := query.Get("ac_id"), query.Get("wlanuserip")
	if match := configBlock.FindStringSubmatch(string(body)); len(match) > 0 {
		if acid == "" {
			acid = firstGroup(configACID, match[1])
		}
		if ip == "" {
			ip = firstGroup(configIP, match[1])
		}
	}
	if ip == "" {
		for _, tag := range inputTag.FindAllString(string(body), -1) {
			attrs := attributes(tag)
			if attrs["id"] == "user_ip" || attrs["name"] == "user_ip" {
				ip = attrs["value"]
				break
			}
		}
	}
	if acid == "" {
		return fmt.Errorf("门户未提供 ac_id；请在宿舍网络检查 portal_ip 和实际登录页面")
	}
	origin := finalURL.Scheme + "://" + finalURL.Host
	cfg.ChallengeURL = origin + "/cgi-bin/get_challenge"
	cfg.AuthURL = origin + "/cgi-bin/srun_portal"
	cfg.ACID, cfg.OnlineIP = acid, ip
	cfg.WlanACIP = query.Get("wlanacip")
	return nil
}

func attributes(tag string) map[string]string {
	attrs := make(map[string]string)
	for _, match := range attribute.FindAllStringSubmatch(tag, -1) {
		value := match[2] + match[3] + match[4]
		attrs[strings.ToLower(match[1])] = html.UnescapeString(value)
	}
	return attrs
}

func firstGroup(pattern *regexp.Regexp, text string) string {
	if match := pattern.FindStringSubmatch(text); len(match) > 0 {
		return match[1]
	}
	return ""
}
