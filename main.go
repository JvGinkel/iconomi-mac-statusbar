package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os/user"
	"strconv"
	"time"

	"github.com/JvGinkel/iconomi-mac-statusbar/config"
	"github.com/caseymrm/menuet"
)

var (
	// IconomiBalanceResponse contains the latest api request data
	IconomiBalanceResponse balance
	// BTCprice contains USD and EUR price
	BTCprice BTCpriceStruct
	// DisplayCurrency show prices in EUR / USD
	DisplayCurrency = "EUR"
)

// BTCpriceStruct contains USD and EUR current price
type BTCpriceStruct struct {
	USDrateFloat float64
	EURrateFloat float64
}
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

func hmac512pass(password string, secret []byte) string {
	hmac512 := hmac.New(sha512.New, secret)
	hmac512.Write([]byte(password))
	return base64.StdEncoding.EncodeToString(hmac512.Sum(nil))
}

type priceResponse struct {
	Bitcoin struct {
		USD float64 `json:"usd"`
		EUR float64 `json:"eur"`
	} `json:"bitcoin"`
}

func getBTCPriceCoingecko() (float64, float64, error) {
	// build a context with timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// CoinGecko simple price endpoint
	url := "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd,eur"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("doing http request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("unexpected status: %s", res.Status)
	}

	var pr priceResponse
	if err := json.NewDecoder(res.Body).Decode(&pr); err != nil {
		return 0, 0, fmt.Errorf("decoding response: %w", err)
	}

	return pr.Bitcoin.USD, pr.Bitcoin.EUR, nil
}

func getBTCPrice() {
	for {
		// Fetch BTC price
		usd, eur, err := getBTCPriceCoingecko()
		if err != nil {
			fmt.Println("Error fetching BTC price:", err)
			return
		}
		BTCprice.USDrateFloat = usd
		BTCprice.EURrateFloat = eur

		if config.Verbose {
			fmt.Printf("USD: %.2f EUR: %.2f\n", BTCprice.USDrateFloat, BTCprice.EURrateFloat)
		}
		setMenu()
		time.Sleep(time.Second * 60)
	}
}

func iconomiBalance() {

	for {
		timestamp := time.Now().UnixNano() / int64(time.Millisecond)
		combined := fmt.Sprintf("%dGET%s", timestamp, "/v1/user/balance")
		sign := hmac512pass(combined, []byte(config.C.Secretkey))
		// Make request
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.iconomi.com/v1/user/balance?currency=%s", DisplayCurrency), nil)
		if err != nil {
			fmt.Println(err)
		}

		req.Header.Set("ICN-API-KEY", config.C.Apikey)
		req.Header.Set("ICN-TIMESTAMP", strconv.FormatInt(timestamp, 10))
		req.Header.Set("ICN-SIGN", sign)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
		} else {

			if resp.StatusCode != 200 {
				responseDump, err := httputil.DumpResponse(resp, true)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(string(responseDump))
			}

			e := json.NewDecoder(resp.Body).Decode(&IconomiBalanceResponse)
			if e != nil {
				fmt.Printf("Error: %+v", e)
			}
			if config.Verbose {
				fmt.Printf("json: %+v\n", IconomiBalanceResponse)
			}

			resp.Body.Close()
			setMenu()
		}
		time.Sleep(time.Second * 60)
	}
}

func main() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	configPath := fmt.Sprintf("%s/.iconomi/config.yaml", usr.HomeDir)
	// showVersion := false
	flag.StringVar(&configPath, "c", configPath, "Path to config")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose-mode (log more)")
	flag.Parse()

	// Read config file
	if e := config.Init(configPath); e != nil {
		panic(e)
	}

	// Fetch data
	go iconomiBalance()
	go getBTCPrice()

	app := menuet.App()
	app.Name = "Iconomi Status"
	app.Label = "com.github.itshosted.iconomi-status"
	app.Children = menuItems
	app.AutoUpdate.Version = "0.0.1"
	app.RunApplication()
	// menuet.App().RunApplication()
}

func iconomiTotal() float64 {
	var totalBalance float64
	var value float64

	for _, v := range IconomiBalanceResponse.DaaList {
		// fmt.Println("\n v:", v.Value)
		value, _ = strconv.ParseFloat(v.Value, 64)
		totalBalance = totalBalance + value
	}
	for _, v := range IconomiBalanceResponse.AssetList {
		// fmt.Println("\n v:", v.Value)
		value, _ = strconv.ParseFloat(v.Value, 64)
		// if IconomiBalanceResponse.Currency == v.Ticker {
		totalBalance = totalBalance + value
		// }
	}
	return totalBalance
}

func menuItems() []menuet.MenuItem {

	var items []menuet.MenuItem
	for _, v := range IconomiBalanceResponse.DaaList {
		// fmt.Println("\n v:", v.Value)
		value, _ := strconv.ParseFloat(v.Value, 64)
		// text := fmt.Sprintf("%-20s €%.2f", v.Name, value)
		text := ""
		if DisplayCurrency == "EUR" {
			text = fmt.Sprintf("%-20s €%.2f", v.Name, value)
		} else {
			text = fmt.Sprintf("%-20s $%.2f", v.Name, value)
		}

		items = append(items, menuet.MenuItem{
			Text: text,
		})
		items = append(items, menuet.MenuItem{
			Type: menuet.Separator,
		})
	}
	for _, v := range IconomiBalanceResponse.AssetList {
		// fmt.Println("\n v:", v.Value)
		value, _ := strconv.ParseFloat(v.Value, 64)
		text := ""
		if DisplayCurrency == "EUR" {
			text = fmt.Sprintf("%-20s €%.2f", v.Name, value)
		} else {
			text = fmt.Sprintf("%-20s $%.2f", v.Name, value)
		}

		items = append(items, menuet.MenuItem{
			Text: text,
		})
		items = append(items, menuet.MenuItem{
			Type: menuet.Separator,
		})
	}

	currency := menuet.Defaults().Boolean("currency")
	items = append(items, menuet.MenuItem{
		Text: "Currency",
		Children: func() []menuet.MenuItem {
			return []menuet.MenuItem{
				{
					Text: "USD",
					Clicked: func() {
						menuet.Defaults().SetBoolean("currency", false)
						setMenu()
					},
					State: !currency,
				},
				{
					Text: "EUR",
					Clicked: func() {
						menuet.Defaults().SetBoolean("currency", true)
						setMenu()
					},
					State: currency,
				},
			}
		},
	})

	return items
}

func setMenu() {
	currency := menuet.Defaults().Boolean("currency")
	var btcpricetext string
	var ticker string
	if !currency {
		DisplayCurrency = "USD"
		btcpricetext = fmt.Sprintf("$%.2f", BTCprice.USDrateFloat)
		ticker = "$"
	} else {
		DisplayCurrency = "EUR"
		btcpricetext = fmt.Sprintf("€%.2f", BTCprice.EURrateFloat)
		ticker = "€"
	}

	menuet.App().SetMenuState(&menuet.MenuState{
		Title: fmt.Sprintf("📊 %s%.2f / ₿ %s", ticker, iconomiTotal(), btcpricetext),
	})
	menuet.App().MenuChanged()
}
