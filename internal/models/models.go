package models

import "time"

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"` // Nicht nach außen geben
}

type BottleSize struct {
	ID     int `json:"id"`
	SizeML int `json:"size_ml"`
}

type WaterIntake struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Date       time.Time `json:"date"`
	BottleSize int       `json:"bottle_size_ml"`
	Quantity   int       `json:"quantity"`
}

type WaterIntakeStats struct {
	Total   int           `json:"total"`
	Intakes []WaterIntake `json:"intakes"`
}
