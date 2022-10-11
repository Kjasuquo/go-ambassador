package controllers

import (
	"ambassador/src/models"
	"ambassador/src/services"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"log"
	"strconv"
	"strings"
	"time"
)

func Register(c *fiber.Ctx) error {
	var data map[string]string

	if err := c.BodyParser(&data); err != nil {
		return err
	}

	data["is_ambassador"] = strconv.FormatBool(strings.Contains(c.Path(), "/api/ambassador"))

	//jsonDate, err := json.Marshal(data)
	//if err != nil {
	//	log.Fatalf("Can't marshal data into json: %v\n", err)
	//}
	//response, err := http.Post("http://host.docker.internal:8001/api/register", "application/json", bytes.NewBuffer(jsonDate))

	//response, err := services.request("POST", "register", "", data)

	response, err := services.UserService.Post("register", "", data)
	if err != nil {
		log.Fatalf("didn't get a response from post request: %v\n", err)
	}

	var user models.User
	err = json.NewDecoder(response.Body).Decode(&user)
	if err != nil {
		log.Fatalf("couldn't decode the response body from the post request: %v\n", err)
	}

	return c.JSON(user)
}

func Login(c *fiber.Ctx) error {
	var data map[string]string

	if err := c.BodyParser(&data); err != nil {
		return err
	}

	isAmbassador := strings.Contains(c.Path(), "/api/ambassador")

	if isAmbassador {
		data["scope"] = "ambassador"
	} else {
		data["scope"] = "admin"
	}

	//jsonDate, err := json.Marshal(data)
	//if err != nil {
	//	log.Fatalf("Can't marshal data into json: %v\n", err)
	//}
	//response, err := http.Post("http://host.docker.internal:8001/api/login", "application/json", bytes.NewBuffer(jsonDate))

	//response, err := services.request("POST", "login", "", data)

	response, err := services.UserService.Post("login", "", data)
	if err != nil {
		log.Fatalf("didn't get a response from post request: %v\n", err)
	}

	var token map[string]string
	err = json.NewDecoder(response.Body).Decode(&token)
	if err != nil {
		log.Fatalf("couldn't decode the response body from the post request: %v\n", err)
	}

	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    token["jwt"],
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	}

	c.Cookie(&cookie)

	return c.JSON(fiber.Map{
		"message": "success",
	})
}

func User(c *fiber.Ctx) error {
	//req, err := http.NewRequest("GET", "http://host.docker.internal:8001/api/user", nil)
	//if err != nil {
	//	log.Fatalf("error in NewRequest: %v\n", err)
	//}
	//
	//req.Header.Add("Cookie", "jwt="+c.Cookies("jwt", ""))
	//
	//client := &http.Client{}
	//response, err := client.Do(req)

	//response, err := services.UserService.request("GET", "user", c.Cookies("jwt", ""), nil)

	return c.JSON(c.Context().UserValue("user"))
}

func Logout(c *fiber.Ctx) error {
	services.UserService.Post("logout", c.Cookies("jwt", ""), nil)

	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	}

	c.Cookie(&cookie)

	return c.JSON(fiber.Map{
		"message": "success",
	})
}

func UpdateInfo(c *fiber.Ctx) error {
	var data map[string]string

	if err := c.BodyParser(&data); err != nil {
		log.Fatalf("cannot parse data: %v\n", err)
	}

	response, err := services.UserService.Put("users/info", c.Cookies("jwt", ""), data)
	if err != nil {
		log.Fatalf("didn't get a response from post request: %v\n", err)
	}

	var user models.User
	err = json.NewDecoder(response.Body).Decode(&user)
	if err != nil {
		log.Fatalf("couldn't decode the response body from the post request: %v\n", err)
	}

	return c.JSON(user)
}

func UpdatePassword(c *fiber.Ctx) error {
	var data map[string]string

	if err := c.BodyParser(&data); err != nil {
		return err
	}

	response, err := services.UserService.Put("users/password", c.Cookies("jwt", ""), data)
	if err != nil {
		log.Fatalf("didn't get a response from post request: %v\n", err)
	}

	var user models.User
	err = json.NewDecoder(response.Body).Decode(&user)
	if err != nil {
		log.Fatalf("couldn't decode the response body from the post request: %v\n", err)
	}

	return c.JSON(user)
}
