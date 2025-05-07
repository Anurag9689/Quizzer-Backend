package models

import (
	"os"
	"fmt"
	"time"
	"strconv"
	"encoding/json"
	"gorm.io/datatypes"
	_ "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email    string `gorm:"unique;not null;size:256" json:"email"`
	Password string `gorm:"not null;size:768" json:"-"`
	UserType     string        `gorm:"not null;size:16" json:"user_type"`
	UserDetails  UserDetails   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"user_details"`
	QuizEvents   []QuizEvent   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"quiz_events"`
	EventResults []EventResult `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"event_results"`
}

type UserDetails struct {
	gorm.Model
	FullName      string  `gorm:"not null;size:256" json:"full_name"`
	Profession    *string `gorm:"size:256" json:"profession"`
	BriefIntro    *string `gorm:"type:TEXT" json:"brief_intro"`
	ProfileImage  *string `json:"profile_image"`
	Department    *string `gorm:"size:256" json:"department"`
	ExtraInfoJson *datatypes.JSON `json:"extra_json_info"`
	UserID        uint  `gorm:"uniqueIndex" json:"user_id"`
	User          *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type QuizEvent struct {
	gorm.Model
	QuizEventName string       `gorm:"not null;size:2048" json:"quiz_event_name"`
	QuizJsonFile  string       `gorm:"not null;size:2048" json:"quiz_json_file"`
	ChannelCode   *string 	   `gorm:"size:64" json:"channel_code"`
	UserID        uint         `gorm:"index" json:"user_id"`
	EventResult   *EventResult `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"event_result"`
}	

type EventResult struct {
	gorm.Model
	UserID        uint `gorm:"uniqueIndex" json:"user_id"`
	QuizEventID   uint `gorm:"uniqueIndex" json:"quiz_event_id"`
	QuizEvent     QuizEvent `json:"-"`
	ExpScore      int  `json:"exp_score"`
	ExtraInfoJson *datatypes.JSON `json:"extra_json_info"`
	User          *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}




func (quizEvent *QuizEvent) GetQuizJsonFileMap()(map[string]any, error){
	// Read file
	fileData, err := os.ReadFile(quizEvent.QuizJsonFile)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON into map
	var data map[string]any
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (quizEvent *QuizEvent) SetQuizJsonFileMap(dataMap map[string]any)(*string, error){
	jsonDir := os.Getenv("QUIZ_JSONS_DIR")
	timeNs := strconv.FormatInt(time.Now().UnixNano(), 10)
	quizJsonFile := fmt.Sprintf("%s/%s.json", jsonDir, timeNs)
	// Convert map to JSON
	jsonData, err := json.MarshalIndent(dataMap, "", "  ")
	if err != nil {
		return nil, err
	}

	// Write to file
	err = os.WriteFile(quizJsonFile, jsonData, 0744)
	if err != nil {
		return nil, err
	}

	quizEvent.QuizJsonFile = quizJsonFile

	return &quizJsonFile, nil
}
