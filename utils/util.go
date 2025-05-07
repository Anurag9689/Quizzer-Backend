package utils


import (
	"os"
	"fmt"
	"errors"
	"strings"
	"net/http"
	"net/smtp"
	"encoding/json"

	"OnlineQuizSystem/db"
	"OnlineQuizSystem/models"

	"github.com/golang-jwt/jwt/v5"
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




