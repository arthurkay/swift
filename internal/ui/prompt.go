package ui

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
)

// VMItem holds a VM name and state for interactive display.
type VMItem struct {
	Name  string
	State string
}

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

// SelectVM presents an interactive VM selection prompt showing name and state.
// Returns the selected VM name.
func SelectVM(label string, items []VMItem) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select")
	}

	maxName := 0
	for _, it := range items {
		if len(it.Name) > maxName {
			maxName = len(it.Name)
		}
	}

	formatted := make([]string, len(items))
	for i, it := range items {
		formatted[i] = fmt.Sprintf("%-*s  (%s)", maxName, it.Name, it.State)
	}

	prompt := promptui.Select{
		Label: label,
		Items: formatted,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}:",
			Active:   "{{ \"▸\" | cyan }} {{ . | cyan }}",
			Inactive: "  {{ . | white }}",
			Selected: "{{ \"▸\" | cyan }} {{ . | cyan }}",
		},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	// Extract just the name from the formatted string
	name := strings.TrimSpace(strings.Split(formatted[idx], "  (")[0])
	return name, nil
}

// ConfirmOperation asks the user for y/n confirmation (default: n).
func ConfirmOperation() (string, error) {
	prompt := promptui.Prompt{
		Label:   "Confirm",
		Default: "n",
	}
	result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}
