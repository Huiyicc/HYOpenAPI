package life

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"hzer/configs"
	"hzer/pkg/LoadBalance"
	"hzer/pkg/tokenbucket"
	"hzer/pkg/util"
	"io"
	"net/http"
	"strconv"
)

var (

	//go:embed city_list.json
	cityRaw []byte
)

// 城市数据
type cityInteface struct {
	citys     []citys
	Datas     map[string]cityCache
	DatasList map[string]cityCache // 单层字典,键为城市代码/值为地区
}
type citys struct {
	Iso3166   string   `json:"ISO_3166"`
	CountryEN string   `json:"Country_EN"`
	CountryCN string   `json:"Country_CN"`
	Regions   []Region `json:"Regions"`
}
type Region struct {
	Name   string `json:"Name"`
	NameEn string `json:"Name_EN"`
	Citys  []City `json:"Citys"`
}
type City struct {
	Name      string     `json:"Name"`
	NameEn    string     `json:"Name_EN"`
	Locations []Location `json:"Locations"`
}

type Location struct {
	LocationID string `json:"LocationID"`
	LocationEN string `json:"Location_EN"`
	Location   string `json:"Location"`
	Latitude   string `json:"Latitude"`
	Longitude  string `json:"Longitude"`
	Adcode     string `json:"Adcode"`
}

type cityCache struct {
	Name string
	Code string
	Son  map[string]cityCache // 一层层嵌套的字典,键为省/城市/地区名
}

type CityWeatherInfo struct {
	Code       string        `json:"code"`
	UpdateTime string        `json:"updateTime"`
	FxLink     string        `json:"fxLink"`
	Now        WeatherStatus `json:"now"`
}

type WeatherStatus struct {
	ObsTime   string `json:"obsTime"`   // 数据观测时间
	Temp      string `json:"temp"`      // 温度，默认单位：摄氏度
	FeelsLike string `json:"feelsLike"` // 体感温度，默认单位：摄氏度
	Icon      string `json:"icon"`      // 天气状况图标代码
	Text      string `json:"text"`      // 天气状况的文字描述，包括阴晴雨雪等天气状态的描述
	Wind360   string `json:"wind360"`   // 风向360角度
	WindDir   string `json:"windDir"`   // 风向
	WindScale string `json:"windScale"` // 风力等级
	WindSpeed string `json:"windSpeed"` // 风速，公里/小时
	Humidity  string `json:"humidity"`  // 相对湿度，百分比数值
	Precip    string `json:"precip"`    // 当前小时累计降水量，默认单位：毫米
	Pressure  string `json:"pressure"`  // 大气压强，默认单位：百帕
	Vis       string `json:"vis"`       // 能见度，默认单位：公里
	Cloud     string `json:"cloud"`     // 云量，百分比数值
	Dew       string `json:"dew"`       // 露点温度.可能为空
}

//
//const (
//	WeatherLightRain               = "小雨"
//	WeatherLightRainToModerateRain = "小到中雨"
//	WeatherModerateRain            = "中雨"
//	WeatherModerateRainToHeavyRain = "中到大雨"
//	WeatherHeavyRain               = "大雨"
//	WeatherHeavyRainToStorm        = "大到暴雨"
//	WeatherStorm                   = "暴雨"
//	WeatherStormToHeavyStorm       = "暴雨到大暴雨"
//	WeatherHeavyStorm              = "大暴雨"
//	WeatherHeavyStormToSevereStorm = "大暴雨到特大暴雨"
//	WeatherSevereStorm             = "特大暴雨"
//	WeatherFreezingRain            = "冻雨"
//	WeatherShower                  = "阵雨"
//	WeatherThundershower           = "雷阵雨"
//	WeatherSleet                   = "雨夹雪"
//	WeatherThundershowerWithHail   = "雷阵雨伴有冰雹"
//	WeatherSpit                    = "小雪"
//	WeatherSpitToModerateSnow      = "小到中雪"
//	WeatherModerateSnow            = "中雪"
//	WeatherModerateSnowToHeavySnow = "中到大雪"
//	WeatherHeavySnow               = "大雪"
//	WeatherHeavySnowToSnowstorm    = "大到暴雪"
//	WeatherSnowstorm               = "暴雪"
//	WeatherSnowShower              = "阵雪"
//	WeatherClear                   = "晴"
//	WeatherCloudy                  = "多云"
//	WeatherOvercast                = "阴"
//	WeatherStrongSandstorm         = "强沙尘暴"
//	WeatherBlowingSand             = "扬沙"
//	WeatherSandstorm               = "沙尘暴"
//	WeatherDuststorm               = "浮尘"
//	WeatherMist                    = "雾"
//	WeatherFoggy                   = "霾"
//)

var (
	cityDatas cityInteface

	rate           float64 = 10 // 每秒补充x个令牌
	capacity       float64 = 30 // 令牌桶容量为y个
	tbWeather              = tokenbucket.NewTokenBucket(rate, capacity)
	tbWeatherIndex         = tokenbucket.NewTokenBucket(rate, capacity)

	// 负载均衡器
	lbs         = LoadBalance.NewWeightRoundRobinBalance()
	accountMaps = make(map[string]string)
)

func initWeatherData() error {
	err := json.Unmarshal(cityRaw, &cityDatas.citys)
	if err != nil {
		return err
	}
	cityDatas.Datas = make(map[string]cityCache)
	cityDatas.DatasList = make(map[string]cityCache)
	for _, cityData := range cityDatas.citys {
		for _, province := range cityData.Regions {
			if _, ifSet := cityDatas.Datas[province.Name]; !ifSet {
				cityDatas.Datas[province.Name] = cityCache{
					Name: province.Name,
					Code: "",
					Son:  make(map[string]cityCache),
				}
			}
			for _, city := range province.Citys {
				if _, ifSet := cityDatas.Datas[province.Name].Son[city.Name]; !ifSet {
					cityDatas.Datas[province.Name].Son[city.Name] = cityCache{
						Name: city.Name,
						Code: "",
						Son:  make(map[string]cityCache),
					}
				}
				for _, county := range city.Locations {
					if _, ifSet := cityDatas.Datas[province.Name].Son[city.Name].Son[county.Location]; !ifSet {
						cityDatas.Datas[province.Name].Son[city.Name].Son[county.Location] = cityCache{
							Name: county.Location,
							Code: county.LocationID,
							Son:  nil,
						}
						cityDatas.DatasList[county.LocationID] = cityDatas.Datas[province.Name].Son[city.Name].Son[county.Location]
					}
				}
			}
		}
	}
	for _, v := range configs.Data.OpenAPI.QWeather.PrivateKEYS {
		if _, ifSet := accountMaps[v.Name]; ifSet {
			fmt.Println("重复的账号名:", v.Name)
			continue
		}
		weight := strconv.Itoa(util.Ifs(v.Weight == 0, 1, v.Weight))
		lbs.Add(v.Name, weight)
		accountMaps[v.Name] = v.Key
	}
	return nil
}

func checkRespCode(code string) error {
	if code == "200" {
		return nil
	} else if code == "204" {
		return errors.New("城市数据不存在")
	} else if code == "400" {
		return errors.New("请求错误")
	} else if code == "401" {
		return errors.New("认证失败,请联系管理员")
	} else if code == "402" {
		return errors.New("超过访问次数,请联系管理员")
	} else if code == "403" {
		return errors.New("无访问权限,请联系管理员")
	} else if code == "404" {
		return errors.New("数据或地区不存在")
	} else if code == "429" {
		return errors.New("超过限制访问次数,请稍后再试")
	} else if code == "500" {
		return errors.New("服务器内部错误,请联系管理员")
	} else {
		return errors.New(fmt.Sprintf("未知错误,错误码:%s", code))
	}
}

// GetCurrentWeather 获取当前天气
//
// cityID: 城市ID
// 内置限流,每秒10个令牌,令牌桶容量30个
func GetCurrentWeather(cityID string) (ret CityWeatherInfo, raw []byte, err error) {
	//{
	//	// 3次重试
	//	max := 3
	//	status := false
	//	if !tbWeather.TryConsume() {
	//		for i := 0; i < max-1; i++ {
	//			if tbWeather.TryConsume() {
	//				status = true
	//				break
	//			}
	//		}
	//		if !status {
	//			// 限流
	//			return ret, nil, errors.New("服务器繁忙,请稍后再试")
	//		}
	//	}
	//}
	if len(cityDatas.citys) == 0 {
		if err := initWeatherData(); err != nil {
			return ret, nil, err
		}
	}
	if _, ifSet := cityDatas.DatasList[cityID]; !ifSet {
		return ret, nil, errors.New("城市ID不存在")
	}
	url := fmt.Sprintf("%s/weather/now?location=%s&key=%s",
		configs.Data.OpenAPI.QWeather.Host,
		cityID,
		accountMaps[lbs.Next()],
	)
	resp, err := http.Get(url)
	if err != nil {
		return ret, nil, err
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return ret, nil, err
	}
	err = json.Unmarshal(respData, &ret)
	if err != nil {
		return ret, nil, err
	}
	err = checkRespCode(ret.Code)
	if err != nil {
		return ret, nil, err
	}
	return ret, respData, nil
}

type CityWeatherIndexInfo struct {
	Code       string        `json:"code"`
	UpdateTime string        `json:"updateTime"`
	FxLink     string        `json:"fxLink"`
	Index      WeatherIndexs `json:"index"` // 生活指数
}
type WeatherIndexs struct {
	Motion    WeatherIndexStatus `json:"motion"`    // 运动指数
	CarWash   WeatherIndexStatus `json:"carWash"`   // 洗车指数
	Dress     WeatherIndexStatus `json:"dress"`     // 穿衣指数
	Fishing   WeatherIndexStatus `json:"fishing"`   // 钓鱼指数
	UV        WeatherIndexStatus `json:"uv"`        // 紫外线指数
	Travel    WeatherIndexStatus `json:"travel"`    // 旅游指数
	Allergy   WeatherIndexStatus `json:"allergy"`   // 过敏指数
	Comfort   WeatherIndexStatus `json:"comfort"`   // 舒适度指数
	Cold      WeatherIndexStatus `json:"cold"`      // 感冒指数
	Air       WeatherIndexStatus `json:"air"`       // 空气污染扩散条件指数
	Ac        WeatherIndexStatus `json:"ac"`        // 空调开启指数
	Sunglass  WeatherIndexStatus `json:"sunglass"`  // 太阳镜指数
	Makeup    WeatherIndexStatus `json:"makeup"`    // 化妆指数
	Dry       WeatherIndexStatus `json:"dry"`       // 晾晒指数
	Traffic   WeatherIndexStatus `json:"traffic"`   // 交通指数
	Sunscreen WeatherIndexStatus `json:"sunscreen"` // 防晒指数
}

type WeatherIndexStatus struct {
	Name     string `json:"name"`
	Level    string `json:"level"`
	Category string `json:"category"`
	Text     string `json:"text"`
}

func convertWeatherIndex(data []byte) (CityWeatherIndexInfo, error) {
	type cityWeatherIndexRaw struct {
		Code       string `json:"code"`
		UpdateTime string `json:"updateTime"`
		FxLink     string `json:"fxLink"`
		Daily      []struct {
			Date     string `json:"date"`
			Type     string `json:"type"`
			Name     string `json:"name"`
			Level    string `json:"level"`
			Category string `json:"category"`
			Text     string `json:"text"`
		} `json:"daily"`
	}
	ret := CityWeatherIndexInfo{}
	raw := cityWeatherIndexRaw{}
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return ret, err
	}
	err = checkRespCode(raw.Code)
	if err != nil {
		return ret, err
	}
	ret.Code = raw.Code
	ret.UpdateTime = raw.UpdateTime
	ret.FxLink = raw.FxLink
	for _, v := range raw.Daily {
		var tmpStatus *WeatherIndexStatus
		switch v.Type {
		case "1":
			// 运动指数
			tmpStatus = &ret.Index.Motion
		case "2":
			// 洗车指数
			tmpStatus = &ret.Index.CarWash
		case "3":
			// 穿衣指数
			tmpStatus = &ret.Index.Dress
		case "4":
			// 钓鱼指数
			tmpStatus = &ret.Index.Fishing
		case "5":
			// 紫外线指数
			tmpStatus = &ret.Index.UV
		case "6":
			// 旅游指数
			tmpStatus = &ret.Index.Travel
		case "7":
			// 过敏指数
			tmpStatus = &ret.Index.Allergy
		case "8":
			// 舒适度指数
			tmpStatus = &ret.Index.Comfort
		case "9":
			// 感冒指数
			tmpStatus = &ret.Index.Cold
		case "10":
			// 空气污染扩散条件指数
			tmpStatus = &ret.Index.Air
		case "11":
			// 空调开启指数
			tmpStatus = &ret.Index.Ac
		case "12":
			// 太阳镜指数
			tmpStatus = &ret.Index.Sunglass
		case "13":
			// 化妆指数
			tmpStatus = &ret.Index.Makeup
		case "14":
			// 晾晒指数
			tmpStatus = &ret.Index.Dry
		case "15":
			// 交通指数
			tmpStatus = &ret.Index.Traffic
		case "16":
			// 防晒指数
			tmpStatus = &ret.Index.Sunscreen
		default:
			continue
		}
		tmpStatus.Name = v.Name
		tmpStatus.Level = v.Level
		tmpStatus.Category = v.Category
		tmpStatus.Text = v.Text
	}
	return ret, nil
}

// GetWeatherIndex 获取当前天气
//
// cityID: 城市ID
// 内置限流,每秒10个令牌,令牌桶容量30个
func GetWeatherIndex(cityID string) (ret CityWeatherIndexInfo, raw []byte, err error) {
	//{
	//	// 3次重试
	//	max := 3
	//	status := false
	//	if !tbWeatherIndex.TryConsume() {
	//		for i := 0; i < max-1; i++ {
	//			if tbWeatherIndex.TryConsume() {
	//				status = true
	//				break
	//			}
	//		}
	//		if !status {
	//			// 限流
	//			return ret, nil, errors.New("服务器繁忙,请稍后再试")
	//		}
	//	}
	//}
	if len(cityDatas.citys) == 0 {
		if err := initWeatherData(); err != nil {
			return ret, nil, err
		}
	}
	if _, ifSet := cityDatas.DatasList[cityID]; !ifSet {
		return ret, nil, errors.New("城市ID不存在")
	}
	url := fmt.Sprintf("%s/indices/1d?type=0&location=%s&key=%s",
		configs.Data.OpenAPI.QWeather.Host,
		cityID,
		accountMaps[lbs.Next()],
	)
	resp, err := http.Get(url)
	if err != nil {
		return ret, nil, err
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return ret, nil, err
	}
	ret, err = convertWeatherIndex(respData)
	if err != nil {
		return ret, nil, err
	}
	return ret, respData, nil
}
