package main

import (
	"coinbar/imgs"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/getlantern/systray"
)

func getCfgPath() string {
	dirname, _ := os.UserHomeDir()
	return dirname + "/.coinbar/config.json"
}

type Config struct {
	FavoriteList []string
	ProxyAddr    string
}

func loadCfg() (*Config, error) {
	path := getCfgPath()
	fmt.Println("Path: ", path)
	file, err := os.OpenFile(path, syscall.O_RDWR, os.ModeAppend)
	if os.IsNotExist(err) {
		init := Config{}
		saveCfg(init)
		return &init, nil
	}

	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	fileStr, err := ioutil.ReadAll(file)
	var data Config
	err = json.Unmarshal(fileStr, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func saveCfg(cfg Config) error {
	path := getCfgPath()
	dirPath := filepath.Dir(path)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.Mkdir(dirPath, 0755)
	}
	file, _ := json.MarshalIndent(cfg, "", " ")
	_ = ioutil.WriteFile(path, file, 0644)
	return nil
}

func getPricesSetTitle() {
	cfg, _ := loadCfg()

	fmt.Println("config.ProxyAddr", cfg.ProxyAddr)
	fmt.Println("config.FavoriteList", cfg.FavoriteList)
	if cfg.ProxyAddr != "" {
		os.Setenv("https_proxy", cfg.ProxyAddr)
		os.Setenv("http_proxy", cfg.ProxyAddr)
	}

	defer time.Sleep(5 * time.Second)
	items, err := getData()
	if err != nil {
		logrus.Error(err)
	}

	for _, item := range items {
		if val, ok := coinMenusMap[item.Symbol]; ok {
			if symbol := SymbolFormat(item.Symbol); symbol != "" {
				val.SetTitle(fmt.Sprintf("%s: %v", SymbolFormat(item.Symbol), item.Price))
			}
		}
	}

}

func getData() ([]SymbolPrice, error) {
	url := "https://api1.binance.com/api/v3/ticker/price"
	var items []SymbolPrice
	err := getJson(url, &items)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Symbol < items[j].Symbol
	})

	return items, nil
}

func loadSelectList() {
	selector := systray.AddMenuItem("select your list", "")
	items, _ := getData()

	for _, item := range items {
		if symbol := SymbolFormat(item.Symbol); symbol != "" {
			selector.AddSubMenuItemCheckbox(SymbolFormat(item.Symbol), "", false)
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
	systray.SetIcon(imgs.BtcIcon)
	cfg, _ := loadCfg()

	for _, white := range cfg.FavoriteList {
		menuItem := systray.AddMenuItem(fmt.Sprintf("%v loading...", SymbolFormat(white)), "")
		if _, ok := coinMenusMap[white]; !ok {
			coinMenusMap[white] = menuItem
		}
	}

	systray.AddSeparator()
	loadSelectList()
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
	list := []string{"USD", "USDT", "ETH", "BTC", "BNB", "USDC", "XRP", "DAI", "GBP", "EUR", "BRL", "TRY", "RUB", "AUD", "KRW"}
	found := false

	for _, s := range list {
		if strings.HasSuffix(symbol, s) {
			symbol = strings.TrimSuffix(symbol, s) + "/" + s
			found = true
			break
		}
	}

	if !found {
		fmt.Println(symbol)
		return ""
	}

	return symbol
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
