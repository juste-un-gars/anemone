# Anemone Translations

This directory contains translation files for Anemone.

## Current Languages

- `fr.json` - Français (479 keys)
- `en.json` - English (479 keys)

## Adding a New Language

To add a new language (e.g., Spanish):

### 1. Create the translation file

Copy an existing file and translate all values:

```bash
cp fr.json es.json
# Edit es.json and translate all values
```

### 2. Update `i18n.go`

Add the embed directive and load the translations:

```go
//go:embed locales/es.json
var esJSON []byte

// In New() function:
esTranslations := make(map[string]string)
if err := json.Unmarshal(esJSON, &esTranslations); err != nil {
    return nil, fmt.Errorf("failed to load Spanish translations: %w", err)
}
t.translations["es"] = esTranslations
```

### 3. Update GetAvailableLanguages()

Add the language to the list:

```go
func GetAvailableLanguages() []Language {
    return []Language{
        {Code: "fr", Name: "Français"},
        {Code: "en", Name: "English"},
        {Code: "es", Name: "Español"},  // NEW
    }
}
```

### 4. Test

```bash
go build -o anemone cmd/anemone/main.go
```

## Translation Keys Structure

Keys are organized hierarchically:

- `setup.*` - Setup page
- `login.*` - Login page
- `dashboard.*` - Dashboard
- `users.*` - User management
- `peers.*` - Peer management
- `shares.*` - Share management
- `admin.*` - Admin pages
- `restore.*` - Restore features
- etc.

## Validation

To check if all keys are present in all languages:

```bash
# Compare key counts
jq 'keys | length' fr.json
jq 'keys | length' en.json
jq 'keys | length' es.json  # Should all be equal

# Find missing keys in English compared to French
comm -23 <(jq -r 'keys[]' fr.json | sort) <(jq -r 'keys[]' en.json | sort)
```

## Best Practices

1. **Keep keys consistent** across all languages
2. **Use descriptive key names** (e.g., `admin.sync.config.frequency`)
3. **Test both languages** after adding new keys
4. **Preserve placeholders** like `{{username}}` in translations
5. **Keep line breaks** (`\n`) consistent with the source language
