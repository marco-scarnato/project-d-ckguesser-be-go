package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/marco-scarnato/multiservices-backend/services/project-d-ckguesser-be-go/internal/domain"
	natsgo "github.com/nats-io/nats.go"
)

const (
	SubjectCocktailList   = "ckguesser-be.cocktail.list"
	SubjectIngredientList = "ckguesser-be.ingredient.list"

	fallbackErrorPayload = `{"status":"error","error":"internal encoding error"}`
)

type catalogHandler struct {
	logger    *slog.Logger
	store     domain.CatalogStore
	opTimeout time.Duration
}

// RegisterCatalogHandlers subscribes to NATS subjects for listing cocktails and ingredients.
func RegisterCatalogHandlers(
	conn *natsgo.Conn,
	logger *slog.Logger,
	store domain.CatalogStore,
	opTimeout time.Duration,
) ([]*natsgo.Subscription, error) {
	if conn == nil {
		return nil, fmt.Errorf("NATS connection is required")
	}
	if store == nil {
		return nil, fmt.Errorf("catalog store is required")
	}
	if opTimeout <= 0 {
		return nil, fmt.Errorf("operation timeout must be greater than zero")
	}
	if logger == nil {
		logger = slog.Default()
	}

	h := &catalogHandler{
		logger:    logger,
		store:     store,
		opTimeout: opTimeout,
	}

	cocktailsSub, err := conn.Subscribe(SubjectCocktailList, func(msg *natsgo.Msg) {
		h.respond(SubjectCocktailList, msg.Respond, h.listCocktailsPayload)
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe %q: %w", SubjectCocktailList, err)
	}

	ingredientsSub, err := conn.Subscribe(SubjectIngredientList, func(msg *natsgo.Msg) {
		h.respond(SubjectIngredientList, msg.Respond, h.listIngredientsPayload)
	})
	if err != nil {
		if unsubErr := cocktailsSub.Unsubscribe(); unsubErr != nil {
			logger.Error("failed to clean up cocktail subscription", "subject", SubjectCocktailList, "error", unsubErr)
		}
		return nil, fmt.Errorf("subscribe %q: %w", SubjectIngredientList, err)
	}

	return []*natsgo.Subscription{cocktailsSub, ingredientsSub}, nil
}

func (h *catalogHandler) respond(subject string, send func([]byte) error, buildPayload func(context.Context) any) {
	ctx, cancel := context.WithTimeout(context.Background(), h.opTimeout)
	defer cancel()

	data, err := json.Marshal(buildPayload(ctx))
	if err != nil {
		h.logger.Error("failed to marshal catalog response", "subject", subject, "error", err)
		data = []byte(fallbackErrorPayload)
	}
	if err := send(data); err != nil {
		h.logger.Error("failed to send NATS catalog response", "subject", subject, "error", err)
	}
}

func (h *catalogHandler) listCocktailsPayload(ctx context.Context) any {
	cocktails, err := h.store.ListCocktails(ctx)
	if err != nil {
		h.logger.Error("failed to load cocktails from store", "error", err)
		return map[string]any{
			"status": "error",
			"error":  "failed to load cocktails",
		}
	}
	return map[string]any{
		"cocktails": cocktails,
	}
}

func (h *catalogHandler) listIngredientsPayload(ctx context.Context) any {
	ingredients, err := h.store.ListIngredients(ctx)
	if err != nil {
		h.logger.Error("failed to load ingredients from store", "error", err)
		return map[string]any{
			"status": "error",
			"error":  "failed to load ingredients",
		}
	}
	return map[string]any{
		"ingredients": ingredients,
	}
}
