package controllers

import (
    "net/http"
    "github.com/ownode/models"
	"gopkg.in/mgo.v2/bson"
    "github.com/ownode/services"
    "github.com/ownode/config"
    "github.com/go-martini/martini"
    validator "github.com/asaskevich/govalidator"
)

var Identity IdentityController

type identityCreateBody struct {
	FullName string    `json:"full_name"`
	Email string
    ObjectName string   `json:"object_name"`
    BaseCurrency string `json:"base_currency"`
}

type soulRenewBody struct {
    IdentityId string  `json:"identity_id"`
    SoulBalance float64 `json:"soul_balance"`
}

func init() {
    Identity = IdentityController{ &Base }
}

type IdentityController struct {
    *BaseController
}

// create an identity
func (c *IdentityController) Create(res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
    
    // parse request body
    var body identityCreateBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // full name is required
    if validator.IsNull(body.FullName) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: full_name")
        return
    }

    // email is required
    if validator.IsNull(body.Email) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: email")
        return
    }

    // email is required
    if !validator.IsEmail(body.Email) {
        services.Res(res).Error(400, "invalid_email", "email is invalid")
        return
    }

    // create identity
    newIdentity := models.Identity {
        ObjectID: bson.NewObjectId().Hex(),
        FullName: body.FullName,
        Email: body.Email,
    }

    // if request is from Issuer controller, set issuer field to true
    if d := req.GetData("isIssuer"); d != nil && d.(bool) {

        // object is required
        if validator.IsNull(body.ObjectName) {
            services.Res(res).Error(400, "missing_parameter", "Missing required field: object_name")
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

        newIdentity.Issuer = true
        newIdentity.ObjectName = body.ObjectName
        newIdentity.BaseCurrency = body.BaseCurrency
    }

    err := models.CreateIdentity(db.GetPostgresHandle(), &newIdentity)
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // create response
    respObj, _ := services.StructToJsonToMap(newIdentity)

    if newIdentity.Issuer {
        respObj["soul_balance"] = newIdentity.SoulBalance
    }
    
    services.Res(res).Json(respObj)

}

// renew an issuer identity soul
func (c *IdentityController) RenewSoul(res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
    
    // parse request body
    var body soulRenewBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // identity id is required
    if validator.IsNull(body.IdentityId) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: identity_id")
        return
    }

    // soul balance is required
    if body.SoulBalance == 0 {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: soul_balance")
        return
    }

    // soul balance must be greater than zero
    if body.SoulBalance < MinimumObjectUnit  {
        services.Res(res).Error(400, "invalid_soul_balance", "Soul balance must be equal or greater than minimum object unit which is 0.00000001")
        return
    }

    // ensure identity exists
    identity, found, err := models.FindIdentityByObjectID(db.GetPostgresHandle(), body.IdentityId)
    if !found {
        services.Res(res).Error(404, "invalid_identity", "identity_id is unknown")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure identity is an issuer
    if !identity.Issuer {
        services.Res(res).Error(400, "invalid_identity", "identity is not an issuer")
        return
    }

    // add to soul balance
    newIdentity, err := models.AddToSoulByObjectID(db.GetPostgresHandle(), identity.ObjectID, body.SoulBalance)
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // create response, hide some fields
    respObj, _ := services.StructToJsonToMap(newIdentity)

    if newIdentity.Issuer {
        respObj["soul_balance"] = newIdentity.SoulBalance
    }

    services.Res(res).Json(respObj)
}

// get an identity
func(c *IdentityController) Get(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
    
    identity, found, err := models.FindIdentityByObjectID(db.GetPostgresHandle(), params["id"])
    if !found {
        services.Res(res).Error(404, "not_found", "identity was not found")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // create response 
    respObj, _ := services.StructToJsonToMap(identity)

    if identity.Issuer {
        respObj["soul_balance"] = identity.SoulBalance
    }

    services.Res(res).Json(respObj)
}