# Floppa API

API co vraci obrazky zvieratiek (kocky, floppy, capybary...)

## Adding new animals

Edit `backend/main.go` and add a new entry to `animalConfigs`:

```go
{
    Endpoint:    "capybara",
    Title:       "Capybara",
    Description: "OK I pull up",
    Animal: &animals.PocketBaseAnimal{
        Name:           "capybara",
        CollectionName: "capybaras",
        PocketBaseURL:  cfg.PocketBaseURL,
    },
},
```

For local images, use `LocalAnimal` instead. The frontend will automatically pick up new animals from `/api/animals`.
