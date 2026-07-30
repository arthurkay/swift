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

// parsedTemplate is the cached, pre-parsed template.
var parsedTemplate = func() *template.Template {
	t, err := template.New("user-data").Parse(userDataTemplate)
	if err != nil {
		panic(fmt.Sprintf("parse user-data template: %v", err))
	}
	return t
}()

// UserData holds the cloud-init configuration for a VM.
type UserData struct {
	VMName       string
	HostName     string
	User         string
	Password     string
	Slug         string
	SSHPublicKey string
}

// NewUserData creates a UserData with sensible defaults.
func NewUserData(vmName, password, sshKey string) UserData {
	user := config.DefaultUser
	if user == "" {
		user = "swift"
	}
	if sshKey == "" {
		sshKey = "# no key provided"
	}
	s := slug.Make(vmName)
	return UserData{
		VMName:       vmName,
		HostName:     "swift-vm-instance",
		User:         user,
		Password:     password,
		Slug:         s,
		SSHPublicKey: sshKey,
	}
}

// CreateProjectFiles creates the per-VM directory and initializes
// user-data and meta-data files.
func (d UserData) CreateProjectFiles() error {
	projectDir, err := config.VMPath(d.Slug)
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

// CreateProjectDir creates the project directory for a VM and initializes
// user-data and meta-data files. Returns the project directory path.
func CreateProjectDir(name string) (string, error) {
	s := slug.Make(name)
	projectDir, err := config.VMPath(s)
	if err != nil {
		return "", fmt.Errorf("get project path: %w", err)
	}
	if err := os.MkdirAll(projectDir, config.FilePermission); err != nil {
		return "", fmt.Errorf("create project dir: %w", err)
	}
	for _, f := range []string{"user-data", "meta-data"} {
		if _, err := os.Create(projectDir + "/" + f); err != nil {
			return "", fmt.Errorf("create %s: %w", f, err)
		}
	}
	return projectDir, nil
}

// WriteCloudInit writes cloud-init user-data to the project directory.
// If userData is empty, a minimal default is generated.
func WriteCloudInit(projectDir, name, userData string) error {
	if userData == "" {
		userData = fmt.Sprintf("#cloud-config\nhostname: %s\nmanage_etc_hosts: true\n", name)
	}
	return os.WriteFile(projectDir+"/user-data", []byte(userData), 0644)
}

// CloudConfig renders the cloud-init user-data template and writes it to disk.
func (d UserData) CloudConfig() error {
	projectDir, err := config.VMPath(d.Slug)
	if err != nil {
		return fmt.Errorf("get project path: %w", err)
	}
	path := projectDir + "/user-data"

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create user-data file: %w", err)
	}
	defer file.Close()

	if err := parsedTemplate.Execute(file, d); err != nil {
		return fmt.Errorf("execute user-data template: %w", err)
	}
	return nil
}
