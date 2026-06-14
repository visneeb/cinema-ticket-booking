package model

import "time"

// Role is the user's permission level.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User is stored in the cinema.users collection.
type User struct {
	UID         string    `bson:"_id"          json:"uid"`
	Email       string    `bson:"email"        json:"email"`
	DisplayName string    `bson:"display_name" json:"display_name"`
	PhotoURL    string    `bson:"photo_url"    json:"photo_url"`
	Role        Role      `bson:"role"         json:"role"`
	CreatedAt   time.Time `bson:"created_at"   json:"created_at"`
	LastLoginAt time.Time `bson:"last_login_at" json:"last_login_at"`
}

