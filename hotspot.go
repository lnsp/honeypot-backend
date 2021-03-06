package main

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func updateHotspot(hotspot Hotspot) string {
	ssid := generateSSID(hotspot.Session)
	database.Model(&hotspot).Update("Session", ssid)
	return ssid
}

func fetchHotspot(hotspot Hotspot) (string, string) {
	var name, color string
	if hotspot.Conqueror > 0 {
		var conqueror Profile
		database.First(&conqueror, "ID = ?", hotspot.Conqueror)
		name = conqueror.Name
		color = conqueror.Color
	} else {
		name = DEFAULT_CONQUEROR
		color = DEFAULT_HOTSPOT_COLOR
	}

	return name, color
}

func captureHotspot(hotspot Hotspot, id uint) bool {
	nextCapture := time.Now().Add(-time.Second * CAPTURE_TIME)
	if nextCapture.After(hotspot.LastCapture) && hotspot.Conqueror != id {
		timeDifference := time.Now().Sub(nextCapture)
		increasePoints(hotspot.Conqueror, uint(timeDifference.Seconds())/CONQUER_POINTS_SCALAR)
		increasePoints(id, CONQUER_POINTS)

		database.Model(&hotspot).Update(Hotspot{LastCapture: time.Now(), Conqueror: id})
		return true
	}

	return false
}

func createHotspot() Hotspot {
	token := generateSSID(ULTIMATE_KEY)
	hotspot := Hotspot{
		Token:       token,
		Session:     generateSSID(token + ULTIMATE_KEY),
		LastCapture: time.Now().Add(-time.Second * CAPTURE_TIME),
		Conqueror:   0,
	}
	database.Create(&hotspot)
	return hotspot
}

func setupHotspotHandler(w http.ResponseWriter, r *http.Request) {
	secrets, ok := r.URL.Query()["secret"]
	if !ok || len(secrets) != 1 || secrets[0] != ULTIMATE_KEY {
		http.Error(w, STATUS_INVALID_TOKEN, http.StatusUnauthorized)
		return
	}

	hotspot := createHotspot()
	sendJSONResponse(struct {
		Token string `json:"token"`
		SSID  string `json:"ssid"`
	}{
		Token: hotspot.Token,
		SSID:  hotspot.Session,
	}, w)
}

func captureHotspotHandler(w http.ResponseWriter, r *http.Request) {
	token, err := validateRequest(r)
	if err != nil {
		http.Error(w, STATUS_INVALID_TOKEN, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	ssid := vars["ssid"]
	id := getID(token)

	if id == 0 {
		http.Error(w, STATUS_INVALID_USER, http.StatusUnauthorized)
		return
	}

	var hotspot Hotspot
	database.First(&hotspot, SQL_FIND_SESSION_ID, ssid)
	if hotspot.Session != ssid {
		http.Error(w, STATUS_INVALID_SSID, http.StatusInternalServerError)
		return
	}

	success := captureHotspot(hotspot, id)
	sendJSONResponse(struct {
		Success bool `json:"success"`
	}{
		Success: success,
	}, w)
}

func updateHotspotHandler(w http.ResponseWriter, r *http.Request) {
	hotspot, err := getHotspot(r)
	if err != nil {
		http.Error(w, STATUS_INVALID_TOKEN, http.StatusUnauthorized)
		return
	}

	ssid := updateHotspot(hotspot)
	sendJSONResponse(struct {
		SSID string `json:"ssid"`
	}{
		SSID: ssid,
	}, w)
}

func fetchHotspotHandler(w http.ResponseWriter, r *http.Request) {
	hotspot, err := getHotspot(r)
	if err != nil {
		http.Error(w, STATUS_INVALID_TOKEN, http.StatusUnauthorized)
		return
	}

	name, color := fetchHotspot(hotspot)
	sendJSONResponse(struct {
		Name    string `json:"name"`
		Color   string `json:"color"`
		Capture int64  `json:"capture"`
	}{
		Name:    name,
		Color:   color,
		Capture: int64(hotspot.LastCapture.Unix()),
	}, w)
}
