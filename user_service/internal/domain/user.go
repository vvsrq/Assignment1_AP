package domain

import "time"

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserProfile struct {
	ID    int64
	Name  string
	Email string
}

type AuthResponse struct {
	Authenticated bool
	Token         string
	UserID        int64
	ErrorMessage  string
}

type UserRepository interface {
	CreateUser(user *User) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id int64) (*User, error)
}

type UserUseCase interface {
	RegisterUser(name, email, password string) (*User, error)
	AuthenticateUser(email, password string) (*AuthResponse, error)
	GetUserProfile(id int64) (*UserProfile, error)
}
