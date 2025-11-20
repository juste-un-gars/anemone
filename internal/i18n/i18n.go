package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
)

//go:embed locales/fr.json
var frJSON []byte

//go:embed locales/en.json
var enJSON []byte

// Translator holds translation maps for different languages
type Translator struct {
	translations map[string]map[string]string
}

// Global translator instance
var globalTranslator *Translator

// Init initializes the global translator (for backward compatibility)
func Init(defaultLang string) error {
	t, err := New()
	if err != nil {
		return err
	}
	globalTranslator = t
	return nil
}

// New creates a new translator instance
func New() (*Translator, error) {
	t := &Translator{
		translations: make(map[string]map[string]string),
	}

	// Load French translations
	frTranslations := make(map[string]string)
	if err := json.Unmarshal(frJSON, &frTranslations); err != nil {
		return nil, fmt.Errorf("failed to load French translations: %w", err)
	}
	t.translations["fr"] = frTranslations

	// Load English translations
	enTranslations := make(map[string]string)
	if err := json.Unmarshal(enJSON, &enTranslations); err != nil {
		return nil, fmt.Errorf("failed to load English translations: %w", err)
	}
	t.translations["en"] = enTranslations

	return t, nil
}

// T translates using the global translator (for backward compatibility)
func T(lang, key string) string {
	if globalTranslator == nil {
		return key
	}
	return globalTranslator.T(lang, key)
}

// T translates a key for the given language
func (t *Translator) T(lang, key string) string {
	if translations, ok := t.translations[lang]; ok {
		if translation, ok := translations[key]; ok {
			return translation
		}
	}
	// Fallback to key if translation not found
	return key
}

// GetAvailableLanguages returns the list of available languages (global function)
func GetAvailableLanguages() []Language {
	return []Language{
		{Code: "fr", Name: "Français"},
		{Code: "en", Name: "English"},
	}
}

// GetAvailableLanguages returns the list of available languages (method)
func (t *Translator) GetAvailableLanguages() []Language {
	return GetAvailableLanguages()
}

// Language represents a supported language
type Language struct {
	Code string
	Name string
}

// FuncMap returns template functions for translations
func (t *Translator) FuncMap() template.FuncMap {
	return template.FuncMap{
		"T": func(lang, key string, args ...interface{}) string {
			translation := t.T(lang, key)
			// Support for placeholder replacement (e.g., {{username}})
			if len(args) > 0 {
				for i := 0; i < len(args); i += 2 {
					if i+1 < len(args) {
						placeholder := fmt.Sprintf("{{%v}}", args[i])
						value := fmt.Sprintf("%v", args[i+1])
						translation = strings.ReplaceAll(translation, placeholder, value)
					}
				}
			}
			return translation
		},
	}
}
