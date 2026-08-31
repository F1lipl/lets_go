# userServer 日志与链路环境

在项目根目录启动 Jaeger：

```powershell
docker compose -f .\deploy\observability\compose.yaml up -d
```

编译并启动服务：

```powershell
go build -o .\userserver.exe .\userserver.go
.\userserver.exe -f .\etc\userserver-api.yaml
```

打开 <http://localhost:16686>，选择 `user-server-api`，发送一次接口请求后点击
**Find Traces**，即可查看调用记录。

JSON 日志写入项目根目录的 `logs` 文件夹。每条请求相关日志会带有
`trace_id` 和 `span_id`，可以通过 `trace_id` 在日志和 Jaeger 之间相互定位。

检查运行状态：

```powershell
docker compose -f .\deploy\observability\compose.yaml ps
docker compose -f .\deploy\observability\compose.yaml logs jaeger
```

停止 Jaeger：

```powershell
docker compose -f .\deploy\observability\compose.yaml down
```

当前配置用于本地开发：日志全量采集，Jaeger 使用内存保存数据，容器重建后旧记录会消失。
