package controllers

import (
	"ambassador/src/database"
	"ambassador/src/models"
	"ambassador/src/services"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"log"
)

func Ambassadors(c *fiber.Ctx) error {

	response, err := services.UserService.Get("users", c.Cookies("jwt", ""))
	if err != nil {
		fmt.Println("error here")
		log.Fatalf("na here error dey: %v\n", err)
	}

	var users []models.User

	var ambassador []models.Ambassador

	json.NewDecoder(response.Body).Decode(&users)

	for _, user := range users {
		if user.IsAmbassador {
			ambassador = append(ambassador, models.Ambassador(user))
		}
	}

	return c.JSON(ambassador)
}

func Rankings(c *fiber.Ctx) error {
	rankings, err := database.Cache.ZRevRangeByScoreWithScores(context.Background(), "rankings", &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()

	if err != nil {
		return err
	}

	result := make(map[string]float64)

	for _, ranking := range rankings {
		result[ranking.Member.(string)] = ranking.Score
	}

	return c.JSON(result)
}
