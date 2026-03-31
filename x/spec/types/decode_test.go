package types

import (
	"encoding/json"
	"testing"
)

func TestCoinUnmarshalStringAmount(t *testing.T) {
	data := []byte(`{"denom":"ulava","amount":"5000000000"}`)
	var c Coin
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("string amount: %v", err)
	}
	if c.Amount != 5000000000 {
		t.Fatalf("expected 5000000000, got %d", c.Amount)
	}
}

func TestCoinUnmarshalNumericAmount(t *testing.T) {
	data := []byte(`{"denom":"ulava","amount":123}`)
	var c Coin
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("numeric amount: %v", err)
	}
	if c.Amount != 123 {
		t.Fatalf("expected 123, got %d", c.Amount)
	}
}

func TestSpecProvidersTypesUnmarshalNull(t *testing.T) {
	data := []byte(`{"index":"TEST","providers_types":null}`)
	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("null providers_types: %v", err)
	}
}

func TestSpecProvidersTypesUnmarshalNumeric(t *testing.T) {
	data := []byte(`{"index":"TEST","providers_types":1}`)
	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("numeric providers_types: %v", err)
	}
	if s.ProvidersTypes != Spec_static {
		t.Fatalf("expected static(1), got %d", s.ProvidersTypes)
	}
}
