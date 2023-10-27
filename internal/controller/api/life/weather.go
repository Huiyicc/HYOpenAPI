package life

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	appLife "hzer/internal/app/life"
	redisPkg "hzer/internal/redis"
	"hzer/internal/response"
	"strconv"
	"time"
)

func ginGetWeather(ctx *gin.Context) {

	//appLife.GetCurrentWeather()
}

func ginPOSTWeather(ctx *gin.Context) {
	cityID, _ := ctx.GetPostForm("cityID")

	// 获取时间
	// 每天3点,8点,13点,19点更新一次天气
	nowTime := time.Now()
	nowHour := nowTime.Hour()
	nowDay := nowTime.Day()
	if nowHour < 4 {
		// 3点前
		nowHour = 0
	} else if nowHour < 9 {
		// 8点前
		nowHour = 8
	} else if nowHour < 14 {
		// 13点前
		nowHour = 13
	} else if nowHour < 20 {
		// 19点前
		nowHour = 19
	} else {
		// 19点后
		nowHour = 24
	}
	timeKey := "weather:" + cityID + ":" + strconv.Itoa(nowDay) + ":" + strconv.Itoa(nowHour)
	rc := redisPkg.GetCoon()
	cityInfo := appLife.CityInfo{}
	// 判断是否存在
	value, _ := redis.String(rc.Do("get", timeKey))
	if value == "" {
		// redis不存在,更新
		resp, respBody, err := appLife.GetCurrentWeather(cityID)
		if err != nil {
			response.FailJson(ctx, response.FailStruct{
				Code: 10001,
				Msg:  err.Error(),
			}, false)
			return
		}
		// 设置redis,4小时过期
		_, _ = rc.Do("set", timeKey, respBody, "ex", 60*60*6)
		value = string(respBody)
		cityInfo = resp
	} else {
		err := json.Unmarshal([]byte(value), &cityInfo)
		if err != nil {
			response.FailJson(ctx, response.FailStruct{
				Code: 10002,
				Msg:  err.Error(),
			}, false)
			return
		}
	}
	response.SuccessJson(ctx, "ok", cityInfo)
}
