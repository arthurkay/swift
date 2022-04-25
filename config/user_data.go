package config

import (
	"fmt"
	"html/template"
	"os"
	"swift/utils"

	"github.com/gosimple/slug"
)

type UserData struct {
	VMName   string
	HostName string
	User     string
	Password string
}

// cloudConfig populates the user-data configuration file
// with preferences for the user for vm instance and other details
func (d UserData) CloudConfig() error {
	configDir, erro := utils.SwiftHome()

	if erro != nil {
		return fmt.Errorf("%v", erro)
	}
	path := configDir + "/" + slug.Make(d.VMName) + "/user-data"
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%v", erro)
	}
	defer file.Close()
	t, er := template.ParseFiles("config/user_data.tmpl")
	if er != nil {
		return fmt.Errorf("%v", er)
	}
	return t.Execute(file, d)
}

// createProject creates the project directory and the
// projects user-data amd meta-data files
func (d UserData) CreateProjectFiles() error {
	configDir, erro := utils.SwiftHome()
	if erro != nil {
		return fmt.Errorf("%v", erro)
	}
	projectDir := configDir + "/" + slug.Make(d.VMName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}
	if _, er := os.Create(projectDir + "/user-data"); er != nil {
		return er
	}
	if _, e := os.Create(projectDir + "/meta-data"); e != nil {
		return e
	}
	return nil
}
