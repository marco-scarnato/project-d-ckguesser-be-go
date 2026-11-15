package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marco-scarnato/multiservices-backend/services/project-d-ckguesser-be-go/internal/domain"
)

// PostgresCatalogStore reads cocktails and ingredients from PostgreSQL.
type PostgresCatalogStore struct {
	pool *pgxpool.Pool
}

// NewPostgresCatalogStore constructs a PostgresCatalogStore backed by the given pool.
func NewPostgresCatalogStore(pool *pgxpool.Pool) *PostgresCatalogStore {
	return &PostgresCatalogStore{pool: pool}
}

// ListIngredients returns all ingredients ordered by name.
func (s *PostgresCatalogStore) ListIngredients(ctx context.Context) ([]domain.Ingredient, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("postgres pool not initialized")
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, category, color, icon
		FROM ingredients
		ORDER BY name ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query ingredients: %w", err)
	}
	defer rows.Close()

	ingredients := make([]domain.Ingredient, 0)
	for rows.Next() {
		var ing domain.Ingredient
		if err := rows.Scan(&ing.ID, &ing.Name, &ing.Category, &ing.Color, &ing.Icon); err != nil {
			return nil, fmt.Errorf("scan ingredient: %w", err)
		}
		ingredients = append(ingredients, ing)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ingredients: %w", err)
	}
	return ingredients, nil
}

// ListCocktails returns all cocktails with their ordered ingredients.
func (s *PostgresCatalogStore) ListCocktails(ctx context.Context) ([]domain.Cocktail, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("postgres pool not initialized")
	}

	cocktailRows, err := s.pool.Query(ctx, `
		SELECT id, name, aliases, difficulty, glass, color
		FROM cocktails
		ORDER BY name ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query cocktails: %w", err)
	}
	defer cocktailRows.Close()

	cocktails := make([]domain.Cocktail, 0)
	indexByID := make(map[string]int)
	for cocktailRows.Next() {
		var c domain.Cocktail
		if err := cocktailRows.Scan(&c.ID, &c.Name, &c.Aliases, &c.Difficulty, &c.Glass, &c.Color); err != nil {
			return nil, fmt.Errorf("scan cocktail: %w", err)
		}
		if c.Aliases == nil {
			c.Aliases = []string{}
		}
		c.Ingredients = []domain.CocktailIngredient{}
		indexByID[c.ID] = len(cocktails)
		cocktails = append(cocktails, c)
	}
	if err := cocktailRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cocktails: %w", err)
	}

	ingredientRows, err := s.pool.Query(ctx, `
		SELECT cocktail_id, ingredient_id, COALESCE(amount, '')
		FROM cocktail_ingredients
		ORDER BY cocktail_id ASC, position ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query cocktail ingredients: %w", err)
	}
	defer ingredientRows.Close()

	for ingredientRows.Next() {
		var cocktailID, ingredientID, amount string
		if err := ingredientRows.Scan(&cocktailID, &ingredientID, &amount); err != nil {
			return nil, fmt.Errorf("scan cocktail ingredient: %w", err)
		}
		if idx, ok := indexByID[cocktailID]; ok {
			cocktails[idx].Ingredients = append(cocktails[idx].Ingredients, domain.CocktailIngredient{
				ID:     ingredientID,
				Amount: amount,
			})
		}
	}
	if err := ingredientRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cocktail ingredients: %w", err)
	}

	return cocktails, nil
}
