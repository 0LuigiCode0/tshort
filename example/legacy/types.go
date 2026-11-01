package legacy

//go:generate mockgen -destination ./mocks/mocks.go -package=mocks . CustomerRepo,PriceService,TaxService,LoyaltyService,DiscountService,StockService,WarehouseService,FraudChecker,PaymentGateway,CreditService,ShippingService,Notifier,InvoiceService,EventBus,AuditLogger

// Доменные типы «производного» сервиса заказов.
// Намеренно устаревший стиль: float64 для денег, string-константы, всё в одном месте.

type OrderKind int

const (
	KindPhysical     OrderKind = 1
	KindDigital      OrderKind = 2
	KindSubscription OrderKind = 3
)

type PayMethod string

const (
	PayCard    PayMethod = "card"
	PayCrypto  PayMethod = "crypto"
	PayInvoice PayMethod = "invoice"
	PayCOD     PayMethod = "cod"
)

type Tier int

const (
	TierNone   Tier = 0
	TierSilver Tier = 1
	TierGold   Tier = 2
)

type Customer struct {
	ID      int
	Name    string
	Email   string
	Phone   string
	Region  string
	Blocked bool
	B2B     bool
}

type Order struct {
	ID           int
	CustomerID   int
	Kind         OrderKind
	ItemID       string
	Qty          int
	BasePrice    float64
	Region       string
	PayMethod    PayMethod
	DiscountCode string
	Express      bool
	Weight       float64
	UsePoints    bool
}

type Quote struct {
	Price    float64
	Currency string
	Err      string
}

type Result struct {
	OrderID   int
	Total     float64
	Status    string
	AuthID    string
	Carrier   string
	PickID    string
	InvoiceID string
	Notes     []string
}

// Зависимости. В легаси всё тянется через одну struct, потому что «так было раньше».
type Deps struct {
	CustomerRepo     CustomerRepo
	PriceService     PriceService
	TaxService       TaxService
	LoyaltyService   LoyaltyService
	DiscountService  DiscountService
	StockService     StockService
	WarehouseService WarehouseService
	FraudChecker     FraudChecker
	PaymentGateway   PaymentGateway
	CreditService    CreditService
	ShippingService  ShippingService
	Notifier         Notifier
	InvoiceService   InvoiceService
	EventBus         EventBus
	AuditLogger      AuditLogger
}

type CustomerRepo interface {
	Get(id int) (*Customer, error)
	IsBlacklisted(id int) (bool, error)
}

type PriceService interface {
	Quote(kind OrderKind, qty int, base float64) (Quote, error)
}

type TaxService interface {
	Calculate(region string, amount float64, digital bool) (float64, error)
}

type LoyaltyService interface {
	Tier(customerID int) (Tier, error)
	Redeem(customerID int, points int) (float64, error)
}

type DiscountService interface {
	Apply(code string, amount float64) (float64, error)
}

type StockService interface {
	Reserve(itemID string, qty int) (bool, error)
	Release(itemID string, qty int) error
}

type WarehouseService interface {
	Pick(itemID string, qty int) (string, error)
}

type FraudChecker interface {
	Score(customerID int, amount float64) (int, error)
}

type PaymentGateway interface {
	Authorize(method string, amount float64) (string, error)
	Capture(authID string, amount float64) error
	Refund(authID string, amount float64) error
}

type CreditService interface {
	Limit(customerID int) (float64, error)
	Charge(customerID int, amount float64) error
}

type ShippingService interface {
	Rate(region string, weight float64, express bool) (float64, error)
	PickCarrier(region string, weight float64) (string, error)
}

type Notifier interface {
	Email(to string, subject string) error
	SMS(to string, body string) error
}

type InvoiceService interface {
	Issue(customerID int, amount float64) (string, error)
}

type EventBus interface {
	Publish(topic string, payload any) error
}

type AuditLogger interface {
	Log(event string, details map[string]any) error
}
