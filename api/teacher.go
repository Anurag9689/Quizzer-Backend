package api

import (
	"log"
	"time"
	"strconv"
	"net/http"
	"encoding/json"

	// "github.com/gorilla/mux"
	// "gorm.io/datatypes"
	"OnlineQuizSystem/db"
	"OnlineQuizSystem/utils"
	"OnlineQuizSystem/models"
	"OnlineQuizSystem/sockets"
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

	// Generate channel code
	channelCode := generateChannelCode()

	// Create quiz event
	newQuizEvent := models.QuizEvent{
		QuizEventName: reqBody.QuizEventName,
		UserID:        user.ID,
	}
	
	// Store quiz JSON in file
	if _, err := newQuizEvent.SetQuizJsonFileMap(reqBody.QuizJson); err != nil {
		http.Error(w, "Internal json file writing error "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := db.DB.Create(&newQuizEvent).Error; err != nil {
		http.Error(w, "Failed to create QuizEvent: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create WebSocket room
	manager := sockets.GetManager()
	manager.CreateRoom(newQuizEvent.ID, channelCode, user.ID)

	response := map[string]any{
		"quiz_event":   newQuizEvent,
		"channel_code": channelCode,
		// "qr_code_url":  generateQRCodeURL(channelCode),
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Printf("Created quiz event %d with channel code %s", newQuizEvent.ID, channelCode)
}