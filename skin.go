package bulutklinik

import (
	"context"
	"encoding/json"
	"net/http"
)

// SkinService covers "Cildimde Neyim Var" AI skin-lesion analysis.
type SkinService struct{ t *transport }

// Analyze submits one or more skin photos for classification. Each image is a
// loose record (for example {"image": "<base64>", "branch_id": 42}); branch_id
// is optional. Each image is classified (lesion label), summarized in Turkish
// (comment) and returned with quality flags, a confidence, possible ICD hints
// and an opaque case_detail blob that may be forwarded verbatim as a payment's
// caseDetail.
func (s *SkinService) Analyze(ctx context.Context, images []map[string]any) (json.RawMessage, error) {
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/imageCheck", auth: authBearer, body: map[string]any{"images": images}})
}

// MealsService covers AI meal-photo calorie/nutrition estimation (sibling of
// [SkinService]).
type MealsService struct{ t *transport }

// Analyze estimates calories and nutrition from a meal photo. The idiomatic
// input names map to the API's snake_case body (portion_size, portion_grams,
// meal_type). PortionGrams and Note are optional and are sent only when
// provided.
func (s *MealsService) Analyze(ctx context.Context, input MealInput) (json.RawMessage, error) {
	body := map[string]any{
		"image":        input.Image,
		"portion_size": input.PortionSize,
		"meal_type":    input.MealType,
	}
	if input.PortionGrams != nil {
		body["portion_grams"] = *input.PortionGrams
	}
	if input.Note != nil {
		body["note"] = *input.Note
	}
	return s.t.do(ctx, request{method: http.MethodPost, path: "/patients/imageAnalyzeMeal", auth: authBearer, body: body})
}
