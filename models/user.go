package models

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	RoleWriter UserRole = "writer"
	RoleEditor UserRole = "editor"
	RoleAdmin  UserRole = "admin"
)

type User struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"not null"`
	Role      UserRole       `json:"role" gorm:"default:'writer'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
