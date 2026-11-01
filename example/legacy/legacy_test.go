package legacy_test

import (
	"errors"
	"testing"

	"github.com/0LuigiCode0/tshort"
	"github.com/0LuigiCode0/tshort/example/legacy"
	"github.com/0LuigiCode0/tshort/example/legacy/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestProcessOrder_Legacy — зеркало дерева ветвлений ProcessOrder.
//
// Дерево повторяет структуру метода (ветки в порядке, в котором они
// встречаются в коде):
//
//	ProcessOrder
//	├── валидация: nil / id / customer / qty / price / item
//	├── клиент: get err / not found / blocked / blacklisted / blacklist err
//	└── основной поток (IsBlacklisted → false)
//	    ├── цена: скидка (err / ok / clamp) + quote err + price error в поле
//	    ├── лояльность: silver (half / clamp) + gold (full / redeem err)
//	    ├── налоги: digital US +5 / digital 5% / physical 8% + 5 err-пропусканий
//	    ├── physical
//	    │   ├── reserve err
//	    │   ├── stock нет: backorder RU / backorder EU / no stock
//	    │   ├── stock ok: pick err (сестринский CASE)
//	    │   └── stock ok
//	    │       ├── express/card: large/EU
//	    │       ├── card: normal / fraud-held / score err / auth err / cap err
//	    │       ├── cod: RU (normal / region default / weight default) / US err
//	    │       ├── crypto: auth err / cap err
//	    │       ├── invoice: b2b (in limit / charge err / issue err)
//	    │       │           + b2b large (credit review / limit err) + non-b2b
//	    │       └── err: rate err / pick carrier err
//	    ├── digital: card (fraud60 / no phone) + crypto (email err / sms err /
//	    │   bus err / audit err) + cod (not physical) + bulk (b2b / non-b2b)
//	    │   + unknown pay method
//	    ├── subscription: crypto (first month / subscription event err)
//	    │   + invoice (over limit)
//	    └── kind: unknown kind
//
// Семантика либы: в НАЧАЛЕ каждого CASE — мутации входов и EXPECT-константы,
// одинаковые для ВСЕХ BREAK'ов этого CASE. BREAK — только дельта сценария.
//
// gomock-подводный камень: два EXPECT'а с одинаковыми аргументами — вызов
// матчит ПЕРВЫЙ зарегистрированный. Поэтому константа CASE и дельта BREAK
// одного сервиса не могут конфликтовать — такие сценарии живут в
// СЕСТРИНСКИХ CASE.
func TestProcessOrder_Legacy(t *testing.T) {
	// объявление моковых классов: один контроллер на все 15 мок-модулей —
	// просто переменная в скоупе теста, захваченная замыканиями CASE/BREAK
	var ctrl *gomock.Controller
	var mCust *mocks.MockCustomerRepo
	var mPrice *mocks.MockPriceService
	var mTax *mocks.MockTaxService
	var mLoyalty *mocks.MockLoyaltyService
	var mDisc *mocks.MockDiscountService
	var mStock *mocks.MockStockService
	var mWh *mocks.MockWarehouseService
	var mFraud *mocks.MockFraudChecker
	var mPay *mocks.MockPaymentGateway
	var mCredit *mocks.MockCreditService
	var mShip *mocks.MockShippingService
	var mNotif *mocks.MockNotifier
	var mInvoice *mocks.MockInvoiceService
	var mBus *mocks.MockEventBus
	var mAudit *mocks.MockAuditLogger
	var d *legacy.Deps
	var order *legacy.Order
	var custStd *legacy.Customer
	var custB2B *legacy.Customer
	var custNoPhone *legacy.Customer
	var custBlocked *legacy.Customer

	// ожидаемые результаты: созданы заранее, BREAK только на них ссылается
	var resLarge *legacy.Result
	var resNormal *legacy.Result
	var resFraudHeld *legacy.Result
	var resCodRU *legacy.Result
	var resInvoice *legacy.Result
	var resDigital60 *legacy.Result
	var resNoPhone *legacy.Result
	var resSubCrypto *legacy.Result
	var resOverLimit *legacy.Result
	var resDiscErr *legacy.Result
	var resDiscOk *legacy.Result
	var resDiscClamp *legacy.Result
	var resSilver *legacy.Result
	var resSilverClamp *legacy.Result
	var resGold *legacy.Result
	var resDigUS *legacy.Result
	var resDigUK *legacy.Result
	var resPhysUK *legacy.Result
	var resBackorderRU *legacy.Result
	var resBackorderEU *legacy.Result
	var resBulkB2B *legacy.Result
	var resLargeInvoice *legacy.Result
	var resSMSErr *legacy.Result

	// ожидаемые ошибки
	var errNil error
	var errBadID error
	var errBadCust error
	var errBadQty error
	var errNegPrice error
	var errNoItem error
	var errGet error
	var errNotFound error
	var errBlocked error
	var errBlacklisted error
	var errBlacklist error
	var errDisc error
	var errQuote error
	var errPriceField error
	var errRedeem error
	var errTaxDigEU error
	var errTaxDigUS error
	var errTaxRU error
	var errTaxEU error
	var errTaxUS error
	var errReserve error
	var errNoStock error
	var errPick error
	var errRate error
	var errPickCarrier error
	var errScore error
	var errAuthCard error
	var errCapCard error
	var errAuthCrypto error
	var errCapCrypto error
	var errInvLimit error
	var errCharge error
	var errIssue error
	var errInvB2B error
	var errCodKind error
	var errCodUS error
	var errPayMethod error
	var errBulk error
	var errUnknownKind error
	var errEmail error
	var errSMS error
	var errBus error
	var errSubEvent error
	var errAudit error

	tshort.Run(t, func(c *tshort.Collector[*legacy.Result]) {
		// один контроллер на все 15 мок-модулей: либа подменяет c.T на
		// первом проходе (без Cleanup → без авто-Finish), на перезапуске
		// цепочки c.T — подтест листа (с авто-Finish)
		ctrl = gomock.NewController(c.T)
		mCust = mocks.NewMockCustomerRepo(ctrl)
		mPrice = mocks.NewMockPriceService(ctrl)
		mTax = mocks.NewMockTaxService(ctrl)
		mLoyalty = mocks.NewMockLoyaltyService(ctrl)
		mDisc = mocks.NewMockDiscountService(ctrl)
		mStock = mocks.NewMockStockService(ctrl)
		mWh = mocks.NewMockWarehouseService(ctrl)
		mFraud = mocks.NewMockFraudChecker(ctrl)
		mPay = mocks.NewMockPaymentGateway(ctrl)
		mCredit = mocks.NewMockCreditService(ctrl)
		mShip = mocks.NewMockShippingService(ctrl)
		mNotif = mocks.NewMockNotifier(ctrl)
		mInvoice = mocks.NewMockInvoiceService(ctrl)
		mBus = mocks.NewMockEventBus(ctrl)
		mAudit = mocks.NewMockAuditLogger(ctrl)

		d = &legacy.Deps{
			CustomerRepo:     mCust,
			PriceService:     mPrice,
			TaxService:       mTax,
			LoyaltyService:   mLoyalty,
			DiscountService:  mDisc,
			StockService:     mStock,
			WarehouseService: mWh,
			FraudChecker:     mFraud,
			PaymentGateway:   mPay,
			CreditService:    mCredit,
			ShippingService:  mShip,
			Notifier:         mNotif,
			InvoiceService:   mInvoice,
			EventBus:         mBus,
			AuditLogger:      mAudit,
		}

		// входящие данные: клиенты (общие для всех CASE)
		custStd = &legacy.Customer{ID: 1, Name: "Ivan", Email: "ivan@example.com", Phone: "+79991234567", Region: "RU"}
		custB2B = &legacy.Customer{ID: 1, Name: "Corp", Email: "b2b@corp.example", Phone: "+79991234567", Region: "RU", B2B: true}
		custNoPhone = &legacy.Customer{ID: 1, Name: "Petr", Email: "petr@example.com", Phone: "", Region: "EU"}
		custBlocked = &legacy.Customer{ID: 1, Name: "Mira", Email: "mira@example.com", Region: "RU", Blocked: true}

		// ожидаемые результаты
		resLarge = &legacy.Result{OrderID: 1, Total: 60500, Status: "paid", AuthID: "auth-4", Carrier: "DHL", PickID: "pick-1", Notes: []string{"express"}}
		resNormal = &legacy.Result{OrderID: 1, Total: 12500, Status: "paid", AuthID: "auth-1", Carrier: "CDEK", PickID: "pick-1"}
		resFraudHeld = &legacy.Result{OrderID: 1, Total: 0, Status: "held", Carrier: "CDEK", PickID: "pick-1"}
		resCodRU = &legacy.Result{OrderID: 1, Total: 12500, Status: "paid", Carrier: "CDEK", PickID: "pick-1"}
		resInvoice = &legacy.Result{OrderID: 1, Total: 18000, Status: "paid", Carrier: "CDEK", PickID: "pick-1", InvoiceID: "inv-5"}
		resDigital60 = &legacy.Result{OrderID: 1, Total: 10000, Status: "paid", AuthID: "auth-7", Notes: []string{"single-license", "fraud-review"}}
		resNoPhone = &legacy.Result{OrderID: 1, Total: 11800, Status: "paid", AuthID: "auth-8", Notes: []string{"single-license"}}
		resSubCrypto = &legacy.Result{OrderID: 1, Total: 10350, Status: "paid", AuthID: "auth-9", Notes: []string{"first-month-10%", "b2b-subscription"}}
		resOverLimit = &legacy.Result{OrderID: 1, Total: 20700, Status: "credit_review", Notes: []string{"first-month-10%", "b2b-subscription"}}
		resDiscErr = &legacy.Result{OrderID: 1, Total: 10000, Status: "paid", AuthID: "auth-12"}
		resDiscOk = &legacy.Result{OrderID: 1, Total: 8000, Status: "paid", AuthID: "auth-13", Notes: []string{"discount:SALE10"}}
		resDiscClamp = &legacy.Result{OrderID: 1, Total: 0, Status: "paid", AuthID: "auth-14", Notes: []string{"discount:SALE10", "small-order"}}
		resSilver = &legacy.Result{OrderID: 1, Total: 9000, Status: "paid", AuthID: "auth-15", Notes: []string{"loyalty:silver"}}
		resSilverClamp = &legacy.Result{OrderID: 1, Total: 0, Status: "paid", AuthID: "auth-16", Notes: []string{"loyalty:silver", "small-order"}}
		resGold = &legacy.Result{OrderID: 1, Total: 7000, Status: "paid", AuthID: "auth-17", Notes: []string{"loyalty:gold"}}
		resDigUS = &legacy.Result{OrderID: 1, Total: 10805, Status: "paid", AuthID: "auth-18"}
		resDigUK = &legacy.Result{OrderID: 1, Total: 10500, Status: "paid", AuthID: "auth-19"}
		resPhysUK = &legacy.Result{OrderID: 1, Total: 11700, Status: "paid", AuthID: "auth-20", Carrier: "UPS", PickID: "pick-1"}
		resBackorderRU = &legacy.Result{OrderID: 1, Total: 12500, Status: "backorder", Carrier: "CDEK"}
		resBackorderEU = &legacy.Result{OrderID: 1, Total: 12800, Status: "backorder", AuthID: "auth-10", Carrier: "DHL"}
		resBulkB2B = &legacy.Result{OrderID: 1, Total: 10000, Status: "paid", AuthID: "auth-22", Notes: []string{"bulk-digital"}}
		resLargeInvoice = &legacy.Result{OrderID: 1, Total: 69500, Status: "credit_review", Carrier: "CDEK", PickID: "pick-1"}
		resSMSErr = &legacy.Result{OrderID: 1, Total: 10000, Status: "paid", AuthID: "auth-21"}

		// ожидаемые ошибки
		errNil = errors.New("legacy: order is nil")
		errBadID = errors.New("legacy: bad order id")
		errBadCust = errors.New("legacy: bad customer id")
		errBadQty = errors.New("legacy: bad qty")
		errNegPrice = errors.New("legacy: negative base price")
		errNoItem = errors.New("legacy: physical order without item")
		errGet = errors.New("client: get failed")
		errNotFound = errors.New("legacy: customer not found")
		errBlocked = errors.New("legacy: customer blocked")
		errBlacklisted = errors.New("legacy: customer blacklisted")
		errBlacklist = errors.New("client: blacklist failed")
		errDisc = errors.New("discount: apply failed")
		errQuote = errors.New("price: quote failed")
		errPriceField = errors.New("legacy: price error: boom")
		errRedeem = errors.New("loyalty: redeem failed")
		errTaxDigEU = errors.New("tax: digital EU failed")
		errTaxDigUS = errors.New("tax: digital US failed")
		errTaxRU = errors.New("tax: RU failed")
		errTaxEU = errors.New("tax: EU failed")
		errTaxUS = errors.New("tax: US failed")
		errReserve = errors.New("stock: reserve failed")
		errNoStock = errors.New("legacy: no stock")
		errPick = errors.New("warehouse: pick failed")
		errRate = errors.New("shipping: rate failed")
		errPickCarrier = errors.New("shipping: pick carrier failed")
		errScore = errors.New("fraud: score failed")
		errAuthCard = errors.New("payment: card authorize failed")
		errCapCard = errors.New("payment: card capture failed")
		errAuthCrypto = errors.New("payment: crypto authorize failed")
		errCapCrypto = errors.New("payment: crypto capture failed")
		errInvLimit = errors.New("credit: limit failed")
		errCharge = errors.New("credit: charge failed")
		errIssue = errors.New("invoice: issue failed")
		errInvB2B = errors.New("legacy: invoice requires b2b")
		errCodKind = errors.New("legacy: cod only for physical")
		errCodUS = errors.New("legacy: cod only in RU")
		errPayMethod = errors.New("legacy: unknown pay method giftcard")
		errBulk = errors.New("legacy: bulk digital requires b2b")
		errUnknownKind = errors.New("legacy: unknown kind 99")
		errEmail = errors.New("notify: email failed")
		errSMS = errors.New("notify: sms failed")
		errBus = errors.New("events: bus failed")
		errSubEvent = errors.New("events: subscription failed")
		errAudit = errors.New("audit: log failed")

		// ================= валидация =================
		// SUT: 6 guards + region по умолчанию. Каждый BREAK ломает ровно
		// одно поле; тело CASE каждый раз создаёт свежий валидный заказ,
		// поэтому утечки между подтестами невозможны.
		c.CASE(func(c *tshort.Collector[*legacy.Result]) {
			order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindDigital, ItemID: "item-1", Qty: 1, BasePrice: 10, Region: "RU"}

			c.BREAK("validation/nil order", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errNil
				order = nil
			})
			c.BREAK("validation/bad id", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errBadID
				order.ID = 0
			})
			c.BREAK("validation/bad customer", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errBadCust
				order.CustomerID = 0
			})
			c.BREAK("validation/bad qty", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errBadQty
				order.Qty = 0
			})
			c.BREAK("validation/negative price", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errNegPrice
				order.BasePrice = -10
			})
			c.BREAK("validation/physical without item", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errNoItem
				order.Kind = legacy.KindPhysical
				order.ItemID = ""
			})
		})

		// ================= клиент =================
		// SUT: Get → nil? → Blocked? → IsBlacklisted → true? → Tier.
		// Все BREAK'и возвращаются до Tier, поэтому Tier здесь не нужен.
		c.CASE(func(c *tshort.Collector[*legacy.Result]) {
			order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindDigital, ItemID: "item-1", Qty: 1, BasePrice: 10, Region: "RU"}

			c.BREAK("client/get err", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errGet
				mCust.EXPECT().Get(order.CustomerID).Return(nil, errGet)
			})
			c.BREAK("client/not found", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errNotFound
				mCust.EXPECT().Get(order.CustomerID).Return(nil, nil)
				mAudit.EXPECT().Log("customer.notfound", gomock.Any()).Return(nil)
			})
			c.BREAK("client/blocked", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errBlocked
				mCust.EXPECT().Get(order.CustomerID).Return(custBlocked, nil)
				mAudit.EXPECT().Log("customer.blocked", gomock.Any()).Return(nil)
			})
			c.BREAK("client/blacklisted", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errBlacklisted
				mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
				mCust.EXPECT().IsBlacklisted(order.CustomerID).Return(true, nil)
				mNotif.EXPECT().Email(custStd.Email, "Order rejected: blacklist").Return(nil)
			})
			c.BREAK("client/blacklist err", func(c *tshort.Case[*legacy.Result]) {
				c.ExpectErr = errBlacklist
				mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
				mCust.EXPECT().IsBlacklisted(order.CustomerID).Return(false, errBlacklist)
			})
		})

		// ================= основной поток =================
		// всё, что доживает до ценообразования: клиент найден, не blocked,
		// не в чёрном списке. IsBlacklisted → false — константа всех этих
		// leaf'ей (blacklisted-сценарии выше, в CASE «клиент»).
		c.CASE(func(c *tshort.Collector[*legacy.Result]) {
			mCust.EXPECT().IsBlacklisted(order.CustomerID).Return(false, nil)

			// ================= цена =================
			// SUT: Quote → err? → q.Err? → скидка → лояльность.
			// digital/RU/crypto: самый дешёвый полный поток (нет склада и
			// доставки), поэтому скидочные сценарии живут здесь.
			c.CASE(func(c *tshort.Collector[*legacy.Result]) {
				order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindDigital, ItemID: "item-1", Qty: 2, BasePrice: 5000, Region: "RU", PayMethod: legacy.PayCrypto}
				mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
				mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)

				// ---------- скидка по промокоду ----------
				// SUT: Apply → err? (не фатально, лог) → amount -= disc → clamp
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					order.DiscountCode = "SALE10"
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)

					// ошибка скидки не фатальна: лог и дальше по потоку
					c.BREAK("price/discount/err non-fatal", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resDiscErr
						mDisc.EXPECT().Apply(order.DiscountCode, 10000.0).Return(0.0, errDisc)
						mAudit.EXPECT().Log("discount.error", gomock.Any()).Return(nil)
						mPay.EXPECT().Authorize("crypto", 10000.0).Return("auth-12", nil)
						mPay.EXPECT().Capture("auth-12", 10000.0).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})

					// скидка применена: amount -= disc
					c.BREAK("price/discount/ok", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resDiscOk
						mDisc.EXPECT().Apply(order.DiscountCode, 10000.0).Return(2000.0, nil)
						mPay.EXPECT().Authorize("crypto", 8000.0).Return("auth-13", nil)
						mPay.EXPECT().Capture("auth-13", 8000.0).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})

					// legacy: не уходить в минус → amount = 0 → small-order
					c.BREAK("price/discount/clamp to zero", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resDiscClamp
						mDisc.EXPECT().Apply(order.DiscountCode, 10000.0).Return(15000.0, nil)
						mPay.EXPECT().Authorize("crypto", 0.0).Return("auth-14", nil)
						mPay.EXPECT().Capture("auth-14", 0.0).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})
				})

				c.BREAK("price/quote err", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errQuote
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{}, errQuote)
				})

				// legacy: ценовой сервис возвращает ошибку в поле, а не в error
				c.BREAK("price/price error field", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errPriceField
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Err: "boom"}, nil)
				})
			})

			// ================= лояльность =================
			// SUT: UsePoints && tier != None → gold (полные) / silver (половина).
			// Tier — константа СЕСТРИНСКИХ CASE (silver/gold): у листа gold
			// не может быть silver-константа над ним (первый EXPECT побеждает).
			c.CASE(func(c *tshort.Collector[*legacy.Result]) {
				// Qty в (1, 5]: не single-license и не bulk (bulk требует b2b)
				order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindDigital, ItemID: "item-1", Qty: 2, BasePrice: 1000, Region: "RU", PayMethod: legacy.PayCrypto}
				order.UsePoints = true
				mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
				mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)

				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)

					// legacy: silver получает половину (так договорились в 2019)
					c.BREAK("loyalty/silver/half of points", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resSilver
						mLoyalty.EXPECT().Redeem(order.CustomerID, order.Qty*5).Return(2000.0, nil)
						mPay.EXPECT().Authorize("crypto", 9000.0).Return("auth-15", nil)
						mPay.EXPECT().Capture("auth-15", 9000.0).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})

					c.BREAK("loyalty/silver/clamp to zero", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resSilverClamp
						mLoyalty.EXPECT().Redeem(order.CustomerID, order.Qty*5).Return(20000.0, nil)
						mPay.EXPECT().Authorize("crypto", 0.0).Return("auth-16", nil)
						mPay.EXPECT().Capture("auth-16", 0.0).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})
				})

				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierGold, nil)

					c.BREAK("loyalty/gold/all points", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resGold
						mLoyalty.EXPECT().Redeem(order.CustomerID, order.Qty*10).Return(3000.0, nil)
						mPay.EXPECT().Authorize("crypto", 7000.0).Return("auth-17", nil)
						mPay.EXPECT().Capture("auth-17", 7000.0).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})

					c.BREAK("loyalty/gold/redeem err", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectErr = errRedeem
						mLoyalty.EXPECT().Redeem(order.CustomerID, order.Qty*10).Return(0.0, errRedeem)
					})
				})
			})

			// ================= налоги =================
			// SUT: самый ветвистый кусок — правила по регионам и типу товара.
			// Листы, доживающие до оплаты: digital — crypto (дешёвый поток).
			c.CASE(func(c *tshort.Collector[*legacy.Result]) {
				order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindDigital, ItemID: "item-1", Qty: 2, BasePrice: 5000, Region: "RU", PayMethod: legacy.PayCrypto}
				mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
				mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)
				// AnyTimes: physical-листы ниже рвут Quote(2,2,5000) своим
				// per-leaf Quote(1,10,1000) — константа у них не расходуется.
				mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil).AnyTimes()

				// legacy: фиксированная доплата за state tax US
				c.BREAK("tax/digital US +5", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectOk = resDigUS
					order.Region = "US"
					mTax.EXPECT().Calculate("US", 10000.0, true).Return(800.0, nil)
					mPay.EXPECT().Authorize("crypto", 10805.0).Return("auth-18", nil)
					mPay.EXPECT().Capture("auth-18", 10805.0).Return(nil)
					mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
					mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
					mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
					mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
				})

				// legacy: остальные регионы — плоские 5% «из экселя»
				c.BREAK("tax/digital UK 5%", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectOk = resDigUK
					order.Region = "UK"
					mPay.EXPECT().Authorize("crypto", 10500.0).Return("auth-19", nil)
					mPay.EXPECT().Capture("auth-19", 10500.0).Return(nil)
					mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
					mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
					mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
					mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
				})

				// legacy: physical в «не названном» регионе — 8%
				c.BREAK("tax/physical UK 8%", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectOk = resPhysUK
					order.Kind = legacy.KindPhysical
					order.Qty = 10
					order.BasePrice = 1000
					order.Weight = 5
					order.Region = "UK"
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mStock.EXPECT().Reserve(order.ItemID, order.Qty).Return(true, nil)
					mWh.EXPECT().Pick(order.ItemID, order.Qty).Return("pick-1", nil)
					mShip.EXPECT().Rate("UK", 5.0, false).Return(900.0, nil)
					mShip.EXPECT().PickCarrier("UK", 5.0).Return("UPS", nil)
					mPay.EXPECT().Authorize("crypto", 11700.0).Return("auth-20", nil)
					mPay.EXPECT().Capture("auth-20", 11700.0).Return(nil)
					mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
					mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
					mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
					mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
				})

				// err-пропускания tax-секции
				c.BREAK("tax/digital EU err", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errTaxDigEU
					order.Region = "EU"
					mTax.EXPECT().Calculate("EU", 10000.0, true).Return(0.0, errTaxDigEU)
				})
				c.BREAK("tax/digital US err", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errTaxDigUS
					order.Region = "US"
					mTax.EXPECT().Calculate("US", 10000.0, true).Return(0.0, errTaxDigUS)
				})
				c.BREAK("tax/physical RU err", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errTaxRU
					order.Kind = legacy.KindPhysical
					order.Qty = 10
					order.BasePrice = 1000
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mTax.EXPECT().Calculate("RU", 10000.0, false).Return(0.0, errTaxRU)
				})
				c.BREAK("tax/physical EU err", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errTaxEU
					order.Kind = legacy.KindPhysical
					order.Qty = 10
					order.BasePrice = 1000
					order.Region = "EU"
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mTax.EXPECT().Calculate("EU", 10000.0, false).Return(0.0, errTaxEU)
				})
				c.BREAK("tax/physical US err", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errTaxUS
					order.Kind = legacy.KindPhysical
					order.Qty = 10
					order.BasePrice = 1000
					order.Region = "US"
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mTax.EXPECT().Calculate("US", 10000.0, false).Return(0.0, errTaxUS)
				})
			})

			// ================= kind == physical =================
			// SUT: Reserve → (!ok: backorder / no stock) → (ok: Pick) →
			// Rate → express? → PickCarrier → оплата.
			c.CASE(func(c *tshort.Collector[*legacy.Result]) {
				order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindPhysical, ItemID: "item-1", Qty: 10, BasePrice: 1000, Region: "RU", Weight: 5}

				// ---------- склад: reserve упал ----------
				c.BREAK("physical/stock/reserve err", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errReserve
					mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
					mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mTax.EXPECT().Calculate("RU", 10000.0, false).Return(2000.0, nil)
					mStock.EXPECT().Reserve(order.ItemID, order.Qty).Return(false, errReserve)
				})

				// ---------- склад: не хватает ----------
				// SUT: RU/EU → backorder и в оплату; остальные → no stock.
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
					mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mStock.EXPECT().Reserve(order.ItemID, order.Qty).Return(false, nil)

					// RU: backorder, деньги соберёт курьер (cod)
					c.BREAK("physical/stock/backorder RU", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resBackorderRU
						order.PayMethod = legacy.PayCOD
						mTax.EXPECT().Calculate("RU", 10000.0, false).Return(2000.0, nil)
						mBus.EXPECT().Publish("stock.backorder", gomock.Any()).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order on backorder").Return(nil)
						mShip.EXPECT().Rate("RU", 5.0, false).Return(500.0, nil)
						mShip.EXPECT().PickCarrier("RU", 5.0).Return("CDEK", nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})

					// EU: backorder, crypto (cod в EU не живёт)
					c.BREAK("physical/stock/backorder EU", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resBackorderEU
						order.PayMethod = legacy.PayCrypto
						order.Region = "EU"
						mTax.EXPECT().Calculate("EU", 10000.0, false).Return(1800.0, nil)
						mBus.EXPECT().Publish("stock.backorder", gomock.Any()).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order on backorder").Return(nil)
						mShip.EXPECT().Rate("EU", 5.0, false).Return(800.0, nil)
						mShip.EXPECT().PickCarrier("EU", 5.0).Return("DHL", nil)
						mPay.EXPECT().Authorize("crypto", 12800.0).Return("auth-10", nil)
						mPay.EXPECT().Capture("auth-10", 12800.0).Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})

					// остальные регионы отменяем, логистика не берёт
					c.BREAK("physical/stock/no stock", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectErr = errNoStock
						order.Region = "US"
						mTax.EXPECT().Calculate("US", 10000.0, false).Return(800.0, nil)
					})
				})

				// ---------- склад: ok ----------
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					mStock.EXPECT().Reserve(order.ItemID, order.Qty).Return(true, nil)
					mWh.EXPECT().Pick(order.ItemID, order.Qty).Return("pick-1", nil)
					mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)

					// ---------- express: тариф + 50% ----------
					c.CASE(func(c *tshort.Collector[*legacy.Result]) {
						order.Express = true

						c.CASE(func(c *tshort.Collector[*legacy.Result]) {
							order.PayMethod = legacy.PayCard
							order.Region = "EU"
							mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)

							// крупный express-заказ: audit order.large,
							// score < 50 → обычный платёж
							c.BREAK("physical/card/express/EU/large", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectOk = resLarge
								mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 50000, Currency: "EUR"}, nil)
								mTax.EXPECT().Calculate("EU", 50000.0, false).Return(8000.0, nil)
								mShip.EXPECT().Rate("EU", 5.0, true).Return(1000.0, nil)
								mShip.EXPECT().PickCarrier("EU", 5.0).Return("DHL", nil)
								mAudit.EXPECT().Log("order.large", gomock.Any()).Return(nil)
								mFraud.EXPECT().Score(order.CustomerID, 60500.0).Return(10, nil)
								mPay.EXPECT().Authorize("card", 60500.0).Return("auth-4", nil)
								mPay.EXPECT().Capture("auth-4", 60500.0).Return(nil)
								mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
								mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
								mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
								mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
							})
						})
					})

					// ---------- payMethod == card ----------
					c.CASE(func(c *tshort.Collector[*legacy.Result]) {
						order.PayMethod = legacy.PayCard
						mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)

						// RU: стандартная цена, все ветки fraud/gateway
						c.CASE(func(c *tshort.Collector[*legacy.Result]) {
							order.Region = "RU"
							mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
							mTax.EXPECT().Calculate("RU", 10000.0, false).Return(2000.0, nil)
							mShip.EXPECT().Rate("RU", 5.0, false).Return(500.0, nil)
							mShip.EXPECT().PickCarrier("RU", 5.0).Return("CDEK", nil)

							// score < 50: ok-путь — authorize → capture → хвост
							c.BREAK("physical/card/RU/normal", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectOk = resNormal
								mFraud.EXPECT().Score(order.CustomerID, 12500.0).Return(10, nil)
								mPay.EXPECT().Authorize("card", 12500.0).Return("auth-1", nil)
								mPay.EXPECT().Capture("auth-1", 12500.0).Return(nil)
								mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
								mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
								mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
								mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
							})

							// score >= 80: держим на ручном осмотре, хвоста нет
							c.BREAK("physical/card/RU/fraud-held", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectOk = resFraudHeld
								// legacy-фича: при hold Total не проставляется
								mFraud.EXPECT().Score(order.CustomerID, 12500.0).Return(90, nil)
								mNotif.EXPECT().Email(custStd.Email, "Order held: fraud check").Return(nil)
							})

							c.BREAK("physical/card/RU/score err", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectErr = errScore
								mFraud.EXPECT().Score(order.CustomerID, 12500.0).Return(0, errScore)
							})

							// authorize упал: откат резерва (physical)
							c.BREAK("physical/card/RU/authorize err", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectErr = errAuthCard
								mFraud.EXPECT().Score(order.CustomerID, 12500.0).Return(10, nil)
								mPay.EXPECT().Authorize("card", 12500.0).Return("", errAuthCard)
								mStock.EXPECT().Release(order.ItemID, order.Qty).Return(nil)
							})

							// capture упал: откат авторизации и склада
							c.BREAK("physical/card/RU/capture err", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectErr = errCapCard
								mFraud.EXPECT().Score(order.CustomerID, 12500.0).Return(10, nil)
								mPay.EXPECT().Authorize("card", 12500.0).Return("auth-1", nil)
								mPay.EXPECT().Capture("auth-1", 12500.0).Return(errCapCard)
								mPay.EXPECT().Refund("auth-1", 12500.0).Return(nil)
								mStock.EXPECT().Release(order.ItemID, order.Qty).Return(nil)
							})
						})
					})

					// ---------- payMethod == cod ----------
					c.CASE(func(c *tshort.Collector[*legacy.Result]) {
						order.PayMethod = legacy.PayCOD
						mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)

						// RU: деньги соберёт курьер, gateway не трогаем.
						// Три BREAK'а с одним и тем же потоком: normal,
						// region="" (legacy-дефолт → RU), weight=0 (дефолт
						// → qty*0.5 = 5.0, тот же тариф).
						c.CASE(func(c *tshort.Collector[*legacy.Result]) {
							order.Region = "RU"
							mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
							mTax.EXPECT().Calculate("RU", 10000.0, false).Return(2000.0, nil)
							mShip.EXPECT().Rate("RU", 5.0, false).Return(500.0, nil)
							mShip.EXPECT().PickCarrier("RU", 5.0).Return("CDEK", nil)
							mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
							mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
							mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
							mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)

							c.BREAK("physical/cod/RU", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectOk = resCodRU
							})

							// legacy: регион по умолчанию, старый фронт не слал поле
							c.BREAK("physical/cod/RU/region default", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectOk = resCodRU
								order.Region = ""
							})

							// legacy: вес по умолчанию, старый фронт не слал
							c.BREAK("physical/cod/RU/weight default", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectOk = resCodRU
								order.Weight = 0
							})
						})

						// region != RU: "legacy: cod only in RU"
						c.BREAK("physical/cod/US/err", func(c *tshort.Case[*legacy.Result]) {
							c.ExpectErr = errCodUS
							order.Region = "US"
							mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "USD"}, nil)
							mTax.EXPECT().Calculate("US", 10000.0, false).Return(800.0, nil)
							mShip.EXPECT().Rate("US", 5.0, false).Return(1200.0, nil)
							mShip.EXPECT().PickCarrier("US", 5.0).Return("USPS", nil)
						})
					})

					// ---------- payMethod == crypto ----------
					// legacy: crypto не ходит в fraud.
					// RU-константы: все crypto-сценарии physical/RU.
					c.CASE(func(c *tshort.Collector[*legacy.Result]) {
						order.PayMethod = legacy.PayCrypto
						order.Region = "RU"
						mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
						mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
						mTax.EXPECT().Calculate("RU", 10000.0, false).Return(2000.0, nil)
						mShip.EXPECT().Rate("RU", 5.0, false).Return(500.0, nil)
						mShip.EXPECT().PickCarrier("RU", 5.0).Return("CDEK", nil)

						c.BREAK("physical/crypto/RU/authorize err", func(c *tshort.Case[*legacy.Result]) {
							c.ExpectErr = errAuthCrypto
							mPay.EXPECT().Authorize("crypto", 12500.0).Return("", errAuthCrypto)
							mStock.EXPECT().Release(order.ItemID, order.Qty).Return(nil)
						})

						c.BREAK("physical/crypto/RU/capture err", func(c *tshort.Case[*legacy.Result]) {
							c.ExpectErr = errCapCrypto
							mPay.EXPECT().Authorize("crypto", 12500.0).Return("auth-11", nil)
							mPay.EXPECT().Capture("auth-11", 12500.0).Return(errCapCrypto)
							mPay.EXPECT().Refund("auth-11", 12500.0).Return(nil)
							mStock.EXPECT().Release(order.ItemID, order.Qty).Return(nil)
						})
					})

					// ---------- payMethod == invoice ----------
					c.CASE(func(c *tshort.Collector[*legacy.Result]) {
						order.PayMethod = legacy.PayInvoice
						order.Region = "RU"

						// ---------- b2b ----------
						c.CASE(func(c *tshort.Collector[*legacy.Result]) {
							mCust.EXPECT().Get(order.CustomerID).Return(custB2B, nil)

							// ---------- в пределах лимита ----------
							c.CASE(func(c *tshort.Collector[*legacy.Result]) {
								mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 15000, Currency: "RUB"}, nil)
								mTax.EXPECT().Calculate("RU", 15000.0, false).Return(2500.0, nil)
								mShip.EXPECT().Rate("RU", 5.0, false).Return(500.0, nil)
								mShip.EXPECT().PickCarrier("RU", 5.0).Return("CDEK", nil)
								mCredit.EXPECT().Limit(order.CustomerID).Return(50000.0, nil)

								c.BREAK("physical/invoice/RU/b2b", func(c *tshort.Case[*legacy.Result]) {
									c.ExpectOk = resInvoice
									mCredit.EXPECT().Charge(order.CustomerID, 18000.0).Return(nil)
									mInvoice.EXPECT().Issue(order.CustomerID, 18000.0).Return("inv-5", nil)
									mNotif.EXPECT().Email(custB2B.Email, "Order confirmed").Return(nil)
									mNotif.EXPECT().SMS(custB2B.Phone, "Order 1 confirmed").Return(nil)
									mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
									mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
								})

								c.BREAK("physical/invoice/RU/b2b/charge err", func(c *tshort.Case[*legacy.Result]) {
									c.ExpectErr = errCharge
									mCredit.EXPECT().Charge(order.CustomerID, 18000.0).Return(errCharge)
								})

								c.BREAK("physical/invoice/RU/b2b/issue err", func(c *tshort.Case[*legacy.Result]) {
									c.ExpectErr = errIssue
									mCredit.EXPECT().Charge(order.CustomerID, 18000.0).Return(nil)
									mInvoice.EXPECT().Issue(order.CustomerID, 18000.0).Return("", errIssue)
								})
							})

							// ---------- крупный: порог 50k + invoice ----------
							// SUT: audit order.large → Limit → amount > lim →
							// credit_review (и в порогах, и в оплате: Limit
							// и письмо идут ДВАЖДЫ).
							c.CASE(func(c *tshort.Collector[*legacy.Result]) {
								mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 60000, Currency: "RUB"}, nil)
								mTax.EXPECT().Calculate("RU", 60000.0, false).Return(9000.0, nil)
								mShip.EXPECT().Rate("RU", 5.0, false).Return(500.0, nil)
								mShip.EXPECT().PickCarrier("RU", 5.0).Return("CDEK", nil)

								c.BREAK("physical/invoice/RU/b2b/large/credit review", func(c *tshort.Case[*legacy.Result]) {
									c.ExpectOk = resLargeInvoice
									mAudit.EXPECT().Log("order.large", gomock.Any()).Return(nil)
									mCredit.EXPECT().Limit(order.CustomerID).Return(50000.0, nil)
									mNotif.EXPECT().Email(custB2B.Email, "Order on credit review").Return(nil)
									mCredit.EXPECT().Limit(order.CustomerID).Return(50000.0, nil)
									mNotif.EXPECT().Email(custB2B.Email, "Order on credit review").Return(nil)
									mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
									mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
								})

								c.BREAK("physical/invoice/RU/b2b/large/limit err", func(c *tshort.Case[*legacy.Result]) {
									c.ExpectErr = errInvLimit
									mAudit.EXPECT().Log("order.large", gomock.Any()).Return(nil)
									mCredit.EXPECT().Limit(order.CustomerID).Return(0.0, errInvLimit)
								})
							})
						})

						// ---------- non-b2b: "legacy: invoice requires b2b" ----------
						c.CASE(func(c *tshort.Collector[*legacy.Result]) {
							mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
							mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 15000, Currency: "RUB"}, nil)
							mTax.EXPECT().Calculate("RU", 15000.0, false).Return(2500.0, nil)
							mShip.EXPECT().Rate("RU", 5.0, false).Return(500.0, nil)
							mShip.EXPECT().PickCarrier("RU", 5.0).Return("CDEK", nil)

							c.BREAK("physical/invoice/RU/non-b2b", func(c *tshort.Case[*legacy.Result]) {
								c.ExpectErr = errInvB2B
							})
						})
					})

					// ---------- секция доставки: err-пропускания ----------
					// Rate / PickCarrier падают до оплаты — gateway не нужен.
					c.CASE(func(c *tshort.Collector[*legacy.Result]) {
						mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
						mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
						mTax.EXPECT().Calculate("RU", 10000.0, false).Return(2000.0, nil)

						c.BREAK("physical/rate err", func(c *tshort.Case[*legacy.Result]) {
							c.ExpectErr = errRate
							mShip.EXPECT().Rate("RU", 5.0, false).Return(0.0, errRate)
						})

						c.BREAK("physical/pick carrier err", func(c *tshort.Case[*legacy.Result]) {
							c.ExpectErr = errPickCarrier
							mShip.EXPECT().Rate("RU", 5.0, false).Return(500.0, nil)
							mShip.EXPECT().PickCarrier("RU", 5.0).Return("", errPickCarrier)
						})
					})
				})

				// ---------- pick упал: откат резерва ----------
				// Сестра «склад: ok»: над листом не должно стоять Pick-константа
				// "pick-1" из «склад: ok» — первый зарегистрированный EXPECT
				// побеждает.
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
					mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mTax.EXPECT().Calculate("RU", 10000.0, false).Return(2000.0, nil)
					mStock.EXPECT().Reserve(order.ItemID, order.Qty).Return(true, nil)
					mWh.EXPECT().Pick(order.ItemID, order.Qty).Return("", errPick)
					mStock.EXPECT().Release(order.ItemID, order.Qty).Return(nil)

					c.BREAK("physical/stock/pick err", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectErr = errPick
					})
				})
			})

			// ================= kind == digital =================
			// SUT: без склада и доставки: qty > 5 → b2b?, qty == 1 → note.
			c.CASE(func(c *tshort.Collector[*legacy.Result]) {
				order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindDigital, ItemID: "item-1", Qty: 1, BasePrice: 1000, Region: "RU"}
				mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)

				// ---------- payMethod == card ----------
				// у двух leaf'ов разные клиенты — Get пер-leaf
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					order.PayMethod = legacy.PayCard

					// RU: digital без НДС; score 60 → note fraud-review
					c.BREAK("digital/card/RU/fraud60", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resDigital60
						mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
						mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
						mFraud.EXPECT().Score(order.CustomerID, 10000.0).Return(60, nil)
						mPay.EXPECT().Authorize("card", 10000.0).Return("auth-7", nil)
						mPay.EXPECT().Capture("auth-7", 10000.0).Return(nil)
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})

					// EU: tax от сервиса; нет телефона → SMS не уходит
					c.BREAK("digital/card/EU/no-phone", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resNoPhone
						order.Region = "EU"
						mCust.EXPECT().Get(order.CustomerID).Return(custNoPhone, nil)
						mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "EUR"}, nil)
						mTax.EXPECT().Calculate("EU", 10000.0, true).Return(1800.0, nil)
						mFraud.EXPECT().Score(order.CustomerID, 11800.0).Return(10, nil)
						mPay.EXPECT().Authorize("card", 11800.0).Return("auth-8", nil)
						mPay.EXPECT().Capture("auth-8", 11800.0).Return(nil)
						mNotif.EXPECT().Email(custNoPhone.Email, "Order confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})
				})

				// ---------- payMethod == crypto: хвост (notify/bus/audit) ----------
				// crypto без fraud: самый дешёвый digital-поток.
				// Каждый BREAK рвёт свой хвостовой вызов.
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					order.PayMethod = legacy.PayCrypto
					order.Qty = 2
					mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mPay.EXPECT().Authorize("crypto", 10000.0).Return("auth-21", nil)
					mPay.EXPECT().Capture("auth-21", 10000.0).Return(nil)

					c.BREAK("digital/crypto/RU/email err", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectErr = errEmail
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(errEmail)
					})

					// legacy: SMS упал — не критично, просто лог
					c.BREAK("digital/crypto/RU/sms err non-fatal", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resSMSErr
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(errSMS)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
					})

					c.BREAK("digital/crypto/RU/bus err", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectErr = errBus
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(errBus)
					})

					c.BREAK("digital/crypto/RU/audit err", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectErr = errAudit
						mNotif.EXPECT().Email(custStd.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custStd.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
						mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(errAudit)
					})
				})

				// ---------- payMethod == cod: только physical ----------
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					order.PayMethod = legacy.PayCOD
					mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)

					c.BREAK("digital/cod/not physical", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectErr = errCodKind
						mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					})
				})

				// ---------- qty > 5: оптовый digital только для B2B ----------
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					order.PayMethod = legacy.PayCrypto
					order.Qty = 10
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)

					c.CASE(func(c *tshort.Collector[*legacy.Result]) {
						mCust.EXPECT().Get(order.CustomerID).Return(custB2B, nil)

						c.BREAK("digital/bulk/b2b", func(c *tshort.Case[*legacy.Result]) {
							c.ExpectOk = resBulkB2B
							mPay.EXPECT().Authorize("crypto", 10000.0).Return("auth-22", nil)
							mPay.EXPECT().Capture("auth-22", 10000.0).Return(nil)
							mNotif.EXPECT().Email(custB2B.Email, "Order confirmed").Return(nil)
							mNotif.EXPECT().SMS(custB2B.Phone, "Order 1 confirmed").Return(nil)
							mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
							mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil)
						})
					})

					c.CASE(func(c *tshort.Collector[*legacy.Result]) {
						mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)

						c.BREAK("digital/bulk/non-b2b", func(c *tshort.Case[*legacy.Result]) {
							c.ExpectErr = errBulk
						})
					})
				})

				// ---------- pay method не из констант ----------
				c.BREAK("digital/unknown pay method", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errPayMethod
					order.PayMethod = "giftcard"
					mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
				})
			})

			// ================= kind == subscription =================
			// SUT: первый месяц -10% (без промокода), b2b → note.
			// Все подписные листы — b2b-клиент.
			c.CASE(func(c *tshort.Collector[*legacy.Result]) {
				order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindSubscription, ItemID: "item-1", Qty: 10, BasePrice: 1000, Region: "RU"}
				mCust.EXPECT().Get(order.CustomerID).Return(custB2B, nil)
				mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)
				mBus.EXPECT().Publish("order.processed", gomock.Any()).Return(nil)
				// AnyTimes: лист «subscription event err» умирает до аудита
				// (SUT return на ошибке subscription.created) — Log не расходуется.
				mAudit.EXPECT().Log("order.processed", gomock.Any()).Return(nil).AnyTimes()

				// ---------- payMethod == crypto ----------
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					order.PayMethod = legacy.PayCrypto
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mTax.EXPECT().Calculate("RU", 10000.0, false).Return(1500.0, nil)
					mPay.EXPECT().Authorize("crypto", 10350.0).Return("auth-9", nil)
					mPay.EXPECT().Capture("auth-9", 10350.0).Return(nil)

					c.BREAK("subscription/crypto/RU/b2b", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resSubCrypto
						mNotif.EXPECT().Email(custB2B.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custB2B.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("subscription.created", gomock.Any()).Return(nil)
					})

					c.BREAK("subscription/crypto/RU/b2b/subscription event err", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectErr = errSubEvent
						mNotif.EXPECT().Email(custB2B.Email, "Order confirmed").Return(nil)
						mNotif.EXPECT().SMS(custB2B.Phone, "Order 1 confirmed").Return(nil)
						mBus.EXPECT().Publish("subscription.created", gomock.Any()).Return(errSubEvent)
					})
				})

				// ---------- payMethod == invoice ----------
				// SUT: amount > limit → credit_review (письмо в ветке).
				c.CASE(func(c *tshort.Collector[*legacy.Result]) {
					order.PayMethod = legacy.PayInvoice

					c.BREAK("subscription/invoice/RU/b2b/over-limit", func(c *tshort.Case[*legacy.Result]) {
						c.ExpectOk = resOverLimit
						mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 20000, Currency: "RUB"}, nil)
						mTax.EXPECT().Calculate("RU", 20000.0, false).Return(3000.0, nil)
						mCredit.EXPECT().Limit(order.CustomerID).Return(10000.0, nil)
						mNotif.EXPECT().Email(custB2B.Email, "Order on credit review").Return(nil)
						mBus.EXPECT().Publish("subscription.created", gomock.Any()).Return(nil)
					})
				})
			})

			// ================= kind не из констант =================
			// SUT: старая версия API присылает kind вне констант —
			// лист доживает до kind-секции и там же умирает.
			c.CASE(func(c *tshort.Collector[*legacy.Result]) {
				order = &legacy.Order{ID: 1, CustomerID: 1, Kind: legacy.KindDigital, ItemID: "item-1", Qty: 2, BasePrice: 5000, Region: "RU"}
				mCust.EXPECT().Get(order.CustomerID).Return(custStd, nil)
				mLoyalty.EXPECT().Tier(order.CustomerID).Return(legacy.TierSilver, nil)

				c.BREAK("kind/unknown", func(c *tshort.Case[*legacy.Result]) {
					c.ExpectErr = errUnknownKind
					order.Kind = 99
					mPrice.EXPECT().Quote(order.Kind, order.Qty, order.BasePrice).Return(legacy.Quote{Price: 10000, Currency: "RUB"}, nil)
					mTax.EXPECT().Calculate("RU", 10000.0, false).Return(2000.0, nil)
				})
			})
		})
	}, func(t *testing.T, leaf *tshort.Case[*legacy.Result]) {
		res, err := legacy.ProcessOrder(order, d)
		assert.Equal(t, leaf.ExpectErr, err)
		// на error-путях SUT возвращает nil — zero-значение ExpectOk
		assert.Equal(t, leaf.ExpectOk, res)
	})
}
