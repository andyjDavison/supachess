package transport

import (
	"api/internal/domain"
	"api/internal/user"
	"encoding/json"
	"errors"
	"net/http"
)

type RegisterHandler struct {
	userService *user.UserService
}

func NewRegisterHandler(userService *user.UserService) *RegisterHandler {
	return &RegisterHandler{userService: userService}
}

// RegisterUserRequest defines the expected incoming JSON payload
type RegisterUserRequest struct {
	Username  string `json:"username"`
	Email string `json:"email"`
	Password string `json:"password"`
	Rating int32 `json:"rating"`
}

// RegisterUserResponse defines the structural JSON output
type RegisterUserResponse struct {
	ID    string  `json:"id"`
	Username  string `json:"username"`
	Rating int32 `json:"rating"`
}

func (handler *RegisterHandler) CreateUserHandler(writer http.ResponseWriter, request *http.Request) {
	
	var req RegisterUserRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		http.Error(writer, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	// 4. Invoke the service layer, automatically forwarding the request context (r.Context())
	// This ensures that if the client disconnects, your database operation terminates immediately.
	createdUser, err := handler.userService.CreateUser(request.Context(), user.UserRegistrationInput{
		Username: req.Username,
		Email: req.Email,
		Password: req.Password,
		Rating: req.Rating,
	})
	if err != nil {
		// In a production environment, parse the error type to send appropriate status codes
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated) // HTTP 201

	resp := RegisterUserResponse{
		ID:    createdUser.UserID,
		Username:  createdUser.Username,
		Rating: createdUser.Rating,
	}
	json.NewEncoder(writer).Encode(resp)
}

func (handler *RegisterHandler) FindUserByIdHandler(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		http.Error(writer, "Missing user id", http.StatusBadRequest)
		return
	}

	foundUser, err := handler.userService.FindUserById(request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			http.Error(writer, "user not found", http.StatusNotFound)
			return
		}
		http.Error(writer, "internal server error", http.StatusInternalServerError)
		return 
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // HTTP 200

	resp := RegisterUserResponse{
		ID:    foundUser.UserID,
		Username:  foundUser.Username,
		Rating: foundUser.Rating,
	}
	json.NewEncoder(writer).Encode(resp)
}

func (handler *RegisterHandler) FindUserByEmailHandler(writer http.ResponseWriter, request *http.Request) {
	email := request.PathValue("email")
	if email == "" {
		http.Error(writer, "Missing email", http.StatusBadRequest)
		return
	}

	foundUser, err := handler.userService.FindUserByEmail(request.Context(), email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			http.Error(writer, "user not found", http.StatusNotFound)
			return
		}
		http.Error(writer, "internal server error", http.StatusInternalServerError)
		return 
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // HTTP 200

	resp := RegisterUserResponse{
		ID:    foundUser.UserID,
		Username:  foundUser.Username,
		Rating: foundUser.Rating,
	}
	json.NewEncoder(writer).Encode(resp)
}

func (handler *RegisterHandler) FindUserByUsernameHandler(writer http.ResponseWriter, request *http.Request) {
	username := request.PathValue("username")
	if username == "" {
		http.Error(writer, "Missing username", http.StatusBadRequest)
		return
	}

	foundUser, err := handler.userService.FindUserByUsername(request.Context(), username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			http.Error(writer, "user not found", http.StatusNotFound)
			return
		}
		http.Error(writer, "internal server error", http.StatusInternalServerError)
		return 
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // HTTP 200

	resp := RegisterUserResponse{
		ID:    foundUser.UserID,
		Username:  foundUser.Username,
		Rating: foundUser.Rating,
	}
	json.NewEncoder(writer).Encode(resp)
}
