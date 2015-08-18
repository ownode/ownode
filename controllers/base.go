package controllers

import (
	"github.com/ownode/config"
	"encoding/json"
	"github.com/ownode/services"
)

var MinimumObjectUnit = 0.00000001
var MaxMetaSize = 51200
var Base BaseController

func init() {
	Base = BaseController{
		log: config.Log(),
		validate: &services.CustomValidator{},
	}
}

type BaseController struct {
	log *config.CustomLog
	validate *services.CustomValidator
}

// Parse json request body to struct
func (base *BaseController) ParseJsonBody(req services.AuxRequestContext, obj interface{}) error {
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(obj)
	if err != nil {
		return err
	} 
	return nil
}

