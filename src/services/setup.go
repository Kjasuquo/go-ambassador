package services

import "github.com/kjasuquo/go-user-service"

var UserService services.Service

func Setup() {
	UserService = services.CreatService("http://users-ms:8000/api/")

}
