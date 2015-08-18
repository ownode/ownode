package controllers

import (
    "net/http"
    "github.com/ownode/models"
	"gopkg.in/mgo.v2/bson"
    "github.com/ownode/services"
    "github.com/ownode/config"
    "github.com/go-martini/martini"
    "strings"
    validator "github.com/asaskevich/govalidator"
)

var Service ServiceController

type createBody struct {
	FullName string `json:"full_name"`
	ServiceName string `json:"service_name"`
	Description string
	Email string `json:email`
}

type enableIssuerBody struct {
	ObjectName string `json:"object_name"`
	ServiceID string `json:"service_id"`
	BaseCurrency string `json:"base_currency"`
}

func init() {
    Service = ServiceController{ &Base }
}

type ServiceController struct {
    *BaseController
}

// create a service
func (c *ServiceController) Create(res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {

	// parse request body
	var body createBody
	if err := c.ParseJsonBody(req, &body); err != nil {
		services.Res(res).Error(400, "invalid_client", "request body is invalid or malformed. Expects valid json body")
		return 
	}

	// name is required
	if c.validate.IsEmpty(body.FullName) {
		services.Res(res).Error(400, "missing_parameter", "Missing required field: full_name")
		return
	}

	// full name must have max of 60 characters
	if !validator.StringLength(body.FullName, "1", "60") {
		services.Res(res).Error(400, "invalid_full_name", "full_name character limit is 60")
		return
	}

	// service name is required
	if c.validate.IsEmpty(body.ServiceName) {
		services.Res(res).Error(400, "missing_parameter", "Missing required field: service_name")
		return
	}

	// service name must have max of 30 characters
	if !validator.StringLength(body.ServiceName, "1", "60") {
		services.Res(res).Error(400, "invalid_service_name", "service_name character limit is 60")
		return
	}

	// description is required
	if c.validate.IsEmpty(body.Description) {
		services.Res(res).Error(400, "missing_parameter", "Missing required field: description")
		return
	}

	// description must have max of 140 characters
	if !validator.StringLength(body.Description, "1", "140") {
		services.Res(res).Error(400, "invalid_description", "description character limit is 140")
		return
	}

	// email is required
	if c.validate.IsEmpty(body.Email) {
		services.Res(res).Error(400, "missing_parameter", "Missing required field: email")
		return
	}

	// email must be valid
	if !validator.IsEmail(body.Email) {
		services.Res(res).Error(400, "invalid_email", "email is invalid")
		return
	}

	// create identity
	newIdentity := &models.Identity {
        ObjectID: bson.NewObjectId().Hex(),
        FullName: body.FullName,
        Email: body.Email,
    }

    // start db transaction
    dbTx := db.GetPostgresHandle().Begin()

    if err := models.CreateIdentity(dbTx, newIdentity); err != nil {
    	dbTx.Rollback()
    	c.log.Error(err.Error())
		services.Res(res).Error(500, "", "server error")
		return
    }

	// create client credentials
	clientId := services.GetRandString(services.GetRandNumRange(32, 42))
	clientSecret := services.GetRandString(services.GetRandNumRange(32, 42))

	// create new service object
	newService := models.Service {
		ObjectID: bson.NewObjectId().Hex(),
		Name: body.ServiceName,
		Description: body.Description,
		ClientID: clientId,
		ClientSecret: clientSecret,
		Identity: newIdentity,
	} 

	// create service
	err := models.CreateService(dbTx, &newService)
	if err != nil {
		dbTx.Rollback()
		c.log.Error(err.Error())
		services.Res(res).Error(500, "", "server error")
		return
	}

	// commit db transaction
	dbTx.Commit()

	// send response
	respObj, _ := services.StructToJsonToMap(newService)
	services.Res(res).Json(respObj)
}

// get a service
func(c *ServiceController) Get(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
	
	service, found, err := models.FindServiceByObjectID(db.GetPostgresHandle(), params["id"])
    if !found {
        services.Res(res).Error(404, "not_found", "service was not found")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    respObj, _ := services.StructToJsonToMap(service)
	services.Res(res).Json(respObj)
}

// enable a service to issuer status
func(c *ServiceController) EnableIssuer(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
	
	// parse request body
	var body enableIssuerBody
	if err := c.ParseJsonBody(req, &body); err != nil {
		services.Res(res).Error(400, "invalid_client", "request body is invalid or malformed. Expects valid json body")
		return 
	}

	// service id is required
	if c.validate.IsEmpty(body.ServiceID) {
		services.Res(res).Error(400, "missing_parameter", "Missing required field: service_id")
		return
	}

	// ensure service exists
	service, found, err := models.FindServiceByObjectID(db.GetPostgresHandle(), body.ServiceID)
	if err != nil {
		c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
	} else if !found {
		services.Res(res).Error(404, "not_found", "service was not found")
        return
	}

	// object name is required
	if c.validate.IsEmpty(body.ObjectName) {
		services.Res(res).Error(400, "missing_parameter", "Missing required field: object_name")
		return
	}

	// ensure no other service has used the object name
	identity, found, err := models.FindIdentityByObjectName(db.GetPostgresHandle(), body.ObjectName)
	if err != nil {
		c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
	} else if found && identity.ObjectID != service.Identity.ObjectID {
		services.Res(res).Error(400, "invalid_object_name", "object name is not available, try a unique name")
        return
	}

	// base currency is required
    if validator.IsNull(body.BaseCurrency) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: base_currency")
        return
    }

    // base currency must be supported
    if !services.StringInStringSlice(config.SupportedBaseCurrencies, body.BaseCurrency){
        services.Res(res).Error(400, "invalid_base_currency", "base currency is unknown")
        return
    }

	// set issuer to true
	service.Identity.Issuer = true
	service.Identity.ObjectName = strings.ToLower(body.ObjectName)
	service.Identity.BaseCurrency = body.BaseCurrency
	db.GetPostgresHandle().Save(&service)

	respObj, _ := services.StructToJsonToMap(service)
	respObj["identity"].(map[string]interface{})["soul_balance"] = service.Identity.SoulBalance
	services.Res(res).Json(respObj)
}	