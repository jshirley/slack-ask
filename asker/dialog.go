package asker

import (
	"fmt"
)

type DialogOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type DialogElement struct {
	Type        string         `json:"type"`
	Subtype     string         `json:"subtype"`
	Label       string         `json:"label"`
	Name        string         `json:"name"`
	Optional    bool           `json:"optional,omitempty"`
	Hint        string         `json:"hint,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
	Value       string         `json:"value,omitempty"`
	MinLength   int            `json:"min_length,omitempty" mapstructure:"min_length"`
	MaxLength   int            `json:"max_length,omitempty" mapstructure:"max_length"`
	Options     []DialogOption `json:"options,omitempty"`
}

type Dialog struct {
	CallbackID  string          `json:"callback_id"`
	Title       string          `json:"title"`
	SubmitLabel string          `json:"submit_label"`
	Elements    []DialogElement `json:"elements" mapstructure:"dialog"`
}

func (a *Asker) SetDialogElements(inDialog Dialog) error {
	if len(inDialog.Elements) < 1 {
		a.dialogElements = defaultElements()
		return nil
	}

	// Verify each element has the required fields
	for _, element := range inDialog.Elements {
		if element.Name == "" {
			return fmt.Errorf("Element does not have required `name` field, check configuration")
		}
		if element.Type == "" {
			return fmt.Errorf("Element `%s` does not have required `type` field, check configuration", element.Name)
		}
		if element.Type != "text" && element.Type != "textarea" && element.Type != "select" {
			return fmt.Errorf("Element `%s` has an invalid `type` field (%s is not text, textarea, or select)", element.Name, element.Type)
		}
		if element.Label == "" {
			return fmt.Errorf("Element `%s` does not have required `label` field, check configuration", element.Name)
		}
		if len(element.Label) > 24 {
			return fmt.Errorf("Element `%s`' label is over 24 characters, check configuration", element.Name)
		}
		if element.Type == "select" {
			if len(element.Options) < 0 {
				return fmt.Errorf("Element `%s` is a select, but has no options! Check configuration", element.Name)
			}
			for _, option := range element.Options {
				if option.Label == "" || option.Value == "" {
					return fmt.Errorf("Element `%s` options require both label and value! Check configuration", element.Name)
				}
				if len(option.Label) > 24 {
					return fmt.Errorf("Element `%s` option label `%s` is over 24 characters, check configuration", element.Name, option.Label)
				}
			}
		}
	}
	a.dialogElements = inDialog.Elements

	return nil
}

func (a *Asker) GetDialog(callback string) Dialog {
	return Dialog{
		CallbackID:  callback,
		Title:       "Ask a Question",
		SubmitLabel: "Ask!",
		Elements:    a.dialogElements,
	}
}

func defaultElements() []DialogElement {
	return []DialogElement{
		DialogElement{Type: "text", Label: "The one liner...", Name: "summary"},
		DialogElement{Type: "textarea", Label: "The details", Name: "description", Placeholder: "What have you tried so far, stack traces, etc."},
		DialogElement{
			Type:        "select",
			Label:       "Blocking?",
			Name:        "blocking",
			Placeholder: "Are you actively blocked and need someone now?",
			Value:       "no",
			Options: []DialogOption{
				DialogOption{"No", "no"},
				DialogOption{"Yes", "yes"},
				DialogOption{"I'm about to page!", "911"},
			},
		},
	}
}
