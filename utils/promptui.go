package utils

import (
	"swift/domain"

	"github.com/manifoldco/promptui"
)

func SelectFromVMs() (string, error) {
	lib, err := InitLib()
	if err != nil {
		return "", err
	}
	vms, er := domain.VmNames(lib)
	if er != nil {
		return "", er
	}
	prompt := promptui.Select{
		Label: "Choose vm instance",
		Items: vms,
	}
	_, result, e := prompt.Run()
	if e != nil {
		return "", e
	}
	return result, nil
}

func ConfirmOperation() (string, error) {
	prompt := promptui.Prompt{
		Label:   "y/n",
		Default: "n",
	}
	result, e := prompt.Run()
	if e != nil {
		return "", e
	}
	return result, nil
}
