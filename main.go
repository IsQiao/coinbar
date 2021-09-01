package main

import (
	"coinbar/config"
	"coinbar/imgs"
	"encoding/json"
	"fmt"
	"github.com/gen2brain/dlgs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/getlantern/systray"
)

var coinListMapLock = new(sync.Mutex)
var favListMapLock = new(sync.Mutex)
var coinListMenuMap = make(map[*systray.MenuItem]string)
var favMenusMap = map[int]*systray.MenuItem{}

var cfg config.Config

func coinListContain(symbol string) bool {
	coinListMapLock.Lock()
	defer coinListMapLock.Unlock()
	for _, s := range coinListMenuMap {
		if symbol == s {
			return true
		}
	}
	return false
}

func loadCfg() error {
	config, err := config.GetCfg()
	if err != nil {
		return err
	}
	cfg = *config
	return nil
}

const MAX_FAV_COUNT = 50

func initFavList() {
	for i := 0; i < MAX_FAV_COUNT; i++ {
		menuItem := systray.AddMenuItem("unset", "")
		menuItem.Hide()
		setFavMenuList(menuItem, i)

		go func(i int) {
			for {
				select {
				case <-menuItem.ClickedCh:
					openFav(i)
				}
			}
		}(i)
	}
}

func openFav(index int) {
	favSymbolMapLock.Lock()
	defer favSymbolMapLock.Unlock()
	if symbol, ok := favSymbolMap[index]; ok {
		url := fmt.Sprintf("https://www.binance.com/zh-CN/trade/%s?layout=pro", symbol)
		openBrowser(url)
	}
}

var favSymbolMapLock = new(sync.Mutex)
var favSymbolMap = map[int]string{}

func refreshPrices() {
	defer time.Sleep(5 * time.Second)
	items, err := getData()
	if err != nil {
		logrus.Error(err)
	}

	favSymbolMapLock.Lock()
	defer favSymbolMapLock.Unlock()

	var FavItems []string
	var i = 0
	for _, item := range items {
		contained := containFavoriteItem(item.Symbol)
		if contained {
			FavItems = append(FavItems, fmt.Sprintf("%s: %v", SymbolFormat(item.Symbol), item.Price))
			favSymbolMap[i] = item.Symbol
			i++
		}
	}

	for i := 0; i < MAX_FAV_COUNT; i++ {
		if len(FavItems) >= i+1 {
			favMenusMap[i].SetTitle(FavItems[i])
			favMenusMap[i].Show()
		} else {
			favMenusMap[i].Hide()
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

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func containFavoriteItem(symbol string) bool {
	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()
	for _, favoriteItem := range cfg.FavoriteList {
		if symbol == favoriteItem {
			return true
		}
	}
	return false
}

func setFavMenuList(menu *systray.MenuItem, index int) {
	favListMapLock.Lock()
	defer favListMapLock.Unlock()
	favMenusMap[index] = menu
}

func setCoinListMap(menu *systray.MenuItem, symbol string) {
	coinListMapLock.Lock()
	defer coinListMapLock.Unlock()
	coinListMenuMap[menu] = symbol
}

func getCoinListItem(menu *systray.MenuItem) string {
	coinListMapLock.Lock()
	defer coinListMapLock.Unlock()
	if val, ok := coinListMenuMap[menu]; ok {
		return val
	}
	return ""
}

var coinListMenu *systray.MenuItem

func initCoinList() {
	coinListMenu = systray.AddMenuItem("Select Your Token", "")
	go loadCoinList()
}

func loadCoinList() {
	items, _ := getData()

	for _, item := range items {
		if symbol := SymbolFormat(item.Symbol); symbol != "" {
			checked := containFavoriteItem(item.Symbol)
			menu := coinListMenu.AddSubMenuItemCheckbox(SymbolFormat(item.Symbol), "", checked)

			setCoinListMap(menu, item.Symbol)

			go func() {
				for {
					select {
					case <-menu.ClickedCh:
						setList(menu)
					}
				}
			}()
		}
	}
}

func setList(item *systray.MenuItem) {
	currentSymbol := getCoinListItem(item)
	contained := containFavoriteItem(currentSymbol)

	cfg.Lock.Lock()
	defer cfg.Lock.Unlock()

	if !item.Checked() {
		if contained {
			return
		}
		cfg.FavoriteList = append(cfg.FavoriteList, currentSymbol)
		config.Save(cfg)
		item.Check()
	} else {
		if !contained {
			return
		}
		var currentFavList []string
		for _, s := range cfg.FavoriteList {
			if s != currentSymbol {
				currentFavList = append(currentFavList, s)
			}
		}
		cfg.FavoriteList = currentFavList
		config.Save(cfg)
		item.Uncheck()
	}
}

func priceLoop() {
	for {
		refreshPrices()
	}
}

func init() {
	loadCfg()
	if cfg.ProxyAddr != "" {
		refreshProxy(cfg.ProxyAddr)
	}
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(imgs.BtcIcon)
	initFavList()

	systray.AddSeparator()
	initCoinList()
	inputFav := systray.AddMenuItem("Input Your Token", "Input your favorite token")
	sysProxy := systray.AddMenuItem("Set Proxy", "Set system proxy")
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			case <-sysProxy.ClickedCh:
				setProxy()
			case <-inputFav.ClickedCh:
				showInputFav()
			}
		}
	}()

	go priceLoop()
}

func checkCoinListItem(symbol string) {
	coinListMapLock.Lock()
	defer coinListMapLock.Unlock()

	for menu, s := range coinListMenuMap {
		if s == symbol {
			menu.Check()
			return
		}
	}
}

func showInputFav() {
	input, _, err := dlgs.Entry("Input your favorite token", "Token", "")
	if err != nil {
		logrus.Error(err)
		return
	}

	input = strings.ReplaceAll(input, "/", "")

	if input == "" {
		return
	}

	if !coinListContain(input) {
		dlgs.Error("Error", "Invalid Token")
		return
	}

	if containFavoriteItem(input) {
		return
	}

	cfg.FavoriteList = append(cfg.FavoriteList, input)
	config.Save(cfg)

	checkCoinListItem(input)
}

func setProxy() {
	input, _, err := dlgs.Entry("Set System Proxy", "Proxy Address", cfg.ProxyAddr)
	if err != nil {
		logrus.Error(err)
		return
	}

	if input != "" && input != cfg.ProxyAddr {
		cfg.ProxyAddr = input
		config.Save(cfg)
		refreshProxy(input)
		go loadCoinList()
	}
}

func refreshProxy(proxyAddr string) {
	proxyUrl, _ := url.Parse(proxyAddr)
	transport := http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
	}
	myClient.Transport = &transport
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

var myClient = &http.Client{Timeout: 3 * time.Second}

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
