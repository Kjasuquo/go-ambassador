package controllers

import (
	"ambassador/src/database"
	"ambassador/src/models"
	"ambassador/src/services"
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
)

func Orders(c *fiber.Ctx) error {
	var orders []models.Order

	database.DB.Preload("OrderItems").Find(&orders)

	for i, order := range orders {
		orders[i].Name = order.FullName()
		orders[i].Total = order.GetTotal()
	}

	return c.JSON(orders)
}

type CreateOrderRequest struct {
	Code      string
	FirstName string
	LastName  string
	Email     string
	Address   string
	Country   string
	City      string
	Zip       string
	Products  []map[string]int
}

func CreateOrder(c *fiber.Ctx) error {
	var request CreateOrderRequest

	if err := c.BodyParser(&request); err != nil {
		return err
	}

	link := models.Link{
		Code: request.Code,
	}

	database.DB.First(&link)

	response, err := services.UserService.Get(fmt.Sprintf("users/%d", link.UserId), "")
	if err != nil {
		return err
	}

	var user models.User

	json.NewDecoder(response.Body).Decode(&user)

	if link.Id == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid link!",
		})
	}

	order := models.Order{
		Code:            link.Code,
		UserId:          link.UserId,
		AmbassadorEmail: user.Email,
		FirstName:       request.FirstName,
		LastName:        request.LastName,
		Email:           request.Email,
		Address:         request.Address,
		Country:         request.Country,
		City:            request.City,
		Zip:             request.Zip,
	}

	tx := database.DB.Begin()

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	var lineItems []*stripe.CheckoutSessionLineItemParams

	for _, requestProduct := range request.Products {
		product := models.Product{}
		product.Id = uint(requestProduct["product_id"])
		database.DB.First(&product)

		total := product.Price * float64(requestProduct["quantity"])

		item := models.OrderItem{
			OrderId:           order.Id,
			ProductTitle:      product.Title,
			Price:             product.Price,
			Quantity:          uint(requestProduct["quantity"]),
			AmbassadorRevenue: 0.1 * total,
			AdminRevenue:      0.9 * total,
		}

		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Name:        stripe.String(product.Title),
			Description: stripe.String(product.Description),
			Images:      []*string{stripe.String(product.Image)},
			Amount:      stripe.Int64(100 * int64(product.Price)),
			Currency:    stripe.String("usd"),
			Quantity:    stripe.Int64(int64(requestProduct["quantity"])),
		})
	}

	stripe.Key = "sk_test_51KrgjpAPp5a8lA8fATWqPC7cHAEPUuYf7wTVzotPI9TXyXW8CWBl814zSKCPeYFNoRaowgYa585VDLs5jDmIUa2V00oAk5LeLI"

	params := stripe.CheckoutSessionParams{
		SuccessURL:         stripe.String("http://localhost:5000/success?source={CHECKOUT_SESSION_ID}"),
		CancelURL:          stripe.String("http://localhost:5000/error"),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems:          lineItems,
	}

	source, err := session.New(&params)

	if err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	order.TransactionId = source.ID

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	tx.Commit()

	return c.JSON(source)
}

func CompleteOrder(c *fiber.Ctx) error {
	var data map[string]string

	if err := c.BodyParser(&data); err != nil {
		return err
	}

	order := models.Order{}

	database.DB.Preload("OrderItems").First(&order, models.Order{
		TransactionId: data["source"],
	})

	if order.Id == 0 {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Order not found",
		})
	}

	order.Complete = true
	database.DB.Save(&order)

	go func(order models.Order) {
		ambassadorRevenue := 0.0
		adminRevenue := 0.0

		for _, item := range order.OrderItems {
			ambassadorRevenue += item.AmbassadorRevenue
			adminRevenue += item.AdminRevenue
		}

		response, err := services.UserService.Get(fmt.Sprintf("users/%d", order.UserId), "")
		if err != nil {
			panic(err)
		}

		var user models.User

		json.NewDecoder(response.Body).Decode(&user)

		database.Cache.ZIncrBy(context.Background(), "rankings", ambassadorRevenue, user.Name())

		producer, err := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": "pkc-l6wr6.europe-west2.gcp.confluent.cloud:9092",
			"security.protocol": "SASL_SSL",
			"sasl.username":     "VF4ELHMVULALFWAA",
			"sasl.password":     "6uW09Sxq4cTSjWHJd48BellhYTdGm1PpbJ9BJzvtGjdun5zCOfbyObPTErftaQk8",
			"sasl.mechanism":    "PLAIN",
		})
		if err != nil {
			panic(err)
		}

		defer producer.Close()

		topic := "default"
		message := map[string]interface{}{
			"id":                 order.Id,
			"ambassador_revenue": ambassadorRevenue,
			"admin_revenue":      adminRevenue,
			"code":               order.Code,
			"ambassador_email":   order.AmbassadorEmail,
		}

		value, _ := json.Marshal(message)

		producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(value),
		}, nil)

		producer.Flush(15000)
	}(order)

	return c.JSON(fiber.Map{
		"message": "success",
	})
}
