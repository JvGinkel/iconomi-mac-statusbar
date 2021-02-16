package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	"github.com/jinzhu/configor"
)

// Config struct
type Config struct {
	Apikey    string
	Secretkey string
}

var (
	// C contains config struct
	C Config
)

// Init read the config file
func Init(f string) error {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	if len(f) == 0 {
		f = fmt.Sprintf("%s/.iconomi/config.yaml", usr.HomeDir)
	}
	if _, err := os.Stat(f); err == nil {
		// File found do nothing
	} else if os.IsNotExist(err) {
		// File missing
		if _, err := os.Stat(fmt.Sprintf("%s/.iconomi", usr.HomeDir)); os.IsNotExist(err) {
			os.Mkdir(fmt.Sprintf("%s/.iconomi", usr.HomeDir), 0775)
		}
		exampleConfig := []byte("---\napikey: APIKEYHERE\nsecretkey: SECRET_KEY_HERE\n")
		err := ioutil.WriteFile(fmt.Sprintf("%s/.iconomi/config.yaml", usr.HomeDir), exampleConfig, 0660)
		if err != nil {
			panic(err)
		}
		fmt.Println("Default config.yaml created in ~/.iconomi/config.yaml, set your apikey and secret")
		os.Exit(1)
	} else {
		panic(err)
	}
	e := configor.Load(&C, f)
	if e != nil {
		return fmt.Errorf("Config error: %s", e)
	}
	return nil
}
