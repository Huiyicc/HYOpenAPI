package life

import (
	"encoding/json"
	"fmt"
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

type weatherResp struct {
	UpdateTime    string                `json:"updateTime"`
	WeatherStatus appLife.WeatherStatus `json:"weather_status"`
	WeatherIndexs appLife.WeatherIndexs `json:"weather_indexs"`
}

func (c *weatherResp) Parse(status *appLife.CityWeatherInfo, indexs *appLife.CityWeatherIndexInfo) {
	c.WeatherStatus = status.Now
	c.WeatherIndexs = indexs.Index
	c.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
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
	cityInfo := weatherResp{}
	// 判断是否存在
	value, _ := redis.String(rc.Do("get", timeKey))
	if value == "" {
		// redis不存在,更新
		resp, respBody, err := appLife.GetCurrentWeather(cityID)
		if err != nil {
			response.FailJson(ctx, response.FailStruct{
				Code: 10001,
				Msg:  "获取天气失败",
			}, false)
			fmt.Println(err.Error())
			return
		}
		respIndex, respBody, err := appLife.GetWeatherIndex(cityID)
		if err != nil {
			response.FailJson(ctx, response.FailStruct{
				Code: 10001,
				Msg:  "获取空气指数失败",
			}, false)
			fmt.Println(err.Error())
			return
		}
		cityInfo.Parse(&resp, &respIndex)
		cacheData, _ := json.Marshal(cityInfo)
		// 设置redis,4小时过期
		_, _ = rc.Do("set", timeKey, cacheData, "ex", 60*60*6)
		value = string(respBody)
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
