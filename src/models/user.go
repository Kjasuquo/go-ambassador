package models

import (
	"gorm.io/gorm"
)

type User struct {
	Model
	FirstName    string   `json:"first_name"`
	LastName     string   `json:"last_name"`
	Email        string   `json:"email" gorm:"unique"`
	Password     []byte   `json:"-"`
	IsAmbassador bool     `json:"is_ambassador"`
	Revenue      *float64 `json:"revenue,omitempty" gorm:"-"`
}

func (user *User) Name() string {
	return user.FirstName + " " + user.LastName
}

type Admin User

func (admin *Admin) CalculateRevenue(db *gorm.DB) {
	var orders []Order

	db.Preload("OrderItems").Find(&orders, &Order{
		UserId:   admin.Id,
		Complete: true,
	})

	var revenue float64 = 0

	for _, order := range orders {
		for _, orderItem := range order.OrderItems {
			revenue += orderItem.AdminRevenue
		}
	}

	admin.Revenue = &revenue
}

type Ambassador User

func (ambassador *Ambassador) CalculateRevenue(db *gorm.DB) {
	var orders []Order

	db.Preload("OrderItems").Find(&orders, &Order{
		UserId:   ambassador.Id,
		Complete: true,
	})

	var revenue float64 = 0.0

	for _, order := range orders {
		for _, orderItem := range order.OrderItems {
			revenue += orderItem.AmbassadorRevenue
		}
	}

	ambassador.Revenue = &revenue
}
