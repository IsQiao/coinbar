package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

func getCfgPath() string {
	dirname, _ := os.UserHomeDir()
	return dirname + "/.coinbar/config.json"
}

type Config struct {
	Lock         *sync.Mutex
	FavoriteList []string
	ProxyAddr    string
}

func GetCfg() (*Config, error) {
	path := getCfgPath()
	fmt.Println("Path: ", path)
	file, err := os.OpenFile(path, syscall.O_RDWR, os.ModeAppend)
	if os.IsNotExist(err) {
		init := Config{
			Lock: new(sync.Mutex),
		}
		Save(init)
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

	data.Lock = new(sync.Mutex)

	return &data, nil
}

func Save(cfg Config) error {
	path := getCfgPath()
	dirPath := filepath.Dir(path)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.Mkdir(dirPath, 0755)
	}
	file, _ := json.MarshalIndent(cfg, "", " ")
	_ = ioutil.WriteFile(path, file, 0644)
	return nil
}
