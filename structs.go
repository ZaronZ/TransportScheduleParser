package main

import "time"

type CsrfToken struct {
	Token string `json:"csrfToken"`
}

type ScheduleJson struct {
	Value string `json:"value"`
	TzOffset int64 `json:"tzOffset"`
	Text string `json:"text"`
}

type EventJson struct {
	Scheduled ScheduleJson `json:"Scheduled"`
	Estimated ScheduleJson `json:"Estimated"`
	VehicleId string `json:"vehicleId"`
}

type BriefScheduleJson struct {
	Events []EventJson `json:"Events"`
}

type ThreadJson struct {
	ThreadId string `json:"threadId"`
	NoBoarding bool `json:"noBoarding"`
	BriefSchedule BriefScheduleJson `json:"BriefSchedule"`
}

type TransportJson struct {
	Id string `json:"lineId"`
	Name string `json:"name"`
	Type string `json:"bus"`
	Threads []ThreadJson `json:"threads"`
}

type DataJson struct {
	Id string `json:"id"`
	Name string `json:"name"`
	CurrentTime int64 `json:"currentTime"`
	TzOffset int64 `json:"tzOffset"`
	Type string `json:"type"`
	Transports []TransportJson `json:"transports"`
}

type StopInfoJson struct {
	Data *DataJson `json:"data"`
}

type BusInfo struct {
	BusName string
	Scheduled [][]time.Time
	Estimated [][]time.Time
}

type StopInfo struct {
	StopName string
	BusesInfo []BusInfo
}
