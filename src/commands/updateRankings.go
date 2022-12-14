package main

import (
	"ambassador/src/database"
	"ambassador/src/models"
	"ambassador/src/services"
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"log"
)

func main() {
	database.Connect()
	database.SetupRedis()

	ctx := context.Background()

	response, err := services.UserService.Get("users", "")
	if err != nil {
		log.Fatalf("na here error dey: %v\n", err)
	}

	var users []models.User

	json.NewDecoder(response.Body).Decode(&users)

	for _, user := range users {
		if user.IsAmbassador {
			ambassador := models.Ambassador(user)
			ambassador.CalculateRevenue(database.DB)

			database.Cache.ZAdd(ctx, "rankings", &redis.Z{
				Score:  *ambassador.Revenue,
				Member: user.Name(),
			})
		}
	}
}
