package main

import (
    "github.com/ownode/controllers"
    "github.com/ownode/services"
    "github.com/ownode/config"
    "github.com/ownode/policies"
    "github.com/ownode/middlewares"
    "github.com/go-martini/martini"
    "net/http"
)


func main() {

    db := &services.DB{}

    // connect to postgres
    if _, err := db.ConnectToPostgres("user=ned dbname=localdb sslmode=disable"); err != nil {
        config.Log().Error(err)
        return
    }
    config.PostgresAutoMigration(db)

    // create martini object
    m := martini.Classic()
    m.Map(db)
    m.Map(config.Log())

    // add auxilliary request context as a service.
    // service is created and added in every new request.
    // auxilliary request wraps the original request 
    m.Use(func(c martini.Context, req *http.Request) {
        c.Map(services.NewAuxRequestContext(c, req))
    })

    // define policies for specific routes
    m.Use(middlewares.Policies(map[string][]middlewares.PolicyFunc{
        "POST /api/token":          []middlewares.PolicyFunc{ policies.MustHaveAuthHeader, policies.MustBeBasic, },  
    }))

    // define routes
    m.Get("/", controllers.APP.Index)

    m.Group("/api", func(r martini.Router) {
        r.Post("/token", controllers.Auth.GetToken)
    })

    m.Group("/v1", func(r martini.Router) {
        r.Post("/services", controllers.Service.Create)
        r.Get("/services/:id", controllers.Service.Get)
        r.Put("/services/enable_issuer", controllers.Service.EnableIssuer)
        
        r.Post("/identities", controllers.Identity.Create)
        r.Post("/identities/renew_soul", controllers.Identity.RenewSoul)
        r.Get("/identities/:id", controllers.Identity.Get)
        
        r.Post("/wallets", controllers.Wallet.Create)
        r.Get("/wallets/:id", controllers.Wallet.Get)
        r.Get("/wallets/:id/objects", controllers.Wallet.List)
        r.Get("/wallets/:id/numbers", controllers.Wallet.Numbers)
        r.Put("/wallets/:id/lock", controllers.Wallet.Lock)
        r.Put("/wallets/:id/open", controllers.Wallet.Open)

        r.Post("/issuers", controllers.Issuer.Create)

        r.Post("/objects", controllers.Object.Create)
        r.Get("/objects/:id", controllers.Object.Get)
        r.Post("/objects/merge", controllers.Object.Merge)
        r.Post("/objects/divide", controllers.Object.Divide)
        r.Post("/objects/subtract", controllers.Object.Subtract)
        r.Put("/objects/:id/open", controllers.Object.Open)
        r.Put("/objects/:id/lock", controllers.Object.Lock)
        r.Post("/objects/charge", controllers.Object.Charge)
    })

    m.Run()
}