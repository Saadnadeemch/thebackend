package router

import (
	controllers "backend/controller"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {

	r := gin.Default()

	r.POST("/api/getvideo", controllers.VideoHandler)

	return r
}
