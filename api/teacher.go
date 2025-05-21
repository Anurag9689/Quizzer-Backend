package api

import (
	"log"
	"time"
	"strconv"
	"net/http"
	"encoding/json"

	"OnlineQuizSystem/db"
	"OnlineQuizSystem/models"
	"OnlineQuizSystem/socManager"
	"OnlineQuizSystem/utils"
)



func generateChannelCode()(string){
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// func generateQRCodeURL(channelCode string)(string){
// 	return "12423"
// }


func CreateQuizEvent(w http.ResponseWriter, r *http.Request) {
	log.Println("\nCreateQuizEventHandler handling request")
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if user.UserType != "admin" && user.UserType != "teacher" {
		http.Error(w, "Unauthorized: Only admins & teachers can create QuizEvent", http.StatusUnauthorized)
		return
	}

	var reqBody struct {
		QuizEventName string         `json:"quiz_event_name"`
		QuizJson      map[string]any `json:"quiz_json"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	channelCode := generateChannelCode()

	newQuizEvent := models.QuizEvent{
		QuizEventName: reqBody.QuizEventName,
		ChannelCode: &channelCode,
		UserID:        user.ID,
	}
	
	if _, err := newQuizEvent.SetQuizJsonFileMap(reqBody.QuizJson); err != nil {
		http.Error(w, "Internal json file writing error "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := db.DB.Create(&newQuizEvent).Error; err != nil {
		http.Error(w, "Failed to create QuizEvent: "+err.Error(), http.StatusBadRequest)
		return
	}

	manager := socManager.GetManager()
	manager.CreateRoom(newQuizEvent.ID, channelCode, user.ID)

	response := map[string]any{
		"channel_code": channelCode,
		"status":      "created",
		"quiz_event":   newQuizEvent,
		"websocket_url": "ws://"+utils.GetServerBaseUrl()+"/ws?channel_code=" + channelCode + "&user_id=" + strconv.Itoa(int(user.ID)),
		"message" : "Please join the room and start quiz event whenever you like.",
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Printf("Created quiz event %d with channel code %s", newQuizEvent.ID, channelCode)
}