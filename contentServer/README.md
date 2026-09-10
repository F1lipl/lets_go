# contentServer

基于 go-zero 的 REST 服务基础工程。

## 启动

```powershell
go run . -f etc/content-api.yaml
```

服务默认监听 `8888` 端口，可用下面的地址检查运行状态：

```text
GET http://localhost:8888/api/v1/health
```

预期响应：

```json
{"status":"ok"}
```

## 开发命令

```powershell
# 重新生成 API 代码
go tool goctl api go -api content.api -dir . --style gozero

# 整理依赖
go mod tidy

# 运行测试
go test ./...

# 编译
go build -o bin/contentserver.exe .
```

接口定义位于 `content.api`，服务配置位于 `etc/content-api.yaml`。
