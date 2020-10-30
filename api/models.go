package api

import (
	"encoding/json"
	"time"
)

type token struct {
	AccessTkn  string `json:"access_token"`
	RefreshTkn string `json:"refresh_token"`
}

type user struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastLogin time.Time `json:"last_login"`
}

type pet struct {
	ID       uint      `json:"pet_id"`
	UserID   uint      `json:"user_id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Breed    string    `json:"breed"`
	Birthday time.Time `json:"birthday"`
}
type pets []pet

// func userJSONReturn(u user) user {
// 	return user{
// 		ID:        u.ID,
// 		Email:     u.Email,
// 		CreatedAt: u.CreatedAt,
// 		UpdatedAt: u.UpdatedAt,
// 		LastLogin: u.LastLogin,
// 	}
// }

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
