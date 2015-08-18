package controllers

var APP AppController

func init() {
	APP = AppController{ &Base }
}

type AppController struct {
	*BaseController
}

func (c *AppController) Index() string {
	return "Hello!"
}