package cloudinit

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"swift/internal/config"

	"github.com/gosimple/slug"
)

//go:embed user_data.tmpl
var userDataTemplate string

// UserData holds the cloud-init configuration for a VM.
type UserData struct {
	VMName   string
	HostName string
	User     string
	Password string
}

// NewUserData creates a UserData with sensible defaults.
func NewUserData(vmName, password string) UserData {
	user := config.DefaultUser
	if user == "" {
		user = "swift"
	}
	return UserData{
		VMName:   vmName,
		HostName: "swift-vm-instance",
		User:     user,
		Password: password,
	}
}

// CreateProjectFiles creates the per-VM directory and initializes
// user-data and meta-data files.
func (d UserData) CreateProjectFiles() error {
	projectDir, err := config.VMPath(slug.Make(d.VMName))
	if err != nil {
		return fmt.Errorf("get project path: %w", err)
	}
	if err := os.MkdirAll(projectDir, config.FilePermission); err != nil {
		return fmt.Errorf("create project dir: %w", err)
	}
	for _, f := range []string{"user-data", "meta-data"} {
		if _, err := os.Create(projectDir + "/" + f); err != nil {
			return fmt.Errorf("create %s: %w", f, err)
		}
	}
	return nil
}

// CloudConfig renders the cloud-init user-data template and writes it to disk.
func (d UserData) CloudConfig() error {
	projectDir, err := config.VMPath(slug.Make(d.VMName))
	if err != nil {
		return fmt.Errorf("get project path: %w", err)
	}
	path := projectDir + "/user-data"

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create user-data file: %w", err)
	}
	defer file.Close()

	t, err := template.New("user-data").Parse(userDataTemplate)
	if err != nil {
		return fmt.Errorf("parse user-data template: %w", err)
	}
	if err := t.Execute(file, d); err != nil {
		return fmt.Errorf("execute user-data template: %w", err)
	}
	return nil
}
