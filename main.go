package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/getlantern/systray"
)

func getPricesSetTitle() {
	defer time.Sleep(5 * time.Second)
	url := "https://api1.binance.com/api/v3/ticker/price"
	var prices []SymbolPrice
	err := getJson(url, &prices)
	if err != nil {
		logrus.Error(err)
	}

	for _, item := range prices {
		if val, ok := coinMenusMap[item.Symbol]; ok {
			val.SetTitle(fmt.Sprintf("%s: %v", SymbolFormat(item.Symbol), item.Price))
		}
	}
}

func priceLoop() {
	for {
		getPricesSetTitle()
	}
}

func main() {
	go priceLoop()
	systray.Run(onReady, onExit)
}

var coinMenusMap = map[string]*systray.MenuItem{}

func onReady() {
	systray.SetTitle("COIN")

	whiteLists := []string{"NEARUSDT", "GTCUSDT", "BTCUSDT", "ETHUSDT", "SOLUSDT", "DOTUSDT", "KSMUSDT"}

	for _, white := range whiteLists {
		menuItem := systray.AddMenuItem(fmt.Sprintf("%v loading...", SymbolFormat(white)), "")
		if _, ok := coinMenusMap[white]; !ok {
			coinMenusMap[white] = menuItem
		}
	}
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
}

type SymbolPrice struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

func SymbolFormat(symbol string) string {
	result := symbol
	result = strings.Replace(result, "USDT", "/USDT", 1)
	return result
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

func format12(x float64) string {
	if x >= 1e12 {
		// Check to see how many fraction digits fit in:
		s := fmt.Sprintf("%.g", x)
		format := fmt.Sprintf("%%12.%dg", 12-len(s))
		return fmt.Sprintf(format, x)
	}

	// Check to see how many fraction digits fit in:
	s := fmt.Sprintf("%.0f", x)
	if len(s) == 12 {
		return s
	}
	format := fmt.Sprintf("%%%d.%df", len(s), 12-len(s)-1)
	return fmt.Sprintf(format, x)
}
