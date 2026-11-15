package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/marco-scarnato/multiservices-backend/services/project-d-ckguesser-be-go/internal/domain"
)

type fakeCatalogStore struct {
	cocktails   []domain.Cocktail
	ingredients []domain.Ingredient
	err         error
}

func (f *fakeCatalogStore) ListIngredients(_ context.Context) ([]domain.Ingredient, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.ingredients, nil
}

func (f *fakeCatalogStore) ListCocktails(_ context.Context) ([]domain.Cocktail, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.cocktails, nil
}

func TestCatalogHandlerSuccess(t *testing.T) {
	h := &catalogHandler{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		opTimeout: 2 * time.Second,
		store: &fakeCatalogStore{
			ingredients: []domain.Ingredient{
				{ID: "gin", Name: "Gin", Category: "spirit", Color: "#0099CC", Icon: "drop"},
			},
			cocktails: []domain.Cocktail{
				{
					ID:         "negroni",
					Name:       "Negroni",
					Aliases:    []string{"negroni cocktail"},
					Difficulty: "easy",
					Glass:      "rocks",
					Color:      "#B3202A",
					Ingredients: []domain.CocktailIngredient{
						{ID: "gin", Amount: "30 ml"},
					},
				},
			},
		},
	}

	var rawCocktails []byte
	h.respond(SubjectCocktailList, func(b []byte) error {
		rawCocktails = b
		return nil
	}, h.listCocktailsPayload)

	var decodedCocktails struct {
		Cocktails []domain.Cocktail `json:"cocktails"`
	}
	if err := json.Unmarshal(rawCocktails, &decodedCocktails); err != nil {
		t.Fatalf("unmarshal cocktails: %v", err)
	}
	if len(decodedCocktails.Cocktails) != 1 || decodedCocktails.Cocktails[0].ID != "negroni" {
		t.Fatalf("unexpected cocktails payload: %s", string(rawCocktails))
	}

	var rawIngredients []byte
	h.respond(SubjectIngredientList, func(b []byte) error {
		rawIngredients = b
		return nil
	}, h.listIngredientsPayload)

	var decodedIngredients struct {
		Ingredients []domain.Ingredient `json:"ingredients"`
	}
	if err := json.Unmarshal(rawIngredients, &decodedIngredients); err != nil {
		t.Fatalf("unmarshal ingredients: %v", err)
	}
	if len(decodedIngredients.Ingredients) != 1 || decodedIngredients.Ingredients[0].ID != "gin" {
		t.Fatalf("unexpected ingredients payload: %s", string(rawIngredients))
	}
}

func TestCatalogHandlerError(t *testing.T) {
	h := &catalogHandler{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		opTimeout: 2 * time.Second,
		store:     &fakeCatalogStore{err: errors.New("db down")},
	}

	var raw []byte
	h.respond(SubjectCocktailList, func(b []byte) error {
		raw = b
		return nil
	}, h.listCocktailsPayload)

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal error payload: %v", err)
	}
	if decoded["status"] != "error" {
		t.Fatalf("status = %v, want error", decoded["status"])
	}
}
