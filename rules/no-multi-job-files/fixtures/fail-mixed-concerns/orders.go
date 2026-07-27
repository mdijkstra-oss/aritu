package orders

import (
	"encoding/json"
	"net/http"
	"os"
)

type Order struct {
	ID    string `json:"id"`
	Total int    `json:"total"`
}

func ParseOrder(raw []byte) (Order, error) {
	var order Order
	err := json.Unmarshal(raw, &order)
	return order, err
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := SaveOrder(order); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func SaveOrder(order Order) error {
	encoded, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return os.WriteFile("orders/"+order.ID+".json", encoded, 0o644)
}
