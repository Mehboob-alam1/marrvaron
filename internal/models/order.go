package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderStatus definisce lo stato dell'ordine
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusSaved      OrderStatus = "saved" // Salvato per pagamento successivo
	OrderStatusPaid       OrderStatus = "paid"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// PaymentStatus definisce lo stato del pagamento
type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPaid    PaymentStatus = "paid"
	PaymentStatusFailed  PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

// PaymentMethod definisce il metodo di pagamento
type PaymentMethod string

const (
	PaymentMethodCard       PaymentMethod = "card"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCash       PaymentMethod = "cash"
	PaymentMethodWallet     PaymentMethod = "wallet"
)

// Order rappresenta un ordine nel sistema
type Order struct {
	ID              uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrderNumber     string        `gorm:"uniqueIndex;not null" json:"order_number"`
	UserID          *uuid.UUID    `gorm:"type:uuid;index" json:"user_id"` // Nullable per ordini guest
	DistributorID   *uuid.UUID    `gorm:"type:uuid;index" json:"distributor_id"`
	CourierID       *uuid.UUID    `gorm:"type:uuid;index" json:"courier_id"`
	Status          OrderStatus   `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	PaymentStatus   PaymentStatus `gorm:"type:varchar(20);default:'pending'" json:"payment_status"`
	PaymentMethod   PaymentMethod `json:"payment_method"`
	SubTotal        float64       `gorm:"not null" json:"sub_total"`
	TaxAmount       float64       `gorm:"default:0" json:"tax_amount"`
	DiscountAmount  float64       `gorm:"default:0" json:"discount_amount"`
	ShippingCost    float64       `gorm:"default:0" json:"shipping_cost"`
	TotalAmount     float64       `gorm:"not null" json:"total_amount"`
	Currency        string        `gorm:"default:USD" json:"currency"`
	ShippingAddress string        `gorm:"type:text" json:"shipping_address"`
	BillingAddress  string        `gorm:"type:text" json:"billing_address"`
	Notes           string        `gorm:"type:text" json:"notes"`
	IsGuestOrder    bool          `gorm:"default:false" json:"is_guest_order"`
	GuestEmail      string        `json:"guest_email"`
	GuestPhone      string        `json:"guest_phone"`
	PaidAt          *time.Time    `json:"paid_at"`
	ShippedAt       *time.Time    `json:"shipped_at"`
	DeliveredAt     *time.Time    `json:"delivered_at"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`

	// Relazioni
	User            *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Distributor     *Distributor `gorm:"foreignKey:DistributorID" json:"distributor,omitempty"`
	Items           []OrderItem  `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Payments        []Payment    `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
}

// OrderItem rappresenta un item in un ordine
type OrderItem struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrderID         uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID       uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	InventoryItemID *uuid.UUID `gorm:"type:uuid;index" json:"inventory_item_id"`
	QRCodeID        *uuid.UUID `gorm:"type:uuid;index" json:"qr_code_id"`
	Quantity        int       `gorm:"not null" json:"quantity"`
	UnitPrice       float64   `gorm:"not null" json:"unit_price"`
	DiscountAmount  float64   `gorm:"default:0" json:"discount_amount"`
	TotalPrice      float64   `gorm:"not null" json:"total_price"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relazioni
	Order           Order     `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Product         Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// Payment rappresenta un pagamento per un ordine
type Payment struct {
	ID              uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrderID         uuid.UUID     `gorm:"type:uuid;not null;index" json:"order_id"`
	Amount          float64       `gorm:"not null" json:"amount"`
	Currency        string        `gorm:"default:USD" json:"currency"`
	Status          PaymentStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Method          PaymentMethod `json:"method"`
	TransactionID   string        `gorm:"index" json:"transaction_id"`
	PaymentGateway  string        `json:"payment_gateway"`
	GatewayResponse string        `gorm:"type:jsonb" json:"gateway_response"`
	PaidAt          *time.Time    `json:"paid_at"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`

	// Relazioni
	Order           Order     `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}

// Cart rappresenta il carrello di un utente
type Cart struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          *uuid.UUID `gorm:"type:uuid;index" json:"user_id"` // Nullable per carrelli guest
	SessionID       string    `gorm:"index" json:"session_id"` // Per carrelli guest
	ProductID       uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	QRCodeID        *uuid.UUID `gorm:"type:uuid;index" json:"qr_code_id"` // Se aggiunto tramite scan
	Quantity        int       `gorm:"default:1" json:"quantity"`
	UnitPrice       float64   `gorm:"not null" json:"unit_price"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relazioni
	User            *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Product         Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	QRCode          *QRCode   `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
}

// BeforeCreate hooks
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.OrderNumber == "" {
		o.OrderNumber = generateOrderNumber()
	}
	return nil
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return nil
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (c *Cart) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// generateOrderNumber genera un numero ordine univoco
func generateOrderNumber() string {
	return "ORD-" + uuid.New().String()[:8]
}
