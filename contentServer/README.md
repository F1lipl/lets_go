# contentServer

基于 go-zero 的 REST 服务基础工程。

## 启动

```powershell
$env:CONTENT_DB_DSN = '<内容服务专用账号>:<口令>@tcp(127.0.0.1:3309)/content_server?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai&timeout=5s&readTimeout=3s&writeTimeout=3s'
$env:CONTENT_ACCESS_SECRET = '<与用户服务一致的签发配置>'
go run . -f etc/content-api.yaml
```

`CONTENT_DB_DSN` 必须指向内容服务自己的 MySQL 实例和 `content_server`
数据库。不要填写用户服务的 `user_server` 连接地址；生产环境还应为内容服务使用独立数据库账号。

服务默认监听 `8889` 端口，可用下面的地址检查运行状态：

```text
GET http://localhost:8889/api/v1/health
```

预期响应：

```json
{"status":"ok"}
```

## 开发命令

```powershell
# 重新生成 API 代码
go tool goctl api go --api api/content.api --dir . --style go_zero

# 整理依赖
go mod tidy

# 运行测试
go test ./...

# 编译
go build -o bin/contentserver.exe .
```

接口定义位于 `api/content.api`，服务配置位于 `etc/content-api.yaml`。

## 业务响应约定

业务接口统一返回一个信封，具体业务字段只出现在 `data` 中：

```json
{
  "errorCode": 0,
  "message": "操作成功",
  "data": {},
  "requestId": ""
}
```

分层约定：

- `internal/logic` 只返回对应的 `*Data` 和 `error`，不组装响应码或提示文本。
- `internal/handler` 解析请求并调用 `writeBusinessResponse`。
- `internal/httpresponse` 统一生成成功或失败回包，并完成 HTTP 状态映射。
- 新增业务接口时必须分别定义 `XxxData` 和 `XxxResponse`；运行状态接口不使用业务信封。
