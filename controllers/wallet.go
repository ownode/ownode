package controllers

import (
    "net/http"
    "github.com/ownode/models"
	"gopkg.in/mgo.v2/bson"
    "github.com/ownode/services"
    "github.com/go-martini/martini"
    "time"
    "strconv"
    validator "github.com/asaskevich/govalidator"
)

var Wallet WalletController

type walletCreateBody struct {
	IdentityId string  `json:"identity_id"`
	Handle string 
    Password string
}

func init() {
    Wallet = WalletController{ &Base }
}

type WalletController struct {
    *BaseController
}

// create a wallet
func (c *WalletController) Create(res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {

    // parse body
    var body walletCreateBody
    if err := c.ParseJsonBody(req, &body); err != nil {
        services.Res(res).Error(400, "invalid_body", "request body is invalid or malformed. Expects valid json body")
        return 
    }

    // identity id is required
    if validator.IsNull(body.IdentityId) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: identity_id")
        return
    }

    // handle is required
    if validator.IsNull(body.Handle) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: handle")
        return
    }

    // password is required
    if validator.IsNull(body.Password) {
        services.Res(res).Error(400, "missing_parameter", "Missing required field: password")
        return
    }

    // identity id must exist
    identity, found, err := models.FindIdentityByObjectID(db.GetPostgresHandle(), body.IdentityId)
    if !found {
        services.Res(res).Error(404, "invalid_identity", "identity_id is unknown")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // handle must be unique across wallets
    _, found, err = models.FindWalletByHandle(db.GetPostgresHandle(), body.Handle)
    if found {
        services.Res(res).Error(400, "handle_registered", "handle has been registered to another wallet")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // password length must be 6 characters
    if len(body.Password) < 6 {
        services.Res(res).Error(400, "invalid_password", "password is too short. minimum length is 6 characters")
        return
    }

    // securely hash password
    hashedPassword, err := services.Bcrypt(body.Password, 10)
    if err != nil {
        c.log.Error("unable to hash password. reason: " + err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    } else {
        body.Password = hashedPassword
    }

    // create wallet object
    newWallet := models.Wallet {
        ObjectID: bson.NewObjectId().Hex(),
        Identity: identity,
        Handle: body.Handle,
        Password: body.Password,
    }

    // create wallet
    err = models.CreateWallet(db.GetPostgresHandle(), &newWallet)
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    respObj, _ := services.StructToJsonToMap(newWallet)
    services.Res(res).Json(respObj)
}

// get a wallet
func (c *WalletController) Get(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {

    wallet, found, err := models.FindWalletByObjectID(db.GetPostgresHandle(), params["id"])
    if !found {
        services.Res(res).Error(404, "not_found", "service was not found")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    respObj, _ := services.StructToJsonToMap(wallet)
    services.Res(res).Json(respObj)
}

// list object
// supports
// - pagination using 'page' query. Use per_page to set the number of results per page. max is 100
// - filters: filter_type, filter_service, filter_open, filter_open_method, filter_gte_date_created
//   filter_lte_date_created
// - sorting: sort_balance, sort_date_created
func (c *WalletController) List(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
    
    // TODO: get from access token
    // authorizing wallet id
    authWalletID := "55c679145fe09c74ed000001"

    dbCon := db.GetPostgresHandle()

    // get wallet
    wallet, found, err := models.FindWalletByObjectID(dbCon, params["id"])
    if !found {
        services.Res(res).Error(404, "not_found", "wallet not found")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure wallet matches authorizing wallet
    if wallet.ObjectID != authWalletID {
        services.Res(res).Error(401, "unauthorized", "client does not have permission to access wallet")
        return
    }

    query := req.URL.Query()
    qPage := query.Get("page")
    if c.validate.IsEmpty(qPage) {
        qPage = "0"
    } else if !validator.IsNumeric(qPage) {
        services.Res(res).Error(400, "invalid_parameter", "page query value must be numeric")
        return 
    } 

    q := make(map[string]interface{})
    q["wallet_id"] = wallet.ID
    order := "id asc"
    limitPerPage := int64(2)
    offset := int64(0)
    currentPage, err := strconv.ParseInt(qPage, 0, 64)
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // set limit per page if provided in query
    qPerPage := query.Get("per_page")
    if !c.validate.IsEmpty(qPerPage) {
        if validator.IsNumeric(qPerPage) {
            qPerPage, _ := strconv.ParseInt(qPerPage, 0, 64)
            if qPerPage > 100 {
                qPerPage = 100
            } else if qPerPage <= 0 {
                qPerPage = limitPerPage
            }
            limitPerPage = qPerPage
        }
    }

    // set current page default and calculate offset
    if currentPage <= 1 {
        currentPage = 1
        offset = 0;
    } else {
        offset = (int64(limitPerPage) * currentPage) - int64(limitPerPage)
    }


    // apply type filter if included in query
    filterType := query.Get("filter_type")
    if !c.validate.IsEmpty(filterType) && services.StringInStringSlice([]string{"obj_value","obj_valueless"}, filterType) {
        q["type"] = filterType
    }

    // apply service filter if included in query
    filterService := query.Get("filter_service")
    if !c.validate.IsEmpty(filterService) {

        // find service
        service, found, err := models.FindServiceByObjectID(db.GetPostgresHandle(), filterService)
        if err != nil {
            c.log.Error(err.Error())
            services.Res(res).Error(500, "", "server error")
            return
        }

        if found {
            q["service_id"] = service.ID
        }
    }

    // apply open filter if included in query
    filterOpen := query.Get("filter_open")
    if !c.validate.IsEmpty(filterOpen) && services.StringInStringSlice([]string{"true","false"}, filterOpen) {
        q["open"] = filterOpen
    }

    // apply open_method filter if included in query
    filterOpenMethod := query.Get("filter_open_method")
    if !c.validate.IsEmpty(filterOpenMethod) && services.StringInStringSlice([]string{"open","open_timed","open_pin"}, filterOpenMethod) {
        q["open_method"] = filterOpenMethod
    }

    // apply filter_gte_date_created filter if included in query
    filterGTEDateCreated := query.Get("filter_gte_date_created")
    if !c.validate.IsEmpty(filterGTEDateCreated) {
        if validator.IsNumeric(filterGTEDateCreated) {
            ts, _ := strconv.ParseInt(filterGTEDateCreated, 0, 64)
            dbCon = dbCon.Where("created_at >= ?", services.UnixToTime(ts).UTC().Format(time.RFC3339Nano))
        }
    }

    // apply filter_lte_date_created filter if included in query
    filterLTEDateCreated := query.Get("filter_lte_date_created")
    if !c.validate.IsEmpty(filterLTEDateCreated) {
        if validator.IsNumeric(filterLTEDateCreated) {
            ts, _ := strconv.ParseInt(filterLTEDateCreated, 0, 64)
            dbCon = dbCon.Where("created_at <= ?", services.UnixToTime(ts).UTC().Format(time.RFC3339Nano))
        }
    }

    // the below connection is used for sorting/ordering 
    var dbConSort = dbCon

    // apply sort_balance sort if included
    sortBalance := query.Get("sort_balance")
    if !c.validate.IsEmpty(sortBalance) {
        orderVal := "asc"
        if (sortBalance == "-1") {
            orderVal = "desc"
        } 
        dbConSort = dbCon.Order("objects.balance " + orderVal)
    }

    // apply ort_date_created sort if included
    sortDateCreated := query.Get("sort_date_created")
    if !c.validate.IsEmpty(sortDateCreated) {
        orderVal := "asc"
        if (sortDateCreated == "-1") {
            orderVal = "desc"
        } 
        dbConSort = dbConSort.Order("objects.created_at " + orderVal)
    }

    // find objects associated with wallet
    objects := []models.Object{}
    var objectsCount int64
    
    // count number of objects. I didnt use dbConSort as count will throw an error
    dbCon.Model(models.Object{}).Where(q).Count(&objectsCount)
    
    // set the original db connection to the sort connection
    dbCon = dbConSort       

    // calculate number of pages
    numPages := services.Round(float64(objectsCount) / float64(limitPerPage))

    // fetch the objects
    dbCon.Where(q).Preload("Service.Identity").Preload("Wallet.Identity").Limit(limitPerPage).Offset(offset).Order(order).Find(&objects)
    
    // prepare response
    respObj, _ := services.StructToJsonToSlice(objects)
    if len(respObj) == 0 {
        respObj = []map[string]interface{}{}
    }

    services.Res(res).Json(map[string]interface{}{
        "results": respObj,
        "_metadata": map[string]interface{}{
            "total_count": objectsCount,
            "per_page": limitPerPage,
            "page_count": numPages,
            "page": currentPage,
        }, 
    })
}

// get counts and other numerical information about the state of a wallet
// e.g object balance and count etc
func (c *WalletController) Numbers(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {

    // TODO: get from access token
    // authorizing wallet id
    authWalletID := "55c679145fe09c74ed000001"

    dbCon := db.GetPostgresHandle()
    resp := map[string]interface{}{}
    query := req.URL.Query()
    count := int64(0)

    // predefine default query values
    query.Set("object_count", "true")
    query.Set("distinct_object_count", "true")
    query.Set("valuable_object_count", "true")
    query.Set("valueless_object_count", "true")
    query.Set("valueable_object_balance", "true")

    // get wallet
    wallet, found, err := models.FindWalletByObjectID(dbCon, params["id"])
    if !found {
        services.Res(res).Error(404, "not_found", "wallet not found")
        return
    } else if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure wallet matches authorizing wallet
    if wallet.ObjectID != authWalletID {
        services.Res(res).Error(401, "unauthorized", "client does not have permission to access wallet")
        return
    }

    // object_count
    objectCountField := query.Get("object_count")
    if !c.validate.IsEmpty(objectCountField) && services.StringInStringSlice([]string{"true","false"}, objectCountField) {
        if objectCountField == "true" {
            q := map[string]interface{}{
                "wallet_id": wallet.ID,
            }
            dbCon.Model(models.Object{}).Where(q).Count(&count)
            resp["object_count"] = count
            count = 0
        }
    }

    // distinct_object_count
    distinctObjectCountField := query.Get("distinct_object_count")
    if !c.validate.IsEmpty(distinctObjectCountField) && services.StringInStringSlice([]string{"true","false"}, distinctObjectCountField) {
        if distinctObjectCountField == "true" {
            row, err := dbCon.Raw("SELECT COUNT(*) FROM (SELECT DISTINCT service_id FROM objects WHERE wallet_id = ?) AS distinct_object_count;", wallet.ID).Rows()
            if err != nil {
                c.log.Error(err.Error())
                services.Res(res).Error(500, "", "server error")
                return
            }
            if row.Next() {
                row.Scan(&count)
            }
            resp["distinct_object_count"] = count
            count = 0
        }
    }

    // valuable_object_count
    valuableObjectCountField := query.Get("valuable_object_count")
    if !c.validate.IsEmpty(distinctObjectCountField) && services.StringInStringSlice([]string{"true","false"}, valuableObjectCountField) {
        if valuableObjectCountField == "true" {
            q := map[string]interface{}{
                "wallet_id": wallet.ID,
                "type": models.ObjectValue,
            }
            dbCon.Model(models.Object{}).Where(q).Count(&count)
            resp["valuable_object_count"] = count
            count = 0
        }
    }

    // valueless_object_count
    valuelessObjectCountField := query.Get("valueless_object_count")
    if !c.validate.IsEmpty(valuelessObjectCountField) && services.StringInStringSlice([]string{"true","false"}, valuelessObjectCountField) {
        if valuelessObjectCountField == "true" {
            q := map[string]interface{}{
                "wallet_id": wallet.ID,
                "type": models.ObjectValueless,
            }
            dbCon.Model(models.Object{}).Where(q).Count(&count)
            resp["valueless_object_count"] = count
            count = 0
        }
    }

    // valueable_object_balance
    valuableObjectBalanceField := query.Get("valueable_object_balance")
    if !c.validate.IsEmpty(distinctObjectCountField) && services.StringInStringSlice([]string{"true","false"}, valuableObjectBalanceField) {
        if valuableObjectBalanceField == "true" {
            row, err := dbCon.Raw("SELECT SUM(balance) AS total_balance FROM objects WHERE wallet_id = ? AND type = ?;", wallet.ID, models.ObjectValue).Rows()
            if err != nil {
                c.log.Error(err.Error())
                services.Res(res).Error(500, "", "server error")
                return
            }
            if row.Next() {
                row.Scan(&count)
            }
            resp["valueable_object_balance"] = count
            count = 0
        }
    }

    // opened_object_count
    openedObjectCountField := query.Get("opened_object_count")
    if !c.validate.IsEmpty(openedObjectCountField) && services.StringInStringSlice([]string{"true","false"}, openedObjectCountField) {
        if openedObjectCountField == "true" {
            q := map[string]interface{}{
                "wallet_id": wallet.ID,
                "open": true,
            }
            dbCon.Model(models.Object{}).Where(q).Count(&count)
            resp["opened_object_count"] = count
            count = 0
        }
    }

    // locked_object_count
    lockedObjectCountField := query.Get("locked_object_count")
    if !c.validate.IsEmpty(lockedObjectCountField) && services.StringInStringSlice([]string{"true","false"}, lockedObjectCountField) {
        if lockedObjectCountField == "true" {
            q := map[string]interface{}{
                "wallet_id": wallet.ID,
                "open": false,
            }
            dbCon.Model(models.Object{}).Where(q).Count(&count)
            resp["locked_object_count"] = count
            count = 0
        }
    }

    // opened_timed_object_count
    openedTimedObjectCountField := query.Get("opened_timed_object_count")
    if !c.validate.IsEmpty(openedObjectCountField) && services.StringInStringSlice([]string{"true","false"}, openedTimedObjectCountField) {
        if openedTimedObjectCountField == "true" {
            q := map[string]interface{}{
                "wallet_id": wallet.ID,
                "open": true,
                "open_method": models.ObjectOpenTimed,
            }
            dbCon.Model(models.Object{}).Where(q).Count(&count)
            resp["opened_timed_object_count"] = count
            count = 0
        }
    }

    // opened_pin_object_count
    openedPinObjectCountField := query.Get("opened_pin_object_count")
    if !c.validate.IsEmpty(openedPinObjectCountField) && services.StringInStringSlice([]string{"true","false"}, openedPinObjectCountField) {
        if openedPinObjectCountField == "true" {
            q := map[string]interface{}{
                "wallet_id": wallet.ID,
                "open": true,
                "open_method": models.ObjectOpenPin,
            }
            dbCon.Model(models.Object{}).Where(q).Count(&count)
            resp["opened_pin_object_count"] = count
            count = 0
        }
    }

    services.Res(res).Json(resp)
}

// lock a wallet. A lock on a wallet prevents charges on opened objects
func (c *WalletController) Lock(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
    
    // TODO: get from access token
    // authorizing wallet id
    authWalletID := "55c679145fe09c74ed000001"

    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // get wallet
    wallet, found, err := models.FindWalletByObjectID(dbTx, params["id"])
    if !found {
        dbTx.Rollback()
        services.Res(res).Error(404, "not_found", "wallet not found")
        return
    } else if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure wallet matches authorizing wallet
    if wallet.ObjectID != authWalletID {
        dbTx.Rollback()
        services.Res(res).Error(401, "unauthorized", "client does not have permission to access wallet")
        return
    }

    // update lock state
    wallet.Lock = true

    // save and commit
    dbTx.Save(&wallet).Commit()
    services.Res(res).Json(wallet)
}

// open/unlock a wallet
func (c *WalletController) Open(params martini.Params, res http.ResponseWriter, req services.AuxRequestContext, db *services.DB) {
    
    // TODO: get from access token
    // authorizing wallet id
    authWalletID := "55c679145fe09c74ed000001"

    dbTx, err := db.GetPostgresHandleWithRepeatableReadTrans()
    if err != nil {
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // get wallet
    wallet, found, err := models.FindWalletByObjectID(dbTx, params["id"])
    if !found {
        dbTx.Rollback()
        services.Res(res).Error(404, "not_found", "wallet not found")
        return
    } else if err != nil {
        dbTx.Rollback()
        c.log.Error(err.Error())
        services.Res(res).Error(500, "", "server error")
        return
    }

    // ensure wallet matches authorizing wallet
    if wallet.ObjectID != authWalletID {
        dbTx.Rollback()
        services.Res(res).Error(401, "unauthorized", "client does not have permission to access wallet")
        return
    }

    // update lock state to false
    wallet.Lock = false

    // save and commit
    dbTx.Save(&wallet).Commit()
    services.Res(res).Json(wallet)
}
    