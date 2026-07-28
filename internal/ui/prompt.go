package ui

import (
	"github.com/manifoldco/promptui"
)

// SelectFromList presents an interactive selection prompt and returns the chosen item.
func SelectFromList(label string, items []string) (string, error) {
	prompt := promptui.Select{
		Label: label,
		Items: items,
	}
	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}

// ConfirmOperation asks the user for y/n confirmation (default: n).
func ConfirmOperation() (string, error) {
	prompt := promptui.Prompt{
		Label:   "y/n",
		Default: "n",
	}
	result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}
