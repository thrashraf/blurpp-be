package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

type TotalSum struct {
	SumTotal int `db:"sum_total"`
}

func dashboardHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		vendorId := c.QueryParam("vendorId")

		//? Daily
		previousDate := time.Now().In(tz).AddDate(0, 0, -1).Format("2006-01-02")
		currentDate := time.Now().In(tz).Format("2006-01-02")
		nextDayDate := time.Now().In(tz).AddDate(0, 0, 1).Format("2006-01-02")

		//? Weekly
		previousWeek := time.Now().In(tz).AddDate(0, 0, -7).Format("2006-01-02")
		startWeek := time.Now().In(tz).AddDate(0, 0, -6).Format("2006-01-02")
		endWeek := time.Now().In(tz).AddDate(0, 0, 1).Format("2006-01-02")

		//? Monthly
		previousMonth := time.Now().In(tz).AddDate(0, -1, 0).Format("2006-01-02")
		startMonth := time.Now().In(tz).AddDate(0, 0, -29).Format("2006-01-02")
		endMonth := time.Now().In(tz).AddDate(0, 0, 1).Format("2006-01-02")

		daily, err := fetchOrdersData(app, vendorId, previousDate, currentDate, nextDayDate)
		if err != nil {
			log.Println("Error fetching daily data:", err)
			return err
		}

		weekly, err := fetchOrdersData(app, vendorId, previousWeek, startWeek, endWeek)
		if err != nil {
			log.Println("Error fetching weekly data:", err)
			return err
		}

		monthly, err := fetchOrdersData(app, vendorId, previousMonth, startMonth, endMonth)
		if err != nil {
			log.Println("Error fetching monthly data:", err)
			return err
		}

		topSales, err := getTopSalesProducts(app, vendorId)
		if err != nil {
			log.Println("Error fetching top sales data:", err)
			return err
		}

		sales, err := getSalesForPast7Days(app, vendorId)
		if err != nil {
			log.Println("Error fetching sales data:", err)
			return err
		}

		dashboardData := Data{
			Daily:    daily,
			Weekly:   weekly,
			Monthly:  monthly,
			TopSales: topSales,
			Sales:    sales,
		}

		return c.JSON(http.StatusOK, dashboardData)
	}
}

func fetchOrdersData(app *pocketbase.PocketBase, vendorId string, previous string, current string, end string) (salesReport, error) {

	reportToday, err := querySalesReport(app, vendorId, current, end)
	if err != nil {
		log.Println("Error fetching data for today from the database:", err)
		return salesReport{}, err
	}

	reportYesterday, err := querySalesReport(app, vendorId, previous, current)
	if err != nil {
		log.Println("Error fetching data for yesterday from the database:", err)
		return salesReport{}, err
	}

	report, err := calculate(reportToday, reportYesterday)

	return report, nil
}

func querySalesReport(app *pocketbase.PocketBase, vendorId, startDate, endDate string) (int, error) {
	var totalSum TotalSum

	query := app.Dao().DB().NewQuery(`
		SELECT coalesce(SUM(total_price), 0.00) AS sum_total FROM orders
		WHERE (vendor_id = {:vendorId} AND order_status = 'paid')
		AND DATETIME(created, '+8 hours') >= {:startDate}
		AND DATETIME(created, '+8 hours') < {:endDate}
	`).Bind(dbx.Params{
		"vendorId":  vendorId,
		"startDate": startDate,
		"endDate":   endDate,
	})

	if err := query.One(&totalSum); err != nil {
		return 0, err
	}

	return totalSum.SumTotal, nil
}

func calculate(currentValue int, previousValue int) (salesReport, error) {
	var report salesReport

	percentage := func() int {
		if previousValue == 0 {
			return 100
		}
		currentPercentage := int(((currentValue - previousValue) / previousValue) * 100)

		if currentPercentage <= 100 {
			return currentPercentage
		}

		return 100
	}()

	isPositive := currentValue > previousValue

	report.Percentage = int(percentage)
	report.IsPositive = isPositive
	report.Total = int(currentValue)

	return report, nil
}

// create a function to calculate the top sales products by quantity and vendorId
func getTopSalesProducts(app *pocketbase.PocketBase, vendorId string) ([]TopSale, error) {
	var topSales []TopSale

	query := app.Dao().DB().NewQuery(`
		SELECT
			p.id AS product_id,
			p.product_name,
			p.product_price,
			SUM(json_extract(o.orders_detail, '$.quantity')) AS total
		FROM
			orders o
		JOIN products p ON json_extract(o.orders_detail, '$.product_id') = p.id
		WHERE o.vendor_id = {:vendorID}
		GROUP BY
			p.id, p.product_name
		ORDER BY
			total DESC
		LIMIT 10
	`).Bind(dbx.Params{
		"vendorID": vendorId,
	})

	if err := query.All(&topSales); err != nil {
		return nil, err
	}

	return topSales, nil
}

func getSalesForPast7Days(app *pocketbase.PocketBase, vendorId string) ([]Sale, error) {
	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	// Initialize the result slice with all days and sales set to 0
	result := make([]Sale, 7)
	for i, dayName := range dayNames {
		result[i].Day = dayName
		result[i].Sales = 0
	}

	var sale []Sale

	query := app.Dao().DB().NewQuery(`
		SELECT
			strftime('%w', o.created) AS day,
			SUM(total_price) AS sales
		FROM
			orders o
		WHERE o.vendor_id = {:vendorID}
		AND DATETIME(o.created, '+8 hours') >= DATETIME('now', '-6 days', '+8 hours')
		GROUP BY
			day
		ORDER BY
			day
	`).Bind(dbx.Params{
		"vendorID": vendorId,
	})

	if err := query.All(&sale); err != nil {
		log.Println("Error fetching sales data:", err)
		return nil, err
	}

	// Update the sales values in the result slice based on fetched data
	for _, data := range sale {
		log.Printf("Day: %s, Sales: %d\n", data.Day, data.Sales)
		dayOfWeek, _ := strconv.Atoi(data.Day)
		result[dayOfWeek].Sales = data.Sales
	}

	return result, nil
}
