package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/caseymrm/menuet"
)

func getPricesSetTitle() {
	defer time.Sleep(5 * time.Second)
	url := fmt.Sprintf("%s%s", "https://api1.binance.com", "/api/v3/ticker/price")
	var prices []SymbolPrice
	err := getJson(url, &prices)
	if err != nil {
		logrus.Error(err)
	}
	nearPrice := 0.0
	btcPrice := 0.0
	gtcPrice := 0.0

	for _, item := range prices {
		switch item.Symbol {
		case "NEARUSDT":
			nearPrice = item.Price
			break
		case "GTCUSDT":
			gtcPrice = item.Price
			break
		case "BTCUSDT":
			btcPrice = item.Price
			break
		}
	}

	title := fmt.Sprintf("GTC: %v NEAR: %v BTC: %v", gtcPrice, nearPrice, btcPrice)
	menuet.App().SetMenuState(&menuet.MenuState{
		Title: title,
	})

}

func priceLoop() {
	for {
		getPricesSetTitle()
	}
}

func main() {
	go priceLoop()
	menuet.App().Label = "qiao-coin-bar"
	menuet.App().RunApplication()
}

type SymbolPrice struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

var myClient = &http.Client{Timeout: 10 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &target)
	if err != nil {
		return err
	}
	return nil
}
