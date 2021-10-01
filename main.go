package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/agent"
	bow "github.com/headzoo/surf/browser"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// csrf token
var token = ""

// инстанс "эмулятора браузера"
var browser *bow.Browser

// генерация подписи запроса
func hash(s string) string {
	var n uint32 = 5381
	for i := 0; i < len(s); i++ {
		n = 33 * n ^ uint32(s[i])
	}
	return strconv.FormatUint(uint64(n), 10)
}


// вывод времени
func formatTime(t time.Time) string {
	location, _ := time.LoadLocation("Europe/Moscow")
	t = t.In(location)
	return t.Format(time.RFC822)
}

// запрос на сервер яндекса
func request(stopName string) (string, error) {
	// формируем данные для запроса
	params := url.Values{}
	params.Add("ajax", "1")
	params.Add("csrfToken", token)
	params.Add("id", stopName)
	params.Add("lang", "ru")
	params.Add("locale", "ru_RU")
	params.Add("mode", "prognosis")
	params.Add("s", hash(params.Encode()))

	// делаем запрос
	err := browser.Open("https://yandex.ru/maps/api/masstransit/getStopInfo?" + params.Encode())
	if err != nil{
		return "bad request", err
	}

	// копируем данные в буфер
	buf := new(bytes.Buffer)
	_, err = browser.Download(buf)
	if err != nil {
		return "bad buf", err
	}

	return fmt.Sprint(buf), nil
}

func getStopInfoJson(stopId string) (*StopInfoJson, error) {
	var stopInfoJson StopInfoJson

	// делаем запрос на сервер яндекса
	data, err := request("stop__" + stopId)
	if err != nil {
		return nil, err
	}

	// парсим информацию об остановке, без токена парсинг не отработает
	if err = json.Unmarshal([]byte(data), &stopInfoJson); err != nil || stopInfoJson.Data == nil {
		// пробуем спарсить csrfToken
		var сsrfToken CsrfToken
		if err = json.Unmarshal([]byte(data), &сsrfToken); err != nil {
			return nil, err
		}

		if сsrfToken.Token == "" {
			return nil, errors.New("bad token: " + data)
		}

		// задаём новый токен
		token = сsrfToken.Token

		// повторяем запрос
		data, err = request("stop__" + stopId)
		if err != nil {
			return nil, err
		}

		// пробуем спарсить ещё раз
		if err = json.Unmarshal([]byte(data), &stopInfoJson); err != nil {
			return nil, err
		}

		if stopInfoJson.Data == nil {
			return nil, errors.New("bad stop info: " + data)
		}
	}

	return &stopInfoJson, nil
}

func format(stopInfoJson *StopInfoJson) string {
	// преобразование данных о рейсах
	stopInfo := StopInfo{}
	stopInfo.StopName = stopInfoJson.Data.Name
	stopInfo.BusesInfo = []BusInfo{}
	if stopInfoJson.Data.Transports != nil && len(stopInfoJson.Data.Transports) > 0 {
		for tr := 0; tr < len(stopInfoJson.Data.Transports); tr++ {
			transportJson := (stopInfoJson.Data.Transports)[tr]
			busInfo := BusInfo{}
			busInfo.BusName = transportJson.Name
			busInfo.Estimated = [][]time.Time{}
			busInfo.Scheduled = [][]time.Time{}
			if transportJson.Threads != nil && len(transportJson.Threads) > 0 {
				for th := 0; th < len(transportJson.Threads); th++ {
					threadJson := (transportJson.Threads)[th]
					if threadJson.NoBoarding == true || len(threadJson.BriefSchedule.Events) == 0 {
						continue
					}
					var estimated []time.Time
					var scheduled []time.Time
					for e := 0; e < len(threadJson.BriefSchedule.Events); e++ {
						eventJson := (threadJson.BriefSchedule.Events)[e]
						if eventJson.Estimated.Value != "" {
							if unix, err := strconv.ParseInt(eventJson.Estimated.Value, 10, 64); err == nil {
								estimated = append(estimated, time.Unix(unix, 0))
							}
						}
						if eventJson.Scheduled.Value != "" {
							if unix, err := strconv.ParseInt(eventJson.Scheduled.Value, 10, 64); err == nil {
								scheduled = append(scheduled, time.Unix(unix, 0))
							}
						}
					}
					if len(estimated) > 0 {
						busInfo.Estimated = append(busInfo.Estimated, estimated)
					}
					if len(scheduled) > 0 
						busInfo.Scheduled = append(busInfo.Scheduled, scheduled)
					}
				}
			}
			stopInfo.BusesInfo = append(stopInfo.BusesInfo, busInfo)
		}
	}

	var builder strings.Builder
	builder.WriteString("Остановка: " + stopInfo.StopName+ "\n")
	for i := 0; i < len(stopInfo.BusesInfo); i++ {
		builder.WriteString("  Маршрут: " + stopInfo.BusesInfo[i].BusName+ "\n")
		if len(stopInfo.BusesInfo[i].Estimated) == 0 && len(stopInfo.BusesInfo[i].Scheduled) == 0 {
			builder.WriteString("    Ближайших рейсов нет"+ "\n")
		}

		for j := 0; j < len(stopInfo.BusesInfo[i].Estimated); j++ {
			if len(stopInfo.BusesInfo[i].Estimated) > 1 && len(stopInfo.BusesInfo[i].Estimated[j]) > 0 {
				builder.WriteString("    Расписание №" + strconv.Itoa(j) + ":\n") // я не уверен, что это можно так называть
			}

			for k := 0; k < len(stopInfo.BusesInfo[i].Estimated[j]); k++ {
				builder.WriteString("    Предполагаемое время прибытия: " + formatTime(stopInfo.BusesInfo[i].Estimated[j][k]) + "\n")
			}
		}

		for j := 0; j < len(stopInfo.BusesInfo[i].Scheduled); j++ {
			if len(stopInfo.BusesInfo[i].Scheduled) > 1 && len(stopInfo.BusesInfo[i].Scheduled[j]) > 0 {
				builder.WriteString("    Расписание №" + strconv.Itoa(j) + ":\n") // я не уверен, что это можно так называть
			}

			for k := 0; k < len(stopInfo.BusesInfo[i].Scheduled[j]); k++ {
				builder.WriteString("    Время прибытия по расписнию: " + formatTime(stopInfo.BusesInfo[i].Scheduled[j][k])+ "\n")
			}
		}
	}

	return builder.String()
}

func main() {
	// Инициализация surf
	surf.DefaultUserAgent = agent.Firefox()
	surf.DefaultSendReferer = true
	surf.DefaultMetaRefreshHandling = true
	surf.DefaultFollowRedirects = true
	browser = surf.NewBrowser()

	// получение информации об остановке
	// https://yandex.ru/maps/213/moscow/stops/stop__9645370
	stopInfoJson, err := getStopInfoJson("9645370")
	if err != nil {
		log.Fatalf(err.Error())
	}

	log.Print(format(stopInfoJson))
}
