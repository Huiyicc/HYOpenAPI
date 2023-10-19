package life

import (
	"github.com/gin-gonic/gin"
)

func GinApi(apiRouter *gin.RouterGroup) {
	r := apiRouter.Group("/life")
	r.GET("/weather", ginGetWeather)
	r.POST("/weather", ginPOSTWeather)

}
