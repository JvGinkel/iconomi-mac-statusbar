package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/JvGinkel/iconomi-mac-statusbar/config"
	"github.com/caseymrm/menuet"
)

type balance struct {
	Currency string `json:"currency"`
	DaaList  []struct {
		Name    string `json:"name"`
		Ticker  string `json:"ticker"`
		Balance string `json:"balance"`
		Value   string `json:"value"`
	} `json:"daaList"`
	AssetList []struct {
		Name    string `json:"name"`
		Ticker  string `json:"ticker"`
		Balance string `json:"balance"`
		Value   string `json:"value"`
	} `json:"assetList"`
}

type coindeskCurrentPrice struct {
	Time struct {
		Updated    string    `json:"updated"`
		UpdatedISO time.Time `json:"updatedISO"`
		Updateduk  string    `json:"updateduk"`
	} `json:"time"`
	Disclaimer string `json:"disclaimer"`
	Bpi        struct {
		USD struct {
			Code        string  `json:"code"`
			Rate        string  `json:"rate"`
			Description string  `json:"description"`
			RateFloat   float64 `json:"rate_float"`
		} `json:"USD"`
		EUR struct {
			Code        string  `json:"code"`
			Rate        string  `json:"rate"`
			Description string  `json:"description"`
			RateFloat   float64 `json:"rate_float"`
		} `json:"EUR"`
	} `json:"bpi"`
}

func hmac512pass(password string, secret []byte) string {
	hmac512 := hmac.New(sha512.New, secret)
	hmac512.Write([]byte(password))
	return base64.StdEncoding.EncodeToString(hmac512.Sum(nil))
}

func getBTCPrice() string {
	// https://api.coindesk.com/v1/bpi/currentprice/EUR.json
	req, e := http.NewRequest("GET", "https://api.coindesk.com/v1/bpi/currentprice/EUR.json", nil)
	if e != nil {
		fmt.Print(e)
		return "NaN"
	}
	client := &http.Client{}
	res, e := client.Do(req)

	var btcprice coindeskCurrentPrice
	e = json.NewDecoder(res.Body).Decode(&btcprice)
	if e != nil {
		fmt.Printf("Error: %+v", e)
		return "NaN"
	}

	return fmt.Sprintf("%.2f", btcprice.Bpi.EUR.RateFloat)
}

func iconomiBalance() {
	// Read config file
	if e := config.Init(""); e != nil {
		panic(e)
	}

	for {
		timestamp := time.Now().UnixNano() / int64(time.Millisecond)
		combined := fmt.Sprintf("%dGET%s", timestamp, "/v1/user/balance")
		sign := hmac512pass(combined, []byte(config.C.Secretkey))
		// Make request
		req, err := http.NewRequest("GET", "https://api.iconomi.com/v1/user/balance?currency=EUR", nil)
		if err != nil {
			fmt.Print(err)
		}

		req.Header.Set("ICN-API-KEY", config.C.Apikey)
		req.Header.Set("ICN-TIMESTAMP", strconv.FormatInt(timestamp, 10))
		req.Header.Set("ICN-SIGN", sign)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Print(err)
		}
		var b balance
		var totalBalance float64
		var value float64

		e := json.NewDecoder(resp.Body).Decode(&b)
		if e != nil {
			fmt.Printf("Error: %+v", e)
		}
		// fmt.Printf("json: %+v", b)

		for _, v := range b.DaaList {
			// fmt.Println("\n v:", v.Value)
			value, _ = strconv.ParseFloat(v.Value, 64)
			totalBalance = totalBalance + value
		}
		for _, v := range b.AssetList {
			// fmt.Println("\n v:", v.Value)
			value, _ = strconv.ParseFloat(v.Value, 64)
			if b.Currency == v.Ticker {
				totalBalance = totalBalance + value
			}
		}

		menuet.App().SetMenuState(&menuet.MenuState{
			Title: fmt.Sprintf("📊 €%.2f / ₿ €%s", totalBalance, getBTCPrice()),
		})

		time.Sleep(time.Minute)
	}
}

func main() {
	// Fetch data
	go iconomiBalance()

	app := menuet.App()
	app.Name = "Iconomi Status"
	app.Label = "com.github.itshosted.iconomi-status"
	// app.Children = menuItems
	app.AutoUpdate.Version = "v0.1"
	app.RunApplication()
	// menuet.App().RunApplication()
}
