package life

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var (

	//go:embed city.json
	cityRaw []byte
)

type city struct {
	Name      string     `json:"name"` //国家
	Provinces []struct { // 省份
		ID    string     `json:"id"`
		Name  string     `json:"name"`
		Citys []struct { //城市
			ID      string     `json:"id"`
			Name    string     `json:"name"`
			Countys []struct { //区
				ID   string `json:"id"`
				Name string `json:"name"`
				Code string `json:"code"`
			} `json:"zone"`
		} `json:"zone"`
	} `json:"zone"`
	Datas     map[string]cityInfo
	DatasList map[string]cityInfo // 单层字典,键为城市代码/值为地区
}

type cityInfo struct {
	ID   string
	Name string
	Code string
	Son  map[string]cityInfo // 一层层嵌套的字典,键为省/城市/地区名
}

type T struct {
	Message  string `json:"message"` //返回message
	Status   int    `json:"status"`  //返回状态
	Date     string `json:"date"`    //当前天气的当天日期
	Time     string `json:"time"`    //系统更新时间
	CityInfo struct {
		City       string `json:"city"` //请求城市
		Citykey    string `json:"citykey"`
		CityID     string `json:"cityId"`
		Parent     string `json:"parent"`     //上级，一般是省份
		UpdateTime string `json:"updateTime"` //天气更新时间
	} `json:"cityInfo"`
	Data struct {
		Shidu     string       `json:"shidu"`     //湿度
		Pm25      float64      `json:"pm25"`      //pm2.5
		Pm10      float64      `json:"pm10"`      //pm10
		Quality   string       `json:"quality"`   //空气质量
		Wendu     string       `json:"wendu"`     //温度
		Ganmao    string       `json:"ganmao"`    //感冒提醒（指数）
		Forecast  []DayWeather `json:"forecast"`  //今天+未来4天
		Yesterday DayWeather   `json:"yesterday"` //昨天天气
	} `json:"data"`
}

type DayWeather struct {
	Date    string `json:"date"`    //日    去掉了原来的  日字 + 星期，如  21日星期五 变成了21，星期和年月日在下面
	High    string `json:"high"`    //当天最高温
	Low     string `json:"low"`     //当天最低温
	Ymd     string `json:"ymd"`     //年月日
	Week    string `json:"week"`    //星期
	Sunrise string `json:"sunrise"` //日出
	Sunset  string `json:"sunset"`  //日落
	Aqi     int    `json:"aqi"`     //空气指数
	Fx      string `json:"fx"`      //风向
	Fl      string `json:"fl"`      //风力
	Type    string `json:"type"`    //天气
	Notice  string `json:"notice"`  //天气描述
}

var cityData city

func initWeatherData() error {
	err := json.Unmarshal(cityRaw, &cityData)
	if err != nil {
		return err
	}
	cityData.Datas = make(map[string]cityInfo)
	cityData.DatasList = make(map[string]cityInfo)
	for _, province := range cityData.Provinces {
		if _, ifSet := cityData.Datas[province.Name]; !ifSet {
			cityData.Datas[province.Name] = cityInfo{
				ID:   province.ID,
				Name: province.Name,
				Code: "",
				Son:  make(map[string]cityInfo),
			}
		}
		for _, city := range province.Citys {
			if _, ifSet := cityData.Datas[province.Name].Son[city.Name]; !ifSet {
				cityData.Datas[province.Name].Son[city.Name] = cityInfo{
					ID:   city.ID,
					Name: city.Name,
					Code: "",
					Son:  make(map[string]cityInfo),
				}
			}
			for _, county := range city.Countys {
				if _, ifSet := cityData.Datas[province.Name].Son[city.Name].Son[county.Name]; !ifSet {
					cityData.Datas[province.Name].Son[city.Name].Son[county.Name] = cityInfo{
						ID:   county.ID,
						Name: county.Name,
						Code: county.Code,
						Son:  nil,
					}
					cityData.DatasList[county.Code] = cityData.Datas[province.Name].Son[city.Name].Son[county.Name]
				}
			}
		}
	}
	return nil
}

func GetWeather(cityID string) error {
	// https://www.sojson.com/api/weather.html
	if cityData.Name == "" {
		if err := initWeatherData(); err != nil {
			return err
		}
	}
	if _, ifSet := cityData.DatasList[cityID]; !ifSet {
		return errors.New("城市ID不存在")
	}
	resp, err := http.Get("http://t.weather.sojson.com/api/weather/city/" + cityID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	println(respData)
	return nil
}
