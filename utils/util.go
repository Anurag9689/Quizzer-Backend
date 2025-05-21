package utils


import (
	"os"
	"log"
	"fmt"
	"sync"
	"errors"
	"strings"
	"net/http"
	"net/smtp"
	"encoding/json"

	"OnlineQuizSystem/db"
	"OnlineQuizSystem/models"
	"OnlineQuizSystem/socManager"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/datatypes"
)



func GetServerBaseUrl()(string){
	return fmt.Sprintf("%s:%s", os.Getenv("DOMAIN"), os.Getenv("PORT"))
}


func GenerateOTPEmail(otp string) string {
	message_html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Your OTP for Verification</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			background-color: #f4f4f4;
			margin: 0;
			padding: 0;
		}
		.container {
			max-width: 600px;
			margin: 20px auto;
			padding: 20px;
			background-color: #ffffff;
			border-radius: 10px;
			box-shadow: 0px 0px 10px rgba(0, 0, 0, 0.1);
		}
		h1 {
			color: #333333;
			text-align: center;
		}
		p {
			color: #666666;
			line-height: 1.6;
		}
		.otp {
			font-size: 24px;
			font-weight: bold;
			text-align: center;
			color: #007bff;
			margin: 20px 0;
			padding: 10px;
			background-color: #f0f8ff;
			border-radius: 5px;
		}
		.note {
			font-size: 14px;
			color: #999999;
			text-align: center;
			margin-top: 20px;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>Your One-Time Password (OTP)</h1>
		<p>Hello,</p>
		<p>You have requested a one-time password for identity verification. Please use the following OTP to complete your action:</p>
		<div class="otp">%s</div>
		<p>This OTP is valid for a limited time. Please do not share this OTP with anyone.</p>
		<p>If you didn't request this OTP, please ignore this email or contact our support team if you have any concerns.</p>
		<p class="note">This is an automated message. Please do not reply to this email.</p>
	</div>
</body>
</html>`, otp)
	return message_html
}







func SendOtpEmail(toEmail string, otp string, subject string) error {
	from := os.Getenv("EMAIL_FROM")
	password := os.Getenv("EMAIL_PASSWORD")
	host := os.Getenv("EMAIL_HOST")
	port := os.Getenv("EMAIL_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	htmlBody := GenerateOTPEmail(otp)

	message := []byte("MIME-version: 1.0;\r\n" +
	"Content-Type: text/html; charset=\"UTF-8\";\r\n" +
	"Subject: " + subject + "\r\n" +
	"To: " + toEmail + "\r\n\r\n" +
	htmlBody + "\r\n")

	address := host + ":" + port
	// smtp.SendMail()
	err := smtp.SendMail(address, auth, from, []string {toEmail}, message)
	return err
}








func AuthorizeUser(r *http.Request) (*models.User, int, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, http.StatusUnauthorized, errors.New("missing or invalid Authorization header")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	secret := os.Getenv("SECRET_KEY")

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil || !token.Valid {
		return nil, http.StatusUnauthorized, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, http.StatusUnauthorized, errors.New("invalid token claims")
	}

	userIDFloat, ok := claims["id"].(float64)
	if !ok {
		return nil, http.StatusUnauthorized, errors.New("user ID not found in token")
	}
	userID := uint(userIDFloat)

	var user models.User
	if err := db.DB.Preload("UserDetails").Preload("QuizEvents").Preload("EventResults").First(&user, userID).Error; err != nil {
		return nil, http.StatusUnauthorized, errors.New("user not found")
	}

	return &user, http.StatusOK, nil
}












func ReadFileToMap(filename string) (map[string]any, error) {
	// Read file
	fileData, err := os.ReadFile(filename)
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




func WriteMapToFile(filename string, data map[string]any) error {
	// Convert map to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	err = os.WriteFile(filename, jsonData, 0744)
	if err != nil {
		return err
	}

	return nil
}





/*######################################################## SOCKET TYPE UTILS ################################################# */



type QuizAnswer struct {
	QuestionID int    `json:"question_id"`
	Answer     any    `json:"answer"`
	Timestamp  int64  `json:"timestamp"`
}

type QuizSession struct {
	sync.Mutex
	Answers map[uint]map[int]QuizAnswer // userID -> questionID -> answer
}


func PrepareEndQuiz(quizEvent models.QuizEvent, user *models.User, activeSessions *map[uint]*QuizSession) (string) {
	log.Println("Starting PrepareEndQuiz function .....")
	// Finalize quiz (process answers)
	FinalizeQuiz(uint(quizEvent.ID), activeSessions)
	log.Println("Finalized results .....")

	// Update quiz status
	quizJson, err := quizEvent.GetQuizJsonFileMap()
	if err != nil {
		return "Internal Error: Not able to read this QuizEvent's Json file: "+err.Error()
	}

	quizJson["status"] = "completed"
	quizEvent.SetQuizJsonFileMap(quizJson)
	db.DB.Save(&quizEvent)
	log.Println("Ending PrepareEndQuiz function .....")
	return "Prepared"
}




func FinalizeQuiz(quizEventID uint, activeSessions *map[uint]*QuizSession) {
	log.Println("Starting FinalizeQuiz function .....")

	log.Println("getting quizEvent answers from activeSessions .....")
	session, exists := (*activeSessions)[quizEventID]
	if !exists {
		return
	}

	
	
	log.Println("getting a quizEvent instance from database .....")
	var quizEvent models.QuizEvent
	if err := db.DB.First(&quizEvent, quizEventID).Error; err != nil {
		log.Printf("Error loading quiz event: %v", err)
		return
	}

	log.Println("Getting quizEvent json file map .....")
	quizData, err := quizEvent.GetQuizJsonFileMap()
	if err != nil {
		log.Printf("Error loading quiz data: %v", err)
		return
	}

	log.Println("Getting socket manager .....")
	manager := socManager.GetManager()
	room, _ := manager.GetRoom(*quizEvent.ChannelCode)
	quizData["event_start_time"] = room.EventStartTime
	quizData["event_end_time"] = room.EventEndTime

	quizEvent.SetQuizJsonFileMap(quizData)
	log.Println("quizEvent's quizData: ", quizData)

	for userID, answers := range session.Answers {
		log.Println("\tuserID : ", userID)
		log.Println("\tanswers : ", answers)
		score, analytics := calculateResults(answers, quizData)
		analyticsByted, _ := json.Marshal(analytics)
		analyticsJson := datatypes.JSON(analyticsByted)
		log.Println("\tscore : ", score)
		result := models.EventResult{
			UserID:        userID,
			QuizEventID:   quizEventID,
			ExpScore:      score,
			ExtraInfoJson: &analyticsJson,
		}

		if err := db.DB.Create(&result).Error; err != nil {
			log.Printf("Error saving final result: %v", err)
		}
		log.Println("\tSeems like eventResult is created : ", result)
	}


	delete(*activeSessions, quizEventID)
	log.Println("Ending FinalizeQuiz function .....")
}

func calculateResults(answers map[int]QuizAnswer, quizData map[string]any) (int, map[string]any) {
	log.Println("Starting calculateResults function .....")
	type AnswerAnalytics struct {
		Answers       map[int]QuizAnswer
		CorrectCount  int
		WrongCount    int
		TimeStats     map[int]float64
	}

	score := 0
	analytics := &AnswerAnalytics{
		Answers:         answers,
		CorrectCount: 	 0,
		WrongCount:      0,
		TimeStats:       make(map[int]float64),
	}

	var prevTimeStat float64 = 0.0

	questions := quizData["questions"].([]any)
	for qIDx, q := range questions {
		question := q.(map[string]any)
		qID := int(question["id"].(float64))
		ans, exists := answers[qID]
		if !exists || ans.Answer == nil {
			analytics.WrongCount = analytics.WrongCount + 1
			continue
		}

		switch strings.TrimSpace(strings.ToLower(question["type"].(string))) {
		case "mcq":
			options := question["options"].([]any)
			for _, optionMap := range options {
				opVal := strings.TrimSpace(strings.ToLower(optionMap.(map[string]any)["option"].(string)))
				opCorrectness := optionMap.(map[string]any)["correct"].(bool)
				if opCorrectness && opVal == analytics.Answers[qID].Answer.([]any)[0].(string) {
					score += int(question["points"].(float64))
					analytics.CorrectCount = analytics.CorrectCount + 1	
				} else {
					analytics.WrongCount = analytics.WrongCount + 1
				}
			}
		case "msq":
			options := question["options"].([]any)
			tmpIdxCount := 0
			ans, exists := answers[qID]
			if !exists || ans.Answer == nil {
				analytics.WrongCount = analytics.WrongCount + 1
				continue
			}
			for _, optionMap := range options {
				opVal := strings.TrimSpace(strings.ToLower(optionMap.(map[string]any)["option"].(string)))
				opCorrectness := optionMap.(map[string]any)["correct"].(bool)
				
				if opCorrectness && tmpIdxCount < len(analytics.Answers[qID].Answer.([]any)) && opVal == answers[qID].Answer.([]any)[tmpIdxCount].(string) {
					score += int(question["points"].(float64))
					analytics.CorrectCount = analytics.CorrectCount + 1
					tmpIdxCount++
				} else if !opCorrectness && tmpIdxCount < len(answers[qID].Answer.([]any)) && opVal == answers[qID].Answer.([]any)[tmpIdxCount].(string) {
					score -= int(question["points"].(float64))
					analytics.WrongCount = analytics.WrongCount + 1
					tmpIdxCount++
				}
			}
		case "numeric":
			ans, exists := answers[qID]
			if !exists || ans.Answer == nil {
				analytics.WrongCount = analytics.WrongCount + 1
				continue
			}
			correctAns := question["correct_answer"]
			if ans, exists := answers[qID]; exists {
				if ans.Answer.(float64) == correctAns {
					score += int(question["points"].(float64))
					analytics.CorrectCount = analytics.CorrectCount + 1
				} else {
					analytics.WrongCount = analytics.WrongCount + 1
				}
			}
		}
		analytics.TimeStats[qIDx] = float64((answers[qID].Timestamp - quizData["event_start_time"].(int64)) / 1000.0) - prevTimeStat
		prevTimeStat = analytics.TimeStats[qIDx]
	}
	log.Println("Ending calculateResults function .....")
	mapified, err := StructToMap(analytics)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, nil
	}

	fmt.Println("Mapified:", mapified)
	return score, mapified
}



func StructToMap(data any) (map[string]any, error) {
    var result map[string]any
    
    jsonData, err := json.Marshal(data)
    if err != nil {
        return nil, err
    }

    if err := json.Unmarshal(jsonData, &result); err != nil {
        return nil, err
    }

    return result, nil
}