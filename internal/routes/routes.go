package routes

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine *gin.Engine
}

func NewRouter(mcpHandler http.Handler) Router {
	r := gin.Default()
	r.Any("/mcp", gin.WrapH(mcpHandler))
	return Router{engine: r}
}

func (r Router) Serve(addr string) {
	if err := r.engine.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
