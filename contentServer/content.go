// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"contentserver/internal/config"
	"contentserver/internal/handler"
	"contentserver/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/content-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	if strings.TrimSpace(c.DataSource) == "" {
		log.Fatal("CONTENT_DB_DSN must be configured")
	}
	if strings.TrimSpace(c.Auth.AccessSecret) == "" {
		log.Fatal("CONTENT_ACCESS_SECRET must be configured")
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
