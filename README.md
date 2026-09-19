# srun-auto-login

电子科技大学 SRun 校园网单次认证客户端。程序完成门户发现、获取 challenge、构造认证请求和检查结果；自动重连尚未实现。

## 配置与运行

在项目目录的 `config.json` 中填写校园网账号，SSH 账号与校园网账号无关：

```json
{
  "username": "你的学号",
  "password": "你的校园网密码",
  "carrier": "dx",
  "portal_ip": "1.1.1.1"
}
```

`username` 填不带运营商后缀的用户名；`carrier` 填实际页面使用的后缀，不带 `@`。例如 `dx` 会生成 `学号@dx`，无后缀时填空字符串。不同网关的后缀可能不同，应以宿舍实际请求为准。

`portal_ip` 支持 IP 或完整 HTTP(S) URL。现有配置的 `1.1.1.1` 用于触发未认证网络的门户跳转；如果它没有跳到 SRun 页面，可填写浏览器实际登录页的完整地址，保留 `ac_id` 等参数。程序从实际页面提取参数，不猜测 `ac_id`。代码缺省入口为 `10.253.0.235`，但配置文件优先。

有 Go 1.25 或更高版本时：

```sh
go test ./...
go vet ./...
go build -o bin/srun-auto-login .
./bin/srun-auto-login
```

本次已为远端 Linux x86_64 构建 `bin/srun-auto-login`，可直接在项目根目录运行。程序从当前工作目录读取 `config.json`。出错时打印阶段及原因，退出码为 1；服务器返回 `error: "ok"` 时退出码为 0。

## 认证逻辑

### 1. 发现门户

访问配置的入口，跟随 HTTP 跳转，兼容一次页面内 `meta refresh`。最终登录页通常类似 `/srun_portal_pc?ac_id=3`。

登录页提供网关、`ac_id` 和客户端 IP，不是提交密码的认证 API。程序优先读取 URL 参数，必要时读取页面 `CONFIG.acid`、`CONFIG.ip` 或 `user_ip` 输入框。遇到 ePortal 页面会报协议不匹配；本项目没有实现它的登录协议。

### 2. 获取 challenge

用完整用户名向同一网关的 `GET /cgi-bin/get_challenge` 发请求。要求响应合法、`error == "ok"` 且 `challenge` 非空。

challenge 是此次计算使用的 token，不是校园网密码。服务器返回 `client_ip` 时，采用这个地址用于后续所有字段；未返回时沿用门户 IP。两处都没有 IP 则停止。

### 3. 构造认证字段

以下记完整用户名为 `U`、密码为 `P`、challenge 为 `T`、接入控制器标识为 `A`、客户端地址为 `I`，`+` 表示字符串直接拼接：

```text
H = HMAC-MD5(key=T, message=P) 的小写十六进制结果
password = "{MD5}" + H

data = {username: U, password: P, ip: I, acid: A, enc_ver: "srun_bx1"}
info = "{SRBX1}" + 自定义Base64(XEncode(JSON(data), T))

chksum = SHA1(T+U + T+H + T+A + T+I + T+"200" + T+"1" + T+info)
```

`chksum` 也转换成小写十六进制。校验字符串中的 `H` **没有 `{MD5}` 前缀**，而 `info` **包含 `{SRBX1}` 前缀**。`U` 在 challenge、info、chksum 和最终请求中必须完全一致。

这里保留项目原有、协议要求的 HMAC-MD5、SHA-1、XEncode 和自定义 Base64，没有添加额外哈希。`info` 是服务器规定的编码字段，不应作为日志输出。

### 4. 提交认证并检查响应

调用 `GET /cgi-bin/srun_portal`，提交上述字段以及 `action=login`、`n=200`、`type=1` 等原有参数。

JSONP 形如 `jQuery_时间戳({...})`：程序核对 callback，解析括号内 JSON，再检查 `error`。HTTP 200 只表示请求完成，不能代表账号登录成功。HTML、非法 JSONP、缺少成功状态或服务端拒绝都会返回错误。

程序成功提示为“认证服务器返回 ok”。服务端可能报告 IP 已在线，所以这个提示不等于已确认新账号上线，也不代替外网连通性验证。不需要另外打开成功页面。

## 文件职责

- `config/config.go`：读取配置、检查必填项、统一完整用户名。
- `portal/redirect.go`：发现实际网关及页面参数。
- `portal/chellenge.go`：获取 challenge 和服务端识别的 IP；保留原文件名。
- `portal/auth.go`：构造协议字段并提交认证。
- `portal/http.go`：共用 HTTP 超时、JSONP 解析和服务端错误处理。
- `sruncrypto/`、`util/info.go`：沿用原有协议计算和编码。
- `main.go`：组织流程并返回明确退出状态。

## 验证范围与宿舍复验

离线测试使用虚拟账号、本地 HTTP 模拟服务器和独立生成的固定协议样例，覆盖完整请求链、参数一致性、不同跳转方式、服务端拒绝及缺少必要字段。它能检查客户端实现，但不能证明当前宿舍的网关、运营商后缀和账户权限有效。

回宿舍后，在 `config.json` 填好校园网凭据，从项目目录运行一次 `./bin/srun-auto-login`，再验证外网连通性。若报错，保留阶段、错误码及错误信息即可；不要公开密码、token、info 或完整认证 URL。未完成宿舍真实认证验证，也未设置 systemd、cron 或循环重试。

协议对照参考：

- [同网关门户的 Portal.js 存档](https://github.com/wuyaxv/uestc-srun-login-script/blob/main/10.253.0.235/static/themes/pro/js/Portal.js)
- [Go 实现的请求结构](https://github.com/fumiama/go-nd-portal/blob/main/portal/server.go)
- [门户页面跳转与参数读取参考](https://github.com/yuzhoujun/AutoLoginUESTC/blob/python/BitSrunLogin/LoginManager.py)
