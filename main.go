package main

import (
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Data struct {
	Daily    salesReport `json:"daily"`
	Weekly   salesReport `json:"weekly"`
	Monthly  salesReport `json:"monthly"`
	Target   TargetData  `json:"target"`
	TopSales []TopSale   `json:"topSales"`
	Sales    []Sale      `json:"sales"`
}

type salesReport struct {
	Percentage int  `json:"percentage"`
	IsPositive bool `json:"isPositive"`
	Total      int  `json:"total"`
}

type TargetData struct {
	Target  int `json:"target"`
	Current int `json:"current"`
}

type TopSale struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	ProductPrice float64 `json:"product_price"`
	Total        int     `json:"total"`
}

type Sale struct {
	Day   string `json:"day"`
	Sales int    `json:"sales"`
}

var tz *time.Location

func init() {
	var err error
	tz, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		//? handle the error
		log.Println("Something when wrong")
		return
	}
}

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/dashboard",
			Handler: dashboardHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.ActivityLogger(app),
			},
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
