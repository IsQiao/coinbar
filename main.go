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
	url := "https://api1.binance.com/api/v3/ticker/price"
	var prices []SymbolPrice
	err := getJson(url, &prices)
	if err != nil {
		logrus.Error(err)
	}

	var symbolPriceMap = map[string]float64{}

	for _, item := range prices {
		symbolPriceMap[item.Symbol] = item.Price
	}

	whiteLists := []string{"NEARUSDT", "GTCUSDT", "BTCUSDT", "ETHUSDT", "SOLUSDT", "DOTUSDT", "KSMUSDT"}

	var childMenus []menuet.MenuItem
	for _, token := range whiteLists {
		if val, ok := symbolPriceMap[token]; ok {
			childMenus = append(childMenus, menuet.MenuItem{
				Text: fmt.Sprintf("%s: %v", token, val),
				Clicked: func() {

				},
			})
		}
	}

	menuet.App().Children = func() []menuet.MenuItem {
		return childMenus
	}
}

func priceLoop() {
	for {
		getPricesSetTitle()
	}
}

func main() {
	go priceLoop()
	menuet.App().Label = "qiao-coin-bar"
	menuet.App().SetMenuState(&menuet.MenuState{
		Title: "COIN",
	})
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
