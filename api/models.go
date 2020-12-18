package api

import (
	"encoding/json"
	"time"
)

type token struct {
	AccessTkn  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
	RefreshTkn string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
}

type user struct {
	ID        uint      `json:"user_id" example:"1"`
	Email     string    `json:"email" example:"john.doe@email.com"`
	Password  string    `json:"password" example:"passw0rd"`
	CreatedAt time.Time `json:"created_at" example:"2019-11-09T21:21:46+00:00"`
	UpdatedAt time.Time `json:"updated_at"  example:"2019-11-09T21:21:46+00:00"`
	LastLogin time.Time `json:"last_login"  example:"2019-11-09T21:21:46+00:00"`
}

type userRequest struct {
	Email    string `json:"email" example:"john.doe@email.com"`
	Password string `json:"password" example:"passw0rd"`
}

type userResponse struct {
	ID        uint      `json:"user_id" example:"1"`
	Email     string    `json:"email" example:"john.doe@email.com"`
	CreatedAt time.Time `json:"created_at" example:"2019-11-09T21:21:46+00:00"`
	UpdatedAt time.Time `json:"updated_at"  example:"2019-11-09T21:21:46+00:00"`
	LastLogin time.Time `json:"last_login"  example:"2019-11-09T21:21:46+00:00"`
}

type pet struct {
	ID        uint      `json:"pet_id"`
	UserID    uint      `json:"user_id"`
	Name      string    `json:"name" example:"Fido"`
	Type      string    `json:"type" example:"Dog"`
	Gender    string    `json:"gender" example:"Female"`
	Breed     string    `json:"breed" example:"Lab/Terrier Mix"`
	Birthday  time.Time `json:"birthday" example:"2019-11-09T21:21:46+00:00"`
	CreatedAt time.Time `json:"created_at"  example:"2019-11-09T21:21:46+00:00"`
	UpdatedAt time.Time `json:"updated_at"  example:"2019-11-09T21:21:46+00:00"`
}

type petRequest struct {
	Name     string    `json:"name" example:"Fido"`
	Type     string    `json:"type" example:"Dog"`
	Breed    string    `json:"breed" example:"Lab/Terrier Mix"`
	Gender   string    `json:"gender" example:"Female"`
	Birthday time.Time `json:"birthday" example:"2019-11-09T21:21:46+00:00"`
}

type emptyBody struct{}

type pets []pet

func (u user) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        uint      `json:"id"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		LastLogin time.Time `json:"last_login"`
	}{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		LastLogin: u.LastLogin,
	})
}
