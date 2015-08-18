package controllers

import (
    "net/http"
    "github.com/ownode/services"
    "github.com/ownode/config"
    "strings"
    "strconv"
    validator "github.com/asaskevich/govalidator"
    "github.com/ownode/models"
    "github.com/go-martini/martini"
    "gopkg.in/mgo.v2/bson"
    "time"
    "fmt"
    "sort"
)

var (   
    Object ObjectController
) 

// clear fields used in setting an object state to `open`
func clearOpen(object *models.Object) {
    object.Open = false
    object.OpenMethod = ""
    object.OpenTime = 0
    object.OpenPin = ""
}

type objectCreateBody struct {
    Type string             `json:"type"`
    WalletID string         `json:"wallet_id"`
    NumberOfObjects int     `json:"number_objects"`
    BalancePerObject float64   `json:"unit_per_object"` 
    Meta string             `json:"meta"` 
}

type objectMergeBody struct {
    Objects []string `json:"objects"`
    Meta string `json:"meta"`
}

type objectDivideBody struct {
    Object string `json:"object"`
    NumObjects int `json:"num_objects"`
    Meta string `json:"meta"`
    InheritMeta bool `json:"inherit_meta"`
}

type objectSubtractBody struct {
    Object string `json:"object"`
    AmountToSubtract float64 `json:"amount"`
    Meta string `json:"meta"`
    InheritMeta bool `json:"inherit_meta"`
}

type objectChargeBody struct {
    IDS []string `json:"ids"`
    DestinationWalletID string `json:"wallet_id"`
    Amount float64 `json:"amount"`
    Pins map[string]int `json:"pins"`
    Meta string `json:"meta"`
}

type objectOpenBody struct {
    OpenMethod string `json:"open_method"`
    Time int64 `json:"time"`
    Pin string `json:pin`
}

func init() {
    Object = ObjectController{ &Base }
}

type ObjectController struct {
    *BaseController
}

// create a new object
func NewObject(pin string, objType string, service models.Service, wallet models.Wallet, balance float64, meta string) models.Object {
    
    // make sure valueless objects have no balance
    if objType == models.ObjectValueless {
        balance = 0
    }

    return models.Object {
        ObjectID: bson.NewObjectId().Hex(),
        Pin: pin,
        Type: objType,
        Wallet: wallet,
        Balance: balance,
        Service: service,
        Meta: meta,
    }
}

// total balance of a slice of objects
func TotalBalance (objects []models.Object) float64 {
    sum := 0.0
    for _, obj := range objects {
        sum += obj.Balance
    }
    return sum
}

// create object controller
func (c *ObjectController) Create(res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
     
    // parse body
    var body objectCreateBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // TODO: get client id from access token
    clientID := "kl14zFDq4SHlmmmVNHgLtE0LqCo8BTjyShOH"

    // get db transaction object
    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // get service
    service, _, _ := models.FindServiceByClientId(dbTx, clientID)

    // ensure service is an issuer
    if !service.Identity.Issuer {
        dbTx.Rollback()
        services.Res(res).Error(401, "unauthorized_service", "service is not an issuer")
        return
    }

    // type is required
    if validator.IsNull(body.Type) {
        dbTx.Rollback()
        services.Res(res).Error(400, "missing_parameter", "Missing required field: type")
        return
    }

    // ensure type is valid
    if strings.ToLower(body.Type) != models.ObjectValue && strings.ToLower(body.Type) != models.ObjectValueless {
        dbTx.Rollback()
        services.Res(res).Error(400, "invalid_type", "type can only be obj_value or obj_valueless")
        return
    }

    // wallet id is required
    if validator.IsNull(body.WalletID) {
        dbTx.Rollback()
        services.Res(res).Error(400, "missing_parameter", "Missing required field: wallet_id")
        return
    }

    // ensure number of objects is greater than 0
    if body.NumberOfObjects < 1 {
        dbTx.Rollback()
        services.Res(res).Error(400, "invalid_number_objects", "number_objects must be atleast 1 but not more than 100")
        return
    }

    // ensure number of objects is not greater than 100(the limit)
    if body.NumberOfObjects > 100 {
        dbTx.Rollback()
        services.Res(res).Error(400, "invalid_number_objects", "number_objects must not be more than 100")
        return
    }

    // for object of value, validate  unit per object
    if strings.ToLower(body.Type) == models.ObjectValue {
       
        if body.BalancePerObject == 0 {
            dbTx.Rollback()
            services.Res(res).Error(400, "missing_parameter", "Missing required field: unit_per_object")
            return
        }

        if body.BalancePerObject < MinimumObjectUnit {
            dbTx.Rollback()
            services.Res(res).Error(400, "invalid_unit_per_object", "unit_per_object must be equal or greater than the minimum object unit which is 0.00000001")
            return
        } 

        soulBalanceRequired := float64(body.NumberOfObjects) * body.BalancePerObject
        if service.Identity.SoulBalance < soulBalanceRequired {
            dbTx.Rollback()
            services.Res(res).Error(400, "insufficient_soul_balance", fmt.Sprintf("not enough soul balance to create object(s). Requires %.2f soul balance", soulBalanceRequired))
            return
        }

        // update services soul balance
        service.Identity.SoulBalance = service.Identity.SoulBalance - soulBalanceRequired
    }

    // if meta is provided, ensure it is not greater than the limit size
    if !c.validate.IsEmpty(body.Meta) && len([]byte(body.Meta)) > MaxMetaSize {
        dbTx.Rollback()
        services.Res(res).Error(400, "invalid_meta_size", fmt.Sprintf("Meta contains too much data. Max size is %d bytes", MaxMetaSize))
        return
    }

    // ensure wallet exists
    wallet, found, err := models.FindWalletByObjectID(dbTx, body.WalletID)
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    } else if !found {
        dbTx.Rollback()
        services.Res(res).Error(404, "invalid_wallet", "wallet_id is unknown")
        return
    }

    // ensure service owns the wallet
    if service.Identity.ObjectID != wallet.Identity.ObjectID {
        dbTx.Rollback()
        services.Res(res).Error(401, "invalid_wallet", "wallet is not owned by this service. Use a wallet created by this service")
        return   
    }

    // create objects
    allNewObjects := []models.Object{}
    for i := 0; i < body.NumberOfObjects; i++ {

        // generate a pin
        countryCallCode := config.CurrencyCallCodes[strings.ToUpper(service.Identity.BaseCurrency)]
        newPin, err := services.NewObjectPin(strconv.Itoa(countryCallCode))
        if err != nil {
            dbTx.Rollback()
            c.log.Error(err.Error())
            services.Res(res).Error(500, "", "server error")
            return
        }

        newObj := NewObject(newPin, body.Type, service, wallet, body.BalancePerObject, body.Meta)
        err = models.CreateObject(dbTx, &newObj)
        if err != nil {
            dbTx.Rollback()
            c.log.Error(err.Error())
            services.Res(res).Error(500, "", "server error")
            return
        }
        allNewObjects = append(allNewObjects, newObj)
    }

    // update identity's soul balance
    dbTx.Save(service.Identity).Commit()
    services.Res(res).Json(allNewObjects)
}

// get an object by its id or pin
func (c *ObjectController) Get(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
    
    dbObj := db.GetPostgresHandle()
    object, found, err := models.FindObjectByObjectIDOrPin(dbObj, params["id"])
    if !found {
        services.Res(res).Error(404, "not_found", "object was not found")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // construct response. remove private fields
    respObj, _ := services.StructToJsonToMap(object)
    delete(respObj["wallet"].(map[string]interface{})["identity"].(map[string]interface{}), "soul_balance")
    delete(respObj["wallet"].(map[string]interface{})["identity"].(map[string]interface{}), "email")
    delete(respObj["service"].(map[string]interface{})["identity"].(map[string]interface{}), "soul_balance")
    delete(respObj["service"].(map[string]interface{})["identity"].(map[string]interface{}), "email")

    services.Res(res).Json(respObj)
}

// merge two or more objects.
// Only a max of 100 identitcal objects can be merged.
// All objects to be merged must exists.
// Only similar objects can be merged.
// Meta is not retained. Optional "meta" parameter can be 
// provided as new meta for the resulting object
// TODO: Needs wallet authorization with scode "obj_merge"
func (c *ObjectController) Merge(res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
    
    // authorizing wallet id
    // TODO: get this from the access token
    authWalletID := "55c679145fe09c74ed000001"

    // parse body
    var body objectMergeBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // objects field is required
    if body.Objects == nil {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: objects")
        return
    }

    // objects field must contain at least two objects
    if len(body.Objects) < 2 {
        services.Res(res).Error(400, "invalid_parameter", "objects: minimum of two objects required")
        return
    }

    // objects field must not contain more than 100 objects
    if len(body.Objects) > 100 {
        services.Res(res).Error(400, "invalid_parameter", "objects: cannot merge more than 100 objects in a request")
        return
    }

    // ensure objects contain no duplicates
    if services.StringSliceHasDuplicates(body.Objects) {
        services.Res(res).Error(400, "invalid_parameter", "objects: must not contain duplicate objects")
        return
    }

    // if meta is provided, ensure it is not greater than the limit size
    if !c.validate.IsEmpty(body.Meta) && len([]byte(body.Meta)) > MaxMetaSize {
        services.Res(res).Error(400, "invalid_meta_size", fmt.Sprintf("Meta contains too much data. Max size is %d bytes", MaxMetaSize))
        return
    }

    // get db transaction object
    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // find all objects
    objectsFound, err := models.FindAllObjectsByObjectID(dbTx, body.Objects)
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure all objects where found
    if len(objectsFound) != len(body.Objects) {
        dbTx.Rollback()
        services.Res(res).Error(400, "unknown_merge_objects", "one or more objects does not exists")
        return
    }

    totalBalance := 0.0
    firstObj := objectsFound[0]
    checkObjName := firstObj.Service.Identity.ObjectName

    for _, object := range objectsFound {
        
        // ensure all objects are valuable and also ensure object's
        if object.Type == models.ObjectValueless {
            dbTx.Rollback()
            services.Res(res).Error(400, "invalid_parameter", "objects: only valuable objects (object_value) can be merged")
            return
        }
        
        // wallet id match the authorizing wallet id 
        if object.Wallet.ObjectID != authWalletID {
            dbTx.Rollback()
            services.Res(res).Error(401, "unauthorized", "objects: one or more objects belongs to a different wallet")
            return
        }

        // ensure all objects are similar by their name / same issuer.
        // this also ensures all objects have the same base currency
        if checkObjName != object.Service.Identity.ObjectName {
            dbTx.Rollback()
            services.Res(res).Error(400, "invalid_parameter", "objects: only similar (by name) objects can be merged")
            return
        }

        // updated total balance
        totalBalance += object.Balance

        // delete object
        dbTx.Delete(&object)
    }

    // create a new object
    // generate a pin
    countryCallCode := config.CurrencyCallCodes[strings.ToUpper(firstObj.Service.Identity.BaseCurrency)]
    newPin, err := services.NewObjectPin(strconv.Itoa(countryCallCode))
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    newObj := NewObject(newPin, models.ObjectValue, firstObj.Service, firstObj.Wallet, totalBalance, body.Meta)
    err = models.CreateObject(dbTx, &newObj)
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }
    
    dbTx.Commit()
    services.Res(res).Json(newObj)
}

// divide an object into two or more equal parts.
// maxinum of 100 equal parts is allowed.
// object to be splitted must belong to authorizing wallet
func (c *ObjectController) Divide(res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
    
    // authorizing wallet id
    // todo: get from access token
    authWalletID := "55c679145fe09c74ed000001"

    // parse body
    var body objectDivideBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // object is required
    if c.validate.IsEmpty(body.Object) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: object")
        return
    }    

    // number of objects must be greater than 1
    if body.NumObjects < 2 {
        services.Res(res).Error(400, "invalid_parameter", "num_objects: must be greater than 1")
        return
    }

    // number of objects must not be greater than 100
    if body.NumObjects > 100 {
        services.Res(res).Error(400, "invalid_parameter", "num_objects: must be greater than 1")
        return
    }

    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // get the object
    object, found, err := models.FindObjectByObjectID(dbTx, body.Object)
    if !found {
        dbTx.Rollback()
        services.Res(res).Error(404, "not_found", "object was not found")
        return
    } else if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure object is a valuable type
    if object.Type == models.ObjectValueless {
        dbTx.Rollback()
        services.Res(res).Error(400, "invalid_parameter", "object: object must be a valuabe type (obj_value) ")
        return
    }

    // ensure object has enough balance (minimum of 0.000001)
    if object.Balance < 0.000001 {
        dbTx.Rollback()
        services.Res(res).Error(400, "invalid_parameter", "object: object must have a minimum balance of 0.000001")
        return
    }

    // ensure object belongs to authorizing wallet
    if object.Wallet.ObjectID != authWalletID {
        dbTx.Rollback()
        services.Res(res).Error(401, "unauthorized", "object does not belong to authorizing wallet")
        return
    }

    // if meta is provided, ensure it is not greater than the limit size
    if !body.InheritMeta && !c.validate.IsEmpty(body.Meta) && len([]byte(body.Meta)) > MaxMetaSize {
        services.Res(res).Error(400, "invalid_meta_size", fmt.Sprintf("Meta contains too much data. Max size is %d bytes", MaxMetaSize))
        return
    } else {
        if body.InheritMeta {
            body.Meta = object.Meta
        }
    }

    // calculate new balance per object
    newBalance := object.Balance / float64(body.NumObjects)

    // delete object
    dbTx.Delete(&object)

    // create new objects
    newObjects := []models.Object{}
    for i := 0; i < body.NumObjects; i++ {

        // generate a pin
        countryCallCode := config.CurrencyCallCodes[strings.ToUpper(object.Service.Identity.BaseCurrency)]
        newPin, err := services.NewObjectPin(strconv.Itoa(countryCallCode))
        if err != nil {
            dbTx.Rollback()
            c.log.Error(err.Error())
            services.Res(res).Error(500, "", "server error")
            return
        }

        newObj := NewObject(newPin, models.ObjectValue, object.Service, object.Wallet, newBalance, body.Meta)
        err = models.CreateObject(dbTx, &newObj)
        if err != nil {
            dbTx.Rollback()
            c.log.Error(err.Error())
            services.Res(res).Error(500, "", "server error")
            return
        }

        newObjects = append(newObjects, newObj)
    }
    
    dbTx.Commit()
    services.Res(res).Json(newObjects)
}

// create a new object by subtracting from a source object
func (c *ObjectController) Subtract(res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
    
    // TODO: get from access token
    // authorizing wallet id
    authWalletID := "55c679145fe09c74ed000001"

    // parse body
    var body objectSubtractBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // object is required
    if c.validate.IsEmpty(body.Object) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: object")
        return
    } 

    // amount to subtract is required
    if body.AmountToSubtract < MinimumObjectUnit {
        services.Res(res).Error(400, "invalid_parameter", "amount: amount must be equal or greater than the minimum object unit which is 0.00000001")
        return
    }

    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // get the object
    object, found, err := models.FindObjectByObjectID(dbTx, body.Object)
    if !found {
        dbTx.Rollback()
        services.Res(res).Error(404, "not_found", "object was not found")
        return
    } else if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure object is a valuable type
    if object.Type == models.ObjectValueless {
        dbTx.Rollback()
        services.Res(res).Error(400, "invalid_parameter", "object: object must be a valuabe type (obj_value) ")
        return
    }

    // ensure object belongs to authorizing wallet
    if object.Wallet.ObjectID != authWalletID {
        dbTx.Rollback()
        services.Res(res).Error(401, "unauthorized", "objects: object does not belong to authorizing wallet")
        return
    }

    // ensure object's balance is sufficient 
    if object.Balance < body.AmountToSubtract {
        dbTx.Rollback()
        services.Res(res).Error(400, "invalid_parameter", "amount: object's balance is insufficient")
        return
    }

    // if meta is provided, ensure it is not greater than the limit size
    if !body.InheritMeta && !c.validate.IsEmpty(body.Meta) && len([]byte(body.Meta)) > MaxMetaSize {
        services.Res(res).Error(400, "invalid_meta_size", fmt.Sprintf("Meta contains too much data. Max size is %d bytes", MaxMetaSize))
        return
    } else {
        if body.InheritMeta {
            body.Meta = object.Meta
       }
   }

    // subtract and update object's balance
    object.Balance = object.Balance - body.AmountToSubtract
    dbTx.Save(&object)

    // create new object
    // generate a pin
    countryCallCode := config.CurrencyCallCodes[strings.ToUpper(object.Service.Identity.BaseCurrency)]
    newPin, err := services.NewObjectPin(strconv.Itoa(countryCallCode))
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    newObj := NewObject(newPin, models.ObjectValue, object.Service, object.Wallet, body.AmountToSubtract, body.Meta)
    err = models.CreateObject(dbTx, &newObj)
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    dbTx.Commit()
    services.Res(res).Json(newObj)
}

// open an object for charge/consumption. An object opened in this method
// will be consumable without restriction
func (c *ObjectController) Open(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
    
    // TODO: get from access token
    // authorizing wallet id
    authWalletID := "55c679145fe09c74ed000001"

    // parse body
    var body objectOpenBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // ensure open method is provided
    if c.validate.IsEmpty(body.OpenMethod) {
        services.Res(res).Error(400, "invalid_parameter", "open_method: open method is required")
        return
    }

    // ensure a known open method is provided
    if body.OpenMethod != "open" && body.OpenMethod != "open_timed" && body.OpenMethod != "open_pin" {
        services.Res(res).Error(400, "invalid_parameter", "unknown open type method")
        return
    }

    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // get the object
    object, found, err := models.FindObjectByObjectID(dbTx, params["id"])
    if !found {
        dbTx.Rollback()
        services.Res(res).Error(404, "not_found", "object was not found")
        return
    } else if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure object belongs to authorizing wallet
    if object.Wallet.ObjectID != authWalletID {
        dbTx.Rollback()
        services.Res(res).Error(401, "unauthorized", "objects: object does not belong to authorizing wallet")
        return
    }

    // set object's open property to true and open_method to `open`
    clearOpen(&object)
    object.Open = true
    object.OpenMethod = models.ObjectOpenDefault

    // for open_timed,
    // set 'open_time' field to indicate object open window
    if body.OpenMethod == "open_timed" {
         
        // ensure time field is provided
        if body.Time == 0 {
            dbTx.Rollback()
            services.Res(res).Error(400, "invalid_parameter", "time: open window time is required. use unix time")
            return 
        }

        // time must be in the future
        now := time.Now().UTC()
        if !now.Before(services.UnixToTime(body.Time).UTC()) {
            dbTx.Rollback()
            services.Res(res).Error(400, "invalid_parameter", "time: use a unix time pointing to a period in the future")
            return
        }

        object.OpenMethod = models.ObjectOpenTimed
        object.OpenTime = body.Time
    }

    // for open_pin
    // open pin sets a pin for used by charge API 
    if body.OpenMethod == "open_pin" {

        // ensure pin is provided
        if c.validate.IsEmpty(body.Pin) {
            dbTx.Rollback()
            services.Res(res).Error(400, "invalid_parameter", "pin: pin is required")
            return 
        }

        // pin must be numeric
        if !validator.IsNumeric(body.Pin) {
            dbTx.Rollback()
            services.Res(res).Error(400, "invalid_parameter", "pin: pin must contain only numeric characters. e.g 4345")
            return 
        }

        // pin length must be between 4 - 12 characters
        if len(body.Pin) < 4 || len(body.Pin) > 12 {
            dbTx.Rollback()
            services.Res(res).Error(400, "invalid_parameter", "pin: pin must have a minimum character length of 4 and maximum of 12")
            return 
        }

        // hash pin using bcrypt
        pinHash, err := services.Bcrypt(body.Pin, 10)
        if err != nil {
            c.log.Error("unable to hash password. reason: " + err.Error())
            services.Res(res).Error(500, "", "server error")
            return
        }

        object.OpenMethod = models.ObjectOpenPin
        object.OpenPin = pinHash
    }

    dbTx.Save(&object).Commit()
    services.Res(res).Json(object)
}

// lock sets set the 'open' property of an object to false. Also resets all fields used by  
// all open methods
func (c *ObjectController) Lock(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
    
    // TODO: get from access token
    // authorizing wallet id
    authWalletID := "55c679145fe09c74ed000001"

    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // get the object
    object, found, err := models.FindObjectByObjectID(dbTx, params["id"])
    if !found {
        dbTx.Rollback()
        services.Res(res).Error(404, "not_found", "object was not found")
        return
    } else if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure object belongs to authorizing wallet
    if object.Wallet.ObjectID != authWalletID {
        dbTx.Rollback()
        services.Res(res).Error(401, "unauthorized", "objects: object does not belong to authorizing wallet")
        return
    }

    // clear open related fields of object
    clearOpen(&object)

    // save update and commit
    dbTx.Save(&object).Commit()
    services.Res(res).Json(object)
}

// charge an object. Deduct from an object, create one or more objects and 
// associated to one or more wallets
func (c *ObjectController) Charge(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, log *config.CustomLog, db *services.DB) {
  
    // parse body
    var body objectChargeBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // TODO: get client id from access token
    clientID := "kl14zFDq4SHlmmmVNHgLtE0LqCo8BTjyShOH"

    // get db transaction object
    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // get service
    service, _, _ := models.FindServiceByClientId(dbTx, clientID)

    // ensure object ids is not empty
    if len(body.IDS) == 0 {
        services.Res(res).ErrParam("ids").Error(400, "invalid_parameter", "provide one or more object ids to charge")
        return
    }

    // ensure object ids length is less than 100
    if len(body.IDS) > 100 {
        services.Res(res).ErrParam("ids").Error(400, "invalid_parameter", "only a maximum of 100 objects can be charge at a time")
        return
    }

    // ensure destination wallet is provided
    if c.validate.IsEmpty(body.DestinationWalletID) {
        services.Res(res).ErrParam("wallet_id").Error(400, "invalid_parameter", "destination wallet id is reqired")
        return
    }

    // ensure amount is provided
    if body.Amount < MinimumObjectUnit {
        services.Res(res).ErrParam("amount").Error(400, "invalid_parameter", fmt.Sprintf("amount is below the minimum charge limit. Mininum charge limit is %.8f", MinimumObjectUnit))
        return
    }

    // if meta is provided, ensure it is not greater than the limit size
    if !c.validate.IsEmpty(body.Meta) && len([]byte(body.Meta)) > MaxMetaSize {
        services.Res(res).ErrParam("meta").Error(400, "invalid_parameter", fmt.Sprintf("Meta contains too much data. Max size is %d bytes", MaxMetaSize))
        return
    }

    // ensure destination wallet exists
    wallet, found, err := models.FindWalletByObjectID(dbTx, body.DestinationWalletID)
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "api_error", "api_error")
        return
    } else if !found {
        dbTx.Rollback()
        services.Res(res).ErrParam("wallet_id").Error(404, "not_found", "wallet_id not found")
        return
    }

    // find all objects
    objectsFound, err := models.FindAllObjectsByObjectID(dbTx, body.IDS)
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "api_error", "api_error")
        return
    }

    // ensure all objects exists
    if len(objectsFound) != len(body.IDS) {
        dbTx.Rollback()
        services.Res(res).ErrParam("ids").Error(404, "object_error", "one or more objects do not exist")
        return
    }
    
    // sort object by balance in descending order
    sort.Sort(services.ByObjectBalance(objectsFound))

    // objects to charge
    objectsToCharge := []models.Object{}

    // validate each object
    // check open status (timed and pin)
    // collect the required objects to sufficiently 
    // complete a charge from the list of found objects
    for _, object := range objectsFound {

        // as long as the total balance of objects to be charged is not above charge amount
        // keep setting aside objects to charge from.
        // once we have the required objects to cover charge amount, stop processing other objects
        if TotalBalance(objectsToCharge) < body.Amount {
            objectsToCharge = append(objectsToCharge, object)
        } else {
            break
        }

        // ensure service is the issuer of object
        if object.Service.ObjectID != service.ObjectID {
            dbTx.Rollback()
            services.Res(res).ErrParam("ids").Error(402, "object_error", fmt.Sprintf("%s: service cannot charge an object not issued by it", object.ObjectID))
            return
        }

        // ensure object is open
        if !object.Open {
            dbTx.Rollback()
            services.Res(res).ErrParam("ids").Error(402, "object_error", fmt.Sprintf("%s: object is not opened and cannot be charged", object.ObjectID))
            return

        } else {

            // for object with open_timed open method, ensure time is not passed
            if object.OpenMethod == models.ObjectOpenTimed {
                objectOpenTime := services.UnixToTime(object.OpenTime).UTC()
                now := time.Now().UTC()
                if now.After(objectOpenTime) {
                    dbTx.Rollback()
                    services.Res(res).ErrParam("ids").Error(402, "object_error", fmt.Sprintf("%s: object open time period has expired", object.ObjectID))
                    return
                }
            }

            // for object with open_pin open method, ensure pin is provided and 
            // it matches. Pin should be found in the optional pin object of the request body
            if object.OpenMethod == models.ObjectOpenPin {
                if pin, found := body.Pins[object.ObjectID]; found {
                    
                    // ensure pin provided matches objects pin
                    if !services.BcryptCompare(object.OpenPin, strconv.Itoa(pin)) {
                        dbTx.Rollback()
                        services.Res(res).ErrParam("ids").Error(402, "object_error", fmt.Sprintf("%s: pin provided to open object is invalid", object.ObjectID))
                        return
                    }

                } else {
                    dbTx.Rollback()
                    services.Res(res).ErrParam("ids").Error(402, "object_error", fmt.Sprintf("%s: object pin not found in pin parameter of request body", object.ObjectID))
                    return
                }
            }
        }
    }

    totalObjectsBalance := TotalBalance(objectsToCharge)

    // ensure total balance of objects to charge is sufficient for charge amount
    if totalObjectsBalance < body.Amount {
        dbTx.Rollback()
        services.Res(res).ErrParam("amount").Error(402, "invalid_parameter", fmt.Sprintf("object%s total balance not sufficient to cover charge amount", services.SIfNotZero(len(body.IDS))))
        return
    }
    
    lastObj := objectsToCharge[len(objectsToCharge) - 1]

    // if there is excess, the last object is always the supplement object.
    // deduct from last object's balance, update object and remove it from the objectsToCharge list
    if totalObjectsBalance > body.Amount {
        lastObj.Balance = totalObjectsBalance - body.Amount
        objectsToCharge = objectsToCharge[0:len(objectsToCharge)-1]
        dbTx.Save(&lastObj)
    }

    // delete the objects to charge
    for _, object := range objectsToCharge {
        dbTx.Delete(&object)
    }

    // create new object. set balance to charge balance
    // generate a pin
    countryCallCode := config.CurrencyCallCodes[strings.ToUpper(service.Identity.BaseCurrency)]
    newPin, err := services.NewObjectPin(strconv.Itoa(countryCallCode))
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "api_error", "server error")
        return
    }

    newObj := NewObject(newPin, models.ObjectValue, service, wallet, body.Amount, body.Meta)
    err = models.CreateObject(dbTx, &newObj)
    if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "api_error", "server error")
        return
    }

    dbTx.Save(&newObj).Commit()
    services.Res(res).Json(newObj)
}


