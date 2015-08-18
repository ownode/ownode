package controllers

import (
    "net/http"
    "github.com/ownode/services"
    "github.com/ownode/config"
    "github.com/ownode/models"
    jwt "github.com/dgrijalva/jwt-go"
    "time"
)

var (   
    Auth AuthController
    SigningKey string
    BackOfficeId string
    BackOfficeSecret string
) 

func init() {
    Auth = AuthController{ &Base }
    SigningKey = services.GetEnvOrDefault("OWNODE_KEY", "sample_key")   //TODO: remove sample key
    services.GetEnvOrDefault("OWNODE_KEY", "sample_key")
    BackOfficeId = services.GetEnvOrDefault("OWNODE_BACKOFFICE_ID", "backoffice")
    BackOfficeSecret = services.GetEnvOrDefault("OWNODE_BACKOFFICE_SECRET", "backofficesecret")
}

// create a jwt token
func createJWTToken(serviceId string, backOffice bool, expires_in int64) (string, error) {
    token := jwt.New(jwt.SigningMethodHS256)
    token.Claims["service_id"] = serviceId
    token.Claims["expires_in"] = expires_in
    token.Claims["back_office"] = backOffice
    tokenString, err := token.SignedString([]byte(SigningKey))
    return tokenString, err
}

type tokenResp struct {
    Token string `json:"access_token"`
    TokenType string `json:"token_type"`
    ExpiresIn int64 `json:"expires_in"`
    IsBackOffice bool `json:"is_back_office,omitempty"`
}

type AuthController struct {
    *BaseController
}

// create authentication token for client_credentials grant type
func (c *AuthController) GetToken(res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
    
    // get grant type
    grantType := req.FormValue("grant_type")
    
    // launch the appropriate function to produce the token
    switch grantType {
    case "client_credentials":
     c.GetClientCredentialToken(res, req, log, db)
     return
    }
}

// generate and return client_credentials token
func (c *AuthController) GetClientCredentialToken(res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
    
    // get base64 encoded credentials
    base64Credential := services.StringSplit(req.Header.Get("Authorization"), " ")[1]
    base64CredentialDecoded := services.DecodeB64(base64Credential)
    credentials := services.StringSplit(base64CredentialDecoded, ":")

    // check if requesting client is a back service id
    if credentials[0] == BackOfficeId && credentials[1] == BackOfficeSecret {
        
        exp := int64(0)
        token, err := createJWTToken("", true, exp)
        if err != nil {
            log.Error(err)
            services.Res(res).Error(500, "", "server error")
            return
        }

        // create and save new token
        newToken := models.Token {
            Token: token,
            Type: "bearer",
            ExpiresIn: time.Time{},
            CreatedAt: time.Now().UTC(),
            UpdatedAt: time.Now().UTC(),
        }

        // persist token
        err = models.CreateToken(db.GetPostgresHandle(), &newToken)
        if err != nil {
            c.log.Error(err.Error())
            services.Res(res).Error(500, "", "server error")
            return
        }

        services.Res(res).Json(newToken)
        return  
    }

    // find service by client id
    service, found, err := models.FindServiceByClientId(db.GetPostgresHandle(), credentials[0]); 
    if !found && err == nil {
        log.Error(err)
        services.Res(res).Error(404, "", "service not found")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // compare secret
    if credentials[1] != service.ClientSecret {
        log.Error(err)
        services.Res(res).Error(401, "", "service credentials are invalid. ensure client id and secret are valid")
        return
    } 
    
    // create access token
    exp := time.Now().Add(time.Hour * 1) 
    token, err := createJWTToken(service.ObjectID, false, exp.UTC().Unix())
    if err != nil {
        log.Error(err)
        services.Res(res).Error(500, "", "server error")
        return
    }

    // create and save new token
    newToken := models.Token {
        Service: service,
        Token: token,
        Type: "bearer",
        ExpiresIn: exp.UTC(),
        CreatedAt: time.Now().UTC(),
        UpdatedAt: time.Now().UTC(),
    }
    
    // persist token
    err = models.CreateToken(db.GetPostgresHandle(), &newToken)
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    respObj, _ := services.StructToJsonToMap(newToken)
    respObj["service"] = services.DeleteKeys(respObj["service"].(map[string]interface{}), "client_id", "client_secret")
    services.Res(res).Json(respObj)
}