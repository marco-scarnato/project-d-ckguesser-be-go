package domain

import "context"

// Ingredient represents a cocktail ingredient available in the catalog.
type Ingredient struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Color    string `json:"color"`
	Icon     string `json:"icon"`
}

// CocktailIngredient links an ingredient ID and optional amount to a cocktail recipe.
type CocktailIngredient struct {
	ID     string `json:"id"`
	Amount string `json:"amount,omitempty"`
}

// Cocktail represents a cocktail recipe stored in the catalog.
type Cocktail struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Aliases     []string             `json:"aliases"`
	Difficulty  string               `json:"difficulty"`
	Glass       string               `json:"glass"`
	Color       string               `json:"color"`
	Ingredients []CocktailIngredient `json:"ingredients"`
}

// CatalogStore defines read operations for cocktails and ingredients.
type CatalogStore interface {
	ListIngredients(ctx context.Context) ([]Ingredient, error)
	ListCocktails(ctx context.Context) ([]Cocktail, error)
}
