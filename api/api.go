package api

import (
	"os"
	// "io"
	"log"
	"fmt"
	"time"
	"strconv"
	"strings"
	"net/http"
	"encoding/json"

	"OnlineQuizSystem/db"
	"OnlineQuizSystem/utils"
	"OnlineQuizSystem/models"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` // admin, teacher, student  
	Otp      string `json:"otp"`
	AdminKey string `json:"admin_key"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ForgotPassword struct {
	Email string `json:"email"`
}

type ChangePassword struct {
	Email 		string `json:"email"`
	Password 	string `json:"password"`
	Otp 		string `json:"otp"`
}





var newRegisterStore = make(map[string]RegisterRequest)


var passwordForgottenStore = make(map[string]ChangePassword)










func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRegisterHandler handling request: ", r)
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	req.Role = strings.TrimSpace(strings.ToLower(req.Role))

	if ( req.Role != "admin") && (req.Role != "teacher") && (req.Role != "student") {
		s := `Role of type '%s' is not valid. Only one of the three roles (admin, teacher & student) can be assigned to a user.`
		http.Error(w, fmt.Sprintf(s, req.Role), http.StatusPreconditionFailed)
		return 
	}

	if ( req.Role == "admin" && req.AdminKey != os.Getenv("ADMIN_KEY")){
		http.Error(w, "To become an Admin user, Contact the developer - Anurag", http.StatusForbidden)
		return
	}

	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)

	newRegisterStore[req.Email] = req

	log.Printf("Sending OTP to %s: %s\n", req.Email, otp)
	
	if strings.ToLower(strings.TrimSpace(os.Getenv("SEND_OTP_FLAG"))) == "yes" {
		err := utils.SendOtpEmail(req.Email, otp, "Your OTP for Email Verification from Online Quiz System")
		if err != nil {
			log.Println("Failed to send OTP on email:", err)
			http.Error(w, "Unable to send OTP", http.StatusForbidden)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "Verification OTP sent to email"})
	} else {
		json.NewEncoder(w).Encode(map[string]string{"message": "Verification OTP sent to email", "otp" : otp})
	}
}












func VerifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nVerifyEmailHandler handling request: ", r)
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	fmt.Println("New register store: ", newRegisterStore)
	var storedReq RegisterRequest
	storedReq, exists := newRegisterStore[req.Email]
	if !exists || req.OTP != storedReq.Otp {
		http.Error(w, "Invalid OTP or Email", http.StatusUnauthorized)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(storedReq.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	user := models.User{
		Email:    storedReq.Email,
		Password: string(hashedPassword),
		UserType: storedReq.Role,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		http.Error(w, "User already exists or DB error", http.StatusBadRequest)
		return
	}

	fmt.Println("user.ID: ", user.ID)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id" : user.ID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	// Remove below 2 lines
	s := token.Claims.(jwt.MapClaims)
	fmt.Println(s)

	secret := os.Getenv("SECRET_KEY")
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	delete(newRegisterStore, storedReq.Email)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}














func LoginHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nLoginHandler handling request: ", r)
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := db.DB.Where("email = ?", strings.TrimSpace(req.Email)).First(&user).Error; err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id" : user.ID,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	secret := os.Getenv("SECRET_KEY")
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}











func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nForgotPasswordHandler handling request: ", r)
	var req ForgotPassword
	var fpCompleteReq ChangePassword
	var user models.User
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Body Value", http.StatusUnprocessableEntity)
	}

	req.Email = strings.TrimSpace(req.Email)
	result := db.DB.Last(&user, "email = ?", req.Email)
	if result.Error != nil {
		http.Error(w, fmt.Sprintf("User with email('%s') do not exist", strings.TrimSpace(req.Email)), http.StatusNotFound)
		return 
	}

	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)	
	fpCompleteReq.Email = user.Email
	fpCompleteReq.Otp = otp
	passwordForgottenStore[user.Email] = fpCompleteReq

	err := utils.SendOtpEmail(req.Email, otp, "Your OTP for forgot password identity verification from Online Quiz System")
	if err != nil {
		log.Println("Failed to send OTP on email:", err)
		http.Error(w, "Unable to send OTP", http.StatusForbidden)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"message": "Forgot password OTP sent to email"})
}














func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nChangePasswordHandler handling request: ", r)
	var req ChangePassword
	var user models.User

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Body Value", http.StatusUnprocessableEntity)
	}

	req.Email = strings.TrimSpace(req.Email)
	result := db.DB.Last(&user, "email = ?", req.Email)
	if result.Error != nil {
		http.Error(w, fmt.Sprintf("User with email('%s') do not exist", strings.TrimSpace(req.Email)), http.StatusNotFound)
		return 
	}

	if strings.TrimSpace(req.Otp) != strings.TrimSpace(passwordForgottenStore[user.Email].Otp) {
		http.Error(w, "OTP did not match", http.StatusUnauthorized)
		return 
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal error : %s", err.Error()), http.StatusInternalServerError)
		return
	}

	uResult := db.DB.Model(&user).Update("password", string(hashedPassword))
	if uResult.Error != nil {
		http.Error(w, fmt.Sprintf("Internal error : %s", uResult.Error.Error()), http.StatusExpectationFailed)
		return 
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password is successfully changed."})
}




























// ################################################ Get Current User Profile #################################
func RetrieveCurrentUserProfileHandler(w http.ResponseWriter, r *http.Request){
	log.Println("\n\nCreateUserHandler handling request: ", r)
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}


















































/* ########################################## USER MODEL FUNCTIONS ################################## */

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nCreateUserHandler handling request: ", r)
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if user.UserType != "admin" {
		http.Error(w, "Unauthorized: Only admins can create users", http.StatusUnauthorized)
		return
	}

	var newUser models.User
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}
	newUser.Password = string(hashedPassword)

	if err := db.DB.Create(&newUser).Error; err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}








func RetrieveUserListHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRetrieveUserListHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var users []models.User
	if err := db.DB.Preload("UserDetails").Preload("QuizEvents").Preload("EventResults").Find(&users).Error; err != nil {
		http.Error(w, "Could not fetch users", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(users)
}








func RetrieveUserDetailHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRetrieveUserDetailHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := db.DB.Preload("UserDetails").Preload("QuizEvent").Preload("EventResult").First(&user, id).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}







func UpdateUserPatchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nUpdateUserPatchHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Do not allow password updates here without hashing
	if password, ok := updates["password"]; ok {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password.(string)), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
		updates["password"] = string(hashedPassword)
	}

	if err := db.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	if _, ok := updates["deleted_at"]; ok {
		var user models.User
		if err := db.DB.Unscoped().First(&user, id).Error; err != nil {
			http.Error(w, "User not found (even among soft-deleted)", http.StatusNotFound)
			return
		}
		if err := db.DB.Unscoped().Model(&user).Update("deleted_at", nil).Error; err != nil {
			http.Error(w, "Failed to restore user", http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "User updated successfully"})
}







func SoftDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nSoftDeleteUserHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := db.DB.Delete(&models.User{}, id).Error; err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User soft-deleted successfully"})
}


























































/* ########################################## USER DETAIL MODEL FUNCTIONS ################################## */

func CreateUserDetailsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nCreateUserDetailsHandler handling request: ", r)
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if user.UserType != "admin" {
		http.Error(w, "Unauthorized: Only admins can create userDetails details", http.StatusUnauthorized)
		return
	}

	var newUserDetails models.UserDetails
	if err := json.NewDecoder(r.Body).Decode(&newUserDetails); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if newUserDetails.UserID != 0 {
		var user models.User
		result := db.DB.First(&user, newUserDetails.UserID)
		if result.Error != nil {
			http.Error(w, 
				fmt.Sprintf("User id is present but User not found in the database: %s", result.Error.Error()), 
				http.StatusNotFound)
			return 
		}
		newUserDetails.User = &user;
	}

	if err := db.DB.Create(&newUserDetails).Error; err != nil {
		http.Error(w, "Failed to create userDetail: "+err.Error(), http.StatusBadRequest)
		return
	}



	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUserDetails)
}








func RetrieveUserDetailsListHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRetrieveUserDetailsListHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var listUserDetails []models.UserDetails
	if err := db.DB.Find(&listUserDetails).Error; err != nil {
		http.Error(w, "Could not fetch list of userDetails", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(listUserDetails)
}








func RetrieveUserDetailsDetailHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRetrieveUserDetailsDetailHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var userDetails models.UserDetails
	if err := db.DB.First(&userDetails, id).Error; err != nil {
		http.Error(w, "UserDetails not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(userDetails)
}







func UpdateUserDetailsPatchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nUpdateUserDetailsPatchHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := db.DB.Model(&models.UserDetails{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update userDetails", http.StatusInternalServerError)
		return
	}

	if _, ok := updates["deleted_at"]; ok {
		var userDetail models.UserDetails
		if err := db.DB.Unscoped().First(&userDetail, id).Error; err != nil {
			http.Error(w, "userDetail not found (even among soft-deleted)", http.StatusNotFound)
			return
		}
		if err := db.DB.Unscoped().Model(&userDetail).Update("deleted_at", nil).Error; err != nil {
			http.Error(w, "Failed to restore userDetails", http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "userDetails updated successfully"})
}







func SoftDeleteUserDetailsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nSoftDeleteUserDetailsHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := db.DB.Delete(&models.UserDetails{}, id).Error; err != nil {
		http.Error(w, "Failed to delete userDetail", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "UserDetail soft-deleted successfully"})
}

























































/* ########################################## QUIZ EVENT MODEL FUNCTIONS ################################## */

func CreateQuizEventHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nCreateQuizEventHandler handling request: ", r)
	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if user.UserType != "admin" && user.UserType != "teacher" {
		http.Error(w, "Unauthorized: Only admins & teachers can create QuizEvent", http.StatusUnauthorized)
		return
	}

	var reqBody map[string]any
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
	}

	quizJson, ok := reqBody["quiz_json"].(map[string]any)
	if !ok {
		http.Error(w, "quiz_json is not a map[string]any type or quiz_json is not provided", http.StatusBadRequest)
		return 
	}
	
	
	var newQuizEvent models.QuizEvent
	if _, err := newQuizEvent.SetQuizJsonFileMap(quizJson); err != nil {
		http.Error(w, "Internal json file writing error "+err.Error(), http.StatusInternalServerError)
		return 
	}

	newQuizEvent.QuizEventName = reqBody["quiz_event_name"].(string)
	newQuizEvent.UserID = user.ID

	if err := db.DB.Create(&newQuizEvent).Error; err != nil {
		http.Error(w, "Failed to create QuizEvent: " + err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newQuizEvent)
}








func RetrieveQuizEventListHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRetrieveQuizEventListHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var listQuizEvent []models.QuizEvent
	if err := db.DB.Preload("EventResult").Find(&listQuizEvent).Error; err != nil {
		http.Error(w, "Could not fetch list of QuizEvents", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(listQuizEvent)
}








func RetrieveQuizEventDetailHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRetrieveQuizEventDetailHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var quizEvent models.QuizEvent
	if err := db.DB.First(&quizEvent, id).Error; err != nil {
		http.Error(w, "QuizEvent not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(quizEvent)
}







func UpdateQuizEventPatchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nUpdateQuizEventPatchHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := db.DB.Model(&models.QuizEvent{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update QuizEvent", http.StatusInternalServerError)
		return
	}

	if _, ok := updates["deleted_at"]; ok {
		var quizEvent models.QuizEvent
		if err := db.DB.Unscoped().First(&quizEvent, id).Error; err != nil {
			http.Error(w, "QuizEvent not found (even among soft-deleted)", http.StatusNotFound)
			return
		}
		if err := db.DB.Unscoped().Model(&quizEvent).Update("deleted_at", nil).Error; err != nil {
			http.Error(w, "Failed to restore QuizEvent", http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "QuizEvent updated successfully"})
}







func SoftDeleteQuizEventHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nSoftDeleteQuizEventHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := db.DB.Delete(&models.QuizEvent{}, id).Error; err != nil {
		http.Error(w, "Failed to delete QuizEvent", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "QuizEvent soft-deleted successfully"})
}



















































/* ########################################## EVENT RESULT MODEL FUNCTIONS ################################## */

func CreateEventResultHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nCreateEventResultHandler handling request: ", r)
	// s, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	http.Error(w, "Internal Error: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// for char := range s {
	// 	fmt.Printf("%c", s[char])
	// }

	user, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if user.UserType != "admin" && user.UserType != "teacher" {
		http.Error(w, "Unauthorized: Only admins & teachers can create EventResult", http.StatusUnauthorized)
		return
	}


	var eventResult models.EventResult
	if err := json.NewDecoder(r.Body).Decode(&eventResult); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
	}
	
	if err := db.DB.Create(&eventResult).Error; err != nil {
		http.Error(w, "Failed to create EventResult: " + err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(eventResult)
}








func RetrieveEventResultListHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRetrieveEventResultListHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var listEventResult []models.EventResult
	if err := db.DB.Preload("QuizEvent").Find(&listEventResult).Error; err != nil {
		http.Error(w, "Could not fetch list of EventResults", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(listEventResult)
}








func RetrieveEventResultDetailHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nRetrieveEventResultDetailHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var eventResult models.EventResult
	if err := db.DB.Preload("QuizEvent").First(&eventResult, id).Error; err != nil {
		http.Error(w, "EventResult not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(eventResult)
}







func UpdateEventResultPatchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nUpdateEventResultPatchHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := db.DB.Model(&models.EventResult{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		http.Error(w, "Failed to update EventResult", http.StatusInternalServerError)
		return
	}

	if _, ok := updates["deleted_at"]; ok {
		var eventResult models.EventResult
		if err := db.DB.Unscoped().First(&eventResult, id).Error; err != nil {
			http.Error(w, "EventResult not found (even among soft-deleted)", http.StatusNotFound)
			return
		}
		if err := db.DB.Unscoped().Model(&eventResult).Update("deleted_at", nil).Error; err != nil {
			http.Error(w, "Failed to restore EventResult", http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "EventResult updated successfully"})
}







func SoftDeleteEventResultHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("\n\nSoftDeleteEventResultHandler handling request: ", r)
	_, _, err := utils.AuthorizeUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := db.DB.Delete(&models.EventResult{}, id).Error; err != nil {
		http.Error(w, "Failed to delete EventResult", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "EventResult soft-deleted successfully"})
}

