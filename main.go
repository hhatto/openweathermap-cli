package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "strconv"
    "time"
)

type WeatherInfo struct {
    countryName string
    cityName string
    weathers []Weather
}

type Weather struct {
    timestamp int64
    weather string
    weatherDetail string
    temp float64
    minTemp float64
    maxTemp float64
    clouds int              // %
    rain float64            // mm/3h
    humidity int            // %
}

func k2c(kelvinTemp float64) (float64) {
    return kelvinTemp - 273.15
}

func c2k(celsiusTemp float64) (float64) {
    return celsiusTemp + 273.15
}

func c2f(celsiusTemp float64) (float64) {
    return celsiusTemp * 9 / 5 + 32
}

func f2c(fahrenheitTemp float64) (float64) {
    return (fahrenheitTemp - 32) * 5 / 9
}

func GetInfo(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("http error")
        os.Exit(1)
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("read error")
        os.Exit(1)
    }

    return body, err
}

func (cur WeatherInfo) Print(body []byte, color bool) {
    isError := false
    errMsg := ""
    var result map[string] interface{}
    _ = json.Unmarshal(body, &result)
    cur.weathers = make([]Weather, 1)

    for key, val := range result {
        switch key {
        case "cod":
            c := 200
            switch val.(type) {
            case string:
                c, _ = strconv.Atoi(val.(string))
            case float64:
                c = int(val.(float64))
            }
            if c != 200 {
                isError = true
            }
        case "message":
            switch val.(type) {
            case string:
                errMsg = val.(string)
            }
        case "name":
            cur.cityName = val.(string)
        case "sys":
            sys := val.(map[string]interface{})
            cur.countryName = sys["country"].(string)
        case "city":
            sys := val.(map[string]interface{})
            cur.cityName = sys["name"].(string)
            cur.countryName = sys["country"].(string)
        case "weather":
            weather := val.([]interface{})[0].(map[string]interface{})
            cur.weathers[0].weather = weather["main"].(string)
            cur.weathers[0].weatherDetail = weather["description"].(string)
        case "clouds":
            sys := val.(map[string]interface{})
            cur.weathers[0].clouds = int(sys["all"].(float64))
        case "rain":
            sys := val.(map[string]interface{})
            for k, v := range sys {
                per, _ := strconv.ParseFloat(k[:len(k)-1], 64)
                cur.weathers[0].rain = v.(float64) / per
                break
            }
        case "main":
            sys := val.(map[string]interface{})
            cur.weathers[0].temp = k2c(sys["temp"].(float64))
            cur.weathers[0].minTemp = k2c(sys["temp_min"].(float64))
            cur.weathers[0].maxTemp = k2c(sys["temp_max"].(float64))
            cur.weathers[0].humidity = int(sys["humidity"].(float64))
        case "dt":
            cur.weathers[0].timestamp = int64(val.(float64))
        case "list":
            weathers := val.([]interface{})
            cur.weathers = make([]Weather, len(weathers))

            for cnt := 0; cnt < len(weathers); cnt++ {
                w := weathers[cnt].(map[string]interface{})
                for k, v := range w {
                    switch k {
                    case "weather":
                        weather := v.([]interface{})[0].(map[string]interface{})
                        cur.weathers[cnt].weather = weather["main"].(string)
                        cur.weathers[cnt].weatherDetail = weather["description"].(string)
                    case "clouds":
                        switch v.(type) {
                        case float64:
                            cur.weathers[cnt].clouds = int(v.(float64))
                        default:
                            sys := v.(map[string]interface{})
                            cur.weathers[cnt].clouds = int(sys["all"].(float64))
                        }
                    case "rain":
                        switch v.(type) {
                        case float64:
                            cur.weathers[cnt].rain = v.(float64) / 24
                        default:
                            sys := v.(map[string]interface{})
                            for k, v := range sys {
                                per, _ := strconv.ParseFloat(k[:len(k)-1], 64)
                                cur.weathers[cnt].rain = v.(float64) / per
                                break
                            }
                        }
                    case "humidity":
                        cur.weathers[cnt].humidity = int(v.(float64))
                    case "temp":
                        sys := v.(map[string]interface{})
                        cur.weathers[cnt].temp = k2c(sys["day"].(float64))
                        cur.weathers[cnt].minTemp = k2c(sys["min"].(float64))
                        cur.weathers[cnt].maxTemp = k2c(sys["max"].(float64))
                    case "main":
                        sys := v.(map[string]interface{})
                        cur.weathers[cnt].temp = k2c(sys["temp"].(float64))
                        cur.weathers[cnt].minTemp = k2c(sys["temp_min"].(float64))
                        cur.weathers[cnt].maxTemp = k2c(sys["temp_max"].(float64))
                        cur.weathers[cnt].humidity = int(sys["humidity"].(float64))
                    case "dt":
                        cur.weathers[cnt].timestamp = int64(v.(float64))
                    }
                }
            }
        }

        if isError {
            fmt.Printf(errMsg)
            os.Exit(1)
        }
    }

    fmt.Printf("[%s,%s]\n", cur.cityName, cur.countryName)
    for _, val := range cur.weathers {
        fmt.Printf(" datetime: %s\n", time.Unix(val.timestamp, 0).Format("2006/01/02 15:04:05"))
        fmt.Printf("  weather: %s (%s)\n", val.weather, val.weatherDetail)
        fmt.Printf("     temp: %5.2f (min:%.2f / max:%.2f)\n", val.temp, val.minTemp, val.maxTemp)
        fmt.Printf("    cloud: %5d[%%]\n", val.clouds)
        fmt.Printf("     hmdy: %5d[%%]\n", val.humidity)
        fmt.Printf("     rain: %5.2f[mm/1h]\n", val.rain)
        fmt.Println("===")
    }
}

func main() {
    var (
        mode = flag.String("mode", "current", "forecast mode(current|per3h|nextday|week)")
        color = flag.Bool("color", true, "colorized output")
        location = flag.String("location", "iceland", "location(default: iceland)")
    )
    flag.Parse()

    url := bytes.NewBufferString("http://api.openweathermap.org/data/2.5/")

    switch {
    case *mode == "current":
        url.WriteString(fmt.Sprintf("weather?q=%s", *location))

        var cur WeatherInfo
        body, err := GetInfo(url.String())
        if err != nil {
            fmt.Println("error")
            os.Exit(1)
        }
        cur.Print(body, *color)

    case *mode == "per3h":
        // forecast of today
        url.WriteString(fmt.Sprintf("forecast?q=%s", *location))

        var forecast WeatherInfo
        body, _ := GetInfo(url.String())
        forecast.Print(body, *color)

    case *mode == "nextday":
        url.WriteString(fmt.Sprintf("forecast/daily?q=%s&cnt=2", *location))
        var forecastWeek WeatherInfo
        body, _ := GetInfo(url.String())
        forecastWeek.Print(body, *color)

    case *mode == "week":
        // forecast of week
        url.WriteString(fmt.Sprintf("forecast/daily?q=%s&cnt=7", *location))
        var forecastWeek WeatherInfo
        body, _ := GetInfo(url.String())
        forecastWeek.Print(body, *color)
    }
}
