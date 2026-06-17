package bulutklinik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PaymentsService covers discount check, saved cards and the 3DS payment.
type PaymentsService struct{ t *transport }

// CheckDiscountCode validates a discount code. Note: the endpoint lives under the
// patients prefix.
func (s *PaymentsService) CheckDiscountCode(ctx context.Context, in DiscountInput) (json.RawMessage, error) {
	body := map[string]any{"checkType": in.CheckType, "discountCode": in.DiscountCode}
	if in.DoctorID != nil {
		body["doctorId"] = in.DoctorID
	}
	if in.OrderID != nil {
		body["orderId"] = in.OrderID
	}
	if in.SpecialServiceID != nil {
		body["specialServiceId"] = in.SpecialServiceID
	}
	if in.ProgramSlug != "" {
		body["programSlug"] = in.ProgramSlug
	}
	return s.t.do(ctx, request{http.MethodPost, "/patients/checkDiscountCode", authBearer, body})
}

// GetCards returns the saved cards.
func (s *PaymentsService) GetCards(ctx context.Context) (json.RawMessage, error) {
	return s.t.do(ctx, request{http.MethodGet, "/payments/getCards", authBearer, nil})
}

// SaveCard tokenizes a card.
func (s *PaymentsService) SaveCard(ctx context.Context, card CardInfo) (json.RawMessage, error) {
	return s.t.do(ctx, request{http.MethodPost, "/payments/saveCard", authBearer, card})
}

// Pay starts an appointment payment. On a 3DS flow the data contains
// payment3DUrl, a browser URL to open; the SDK does not follow it.
func (s *PaymentsService) Pay(ctx context.Context, in PaymentInput) (json.RawMessage, error) {
	appointmentType := in.AppointmentType
	if appointmentType == "" {
		appointmentType = "interview"
	}
	body := map[string]any{
		"doctorId":        in.DoctorID,
		"appointmentDate": in.AppointmentDate,
		"appointmentType": appointmentType,
		"is3D":            in.Is3D,
		"termsAccept":     in.TermsAccept,
		"saveCard":        in.SaveCard,
		"discountCode":    in.DiscountCode,
	}
	if in.CardID != nil {
		body["cardId"] = in.CardID
	}
	if in.CardInfo != nil {
		body["cardInfo"] = in.CardInfo
	}
	if in.CaseDetail != "" {
		body["caseDetail"] = in.CaseDetail
	}
	return s.t.do(ctx, request{http.MethodPost, "/payments/interviewPayment", authBearer, body})
}

// DeleteCard removes a saved card.
func (s *PaymentsService) DeleteCard(ctx context.Context, cardID any) (json.RawMessage, error) {
	path := fmt.Sprintf("/payments/deleteCard/%v", cardID)
	return s.t.do(ctx, request{http.MethodDelete, path, authBearer, nil})
}
