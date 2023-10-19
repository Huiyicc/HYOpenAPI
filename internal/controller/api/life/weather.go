package life

import (
	"github.com/gin-gonic/gin"
	appLife "hzer/internal/app/life"
	"hzer/internal/response"
)

func ginGetWeather(ctx *gin.Context) {

	//appLife.GetWeather()
}

func ginPOSTWeather(ctx *gin.Context) {
	cityID, _ := ctx.GetPostForm("cityID")
	err := appLife.GetWeather(cityID)
	if err != nil {
		response.FailJson(ctx, response.FailStruct{
			Code: 10001,
			Msg:  err.Error(),
		}, false)
		return
	}

}
