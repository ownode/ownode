package controllers

import (
    "net/http"
    // "github.com/ownode/models"
	// "gopkg.in/mgo.v2/bson"
    "github.com/ownode/services"
    // "time"
    // validator "github.com/asaskevich/govalidator"
)

var Issuer IssuerController

func init() {
    Issuer = IssuerController{ &Base }
}

type IssuerController struct {
    *BaseController
}

// create an issuer enabled identity
func (c *IssuerController) Create(res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
    req.SetData("isIssuer", true)
    Identity.Create(res, req, db)
}