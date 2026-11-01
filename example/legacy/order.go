package legacy

import (
	"errors"
	"fmt"
	"log"
	"strconv"
)

// ProcessOrder — основной конвейер обработки заказа.
//
// Легаси: одна большая функция, потому что «всё работало и никто не хочет трогать».
// Структура намеренно повторяет реальные следы эволюции:
//   - дублирующаяся валидация (старые клиенты присылают мусор),
//   - ветки по kind / region / pay method,
//   - «legacy» хаки, которые нельзя удалять,
//   - обработчики ошибок, которые иногда игнорируются намеренно.
func ProcessOrder(o *Order, d *Deps) (res *Result, err error) {
	// ==================================================================
	// Валидация. Дублируется намеренно: до этого кода были три ревизии
	// API, и каждая присылает что-то своё.
	// ==================================================================
	if o == nil {
		return nil, errors.New("legacy: order is nil")
	}
	res = &Result{OrderID: o.ID}
	if o.ID <= 0 {
		return nil, errors.New("legacy: bad order id")
	}
	if o.CustomerID <= 0 {
		return nil, errors.New("legacy: bad customer id")
	}
	if o.Qty <= 0 {
		return nil, errors.New("legacy: bad qty")
	}
	if o.BasePrice < 0 {
		return nil, errors.New("legacy: negative base price")
	}
	if o.ItemID == "" && o.Kind == KindPhysical {
		return nil, errors.New("legacy: physical order without item")
	}
	if o.Region == "" {
		// legacy: регион по умолчанию, старый фронтенд не слал это поле
		o.Region = "RU"
	}

	// ==================================================================
	// Клиент
	// ==================================================================
	cust, err := d.CustomerRepo.Get(o.CustomerID)
	if err != nil {
		return nil, err
	}
	if cust == nil {
		_ = d.AuditLogger.Log("customer.notfound", map[string]any{"customer": o.CustomerID})
		return nil, errors.New("legacy: customer not found")
	}
	if cust.Blocked {
		_ = d.AuditLogger.Log("customer.blocked", map[string]any{"customer": o.CustomerID})
		return nil, errors.New("legacy: customer blocked")
	}
	// Чёрный список проверяется отдельно: таблица живёт в другой базе,
	// и синхронизация с ней иногда лагает (legacy).
	blocked, err := d.CustomerRepo.IsBlacklisted(o.CustomerID)
	if err != nil {
		return nil, err
	}
	if blocked {
		_ = d.Notifier.Email(cust.Email, "Order rejected: blacklist")
		return nil, errors.New("legacy: customer blacklisted")
	}

	tier, err := d.LoyaltyService.Tier(o.CustomerID)
	if err != nil {
		return nil, err
	}

	// ==================================================================
	// Ценообразование
	// ==================================================================
	q, err := d.PriceService.Quote(o.Kind, o.Qty, o.BasePrice)
	if err != nil {
		return nil, err
	}
	if q.Err != "" {
		// legacy: ценовой сервис иногда возвращает ошибку в поле, а не в error
		return nil, errors.New("legacy: price error: " + q.Err)
	}
	amount := q.Price

	// ==================================================================
	// Скидка по промокоду
	// ==================================================================
	if o.DiscountCode != "" {
		disc, err := d.DiscountService.Apply(o.DiscountCode, amount)
		if err != nil {
			// legacy: ошибка скидки не фатальна, просто логируем и идём дальше
			_ = d.AuditLogger.Log("discount.error", map[string]any{"code": o.DiscountCode, "err": err.Error()})
		} else {
			amount -= disc
			if amount < 0 {
				// legacy: не уходить в минус, бухгалтерия ругалась
				amount = 0
			}
			res.Notes = append(res.Notes, "discount:"+o.DiscountCode)
		}
	}

	// ==================================================================
	// Лояльность: списание баллов
	// ==================================================================
	if o.UsePoints && tier != TierNone {
		if tier == TierGold {
			red, err := d.LoyaltyService.Redeem(o.CustomerID, o.Qty*10)
			if err != nil {
				return nil, err
			}
			amount -= red
			if amount < 0 {
				amount = 0
			}
			res.Notes = append(res.Notes, "loyalty:gold")
		} else if tier == TierSilver {
			red, err := d.LoyaltyService.Redeem(o.CustomerID, o.Qty*5)
			if err != nil {
				return nil, err
			}
			// legacy: silver получает половину (так договорились в 2019)
			amount -= red / 2
			if amount < 0 {
				amount = 0
			}
			res.Notes = append(res.Notes, "loyalty:silver")
		}
	}

	// ==================================================================
	// Налоги. Самые ветвистый кусок легаси: правила отличаются по
	// регионам и по типу товара. В 2021 пытались вынести в таблицу,
	// откатили, потому что «таблица не покрывала US».
	// ==================================================================
	var tax float64
	if o.Kind == KindDigital {
		if o.Region == "RU" {
			// legacy: digital в RU уже с НДС в цене, ничего не считаем
			tax = 0
		} else if o.Region == "EU" {
			tax, err = d.TaxService.Calculate(o.Region, amount, true)
			if err != nil {
				return nil, err
			}
		} else if o.Region == "US" {
			tax, err = d.TaxService.Calculate(o.Region, amount, true)
			if err != nil {
				return nil, err
			}
			// legacy: фиксированная доплата за state tax, который сервис не знает
			tax += 5
		} else {
			// legacy: для остальных регионов плоские 5% — «взяли из экселя»
			tax = amount * 0.05
		}
	} else {
		if o.Region == "RU" {
			tax, err = d.TaxService.Calculate(o.Region, amount, false)
			if err != nil {
				return nil, err
			}
		} else if o.Region == "EU" {
			tax, err = d.TaxService.Calculate(o.Region, amount, false)
			if err != nil {
				return nil, err
			}
			// legacy: импортная пошлина EU, сверху к тому, что посчитал сервис
			tax += amount * 0.02
		} else if o.Region == "US" {
			tax, err = d.TaxService.Calculate(o.Region, amount, false)
			if err != nil {
				return nil, err
			}
		} else {
			// legacy: всё, что не назвали выше — 8%
			tax = amount * 0.08
		}
	}
	amount += tax

	// ==================================================================
	// Секция по типу заказа. Тут начинается самое интересное.
	// ==================================================================
	if o.Kind == KindPhysical {
		// ---------- склад ----------
		ok, err := d.StockService.Reserve(o.ItemID, o.Qty)
		if err != nil {
			return nil, err
		}
		if !ok {
			// не хватает на складе
			if o.Region == "RU" || o.Region == "EU" {
				// legacy: в RU/EU делаем backorder и всё равно идём в оплату
				res.Status = "backorder"
				_ = d.EventBus.Publish("stock.backorder", map[string]any{"order": o.ID, "item": o.ItemID})
				_ = d.Notifier.Email(cust.Email, "Order on backorder")
			} else {
				// остальные регионы отменяем, логистика не берёт
				return nil, errors.New("legacy: no stock")
			}
		}
		if ok {
			pick, err := d.WarehouseService.Pick(o.ItemID, o.Qty)
			if err != nil {
				// legacy: откат резерва при ошибке сборки
				_ = d.StockService.Release(o.ItemID, o.Qty)
				return nil, err
			}
			res.PickID = pick
		}

		// ---------- доставка ----------
		weight := o.Weight
		if weight <= 0 {
			// legacy: вес по умолчанию, старый фронт не слал
			weight = float64(o.Qty) * 0.5
		}
		rate, err := d.ShippingService.Rate(o.Region, weight, o.Express)
		if err != nil {
			return nil, err
		}
		if o.Express {
			// legacy: express = тариф + 50% сверху, так было договорено с перевозчиком
			amount += rate * 1.5
			res.Notes = append(res.Notes, "express")
		} else {
			amount += rate
		}
		carrier, err := d.ShippingService.PickCarrier(o.Region, weight)
		if err != nil {
			return nil, err
		}
		res.Carrier = carrier
	} else if o.Kind == KindDigital {
		// digital: нет склада, нет доставки
		if o.Qty > 5 {
			// legacy: оптовый digital возможен только для B2B
			if !cust.B2B {
				return nil, errors.New("legacy: bulk digital requires b2b")
			}
			res.Notes = append(res.Notes, "bulk-digital")
		}
		if o.Qty == 1 {
			// legacy: одиночная лицензия, старая интеграция ждёт заметку
			res.Notes = append(res.Notes, "single-license")
		}
	} else if o.Kind == KindSubscription {
		// подписка
		if o.DiscountCode == "" {
			// legacy: первый месяц -10% всем новым подписчикам
			amount *= 0.9
			res.Notes = append(res.Notes, "first-month-10%")
		}
		if cust.B2B {
			res.Notes = append(res.Notes, "b2b-subscription")
		}
	} else {
		// kind не из констант: так бывает, когда приходит старая версия API
		return nil, fmt.Errorf("legacy: unknown kind %d", int(o.Kind))
	}

	// ==================================================================
	// Пороги по сумме
	// ==================================================================
	if amount >= 50000 {
		_ = d.AuditLogger.Log("order.large", map[string]any{"order": o.ID, "amount": amount})
		if tier != TierGold && o.PayMethod == PayInvoice {
			// legacy: крупные invoice-заказы без gold-статуса проверяем по кредитке
			lim, err := d.CreditService.Limit(o.CustomerID)
			if err != nil {
				return nil, err
			}
			if amount > lim {
				res.Status = "credit_review"
				_ = d.Notifier.Email(cust.Email, "Order on credit review")
			}
		}
	}
	if amount < 100 {
		res.Notes = append(res.Notes, "small-order")
	}

	// ==================================================================
	// Оплата
	// ==================================================================
	if o.PayMethod == PayCard {
		score, err := d.FraudChecker.Score(o.CustomerID, amount)
		if err != nil {
			return nil, err
		}
		if score >= 80 {
			// держим на ручном осмотре
			res.Status = "held"
			_ = d.Notifier.Email(cust.Email, "Order held: fraud check")
			// legacy: при hold не шлём event и не пишем audit — «и так увидим»
			return res, nil
		}
		if score >= 50 {
			res.Notes = append(res.Notes, "fraud-review")
		}
		auth, err := d.PaymentGateway.Authorize(string(o.PayMethod), amount)
		if err != nil {
			if o.Kind == KindPhysical {
				_ = d.StockService.Release(o.ItemID, o.Qty)
			}
			return nil, err
		}
		res.AuthID = auth
		if err := d.PaymentGateway.Capture(auth, amount); err != nil {
			// legacy: capture упал — откатываем авторизацию и склад
			_ = d.PaymentGateway.Refund(auth, amount)
			if o.Kind == KindPhysical {
				_ = d.StockService.Release(o.ItemID, o.Qty)
			}
			return nil, err
		}
	} else if o.PayMethod == PayCrypto {
		// legacy: crypto не ходит в fraud (интеграцию не доделали в 2020)
		auth, err := d.PaymentGateway.Authorize(string(o.PayMethod), amount)
		if err != nil {
			if o.Kind == KindPhysical {
				_ = d.StockService.Release(o.ItemID, o.Qty)
			}
			return nil, err
		}
		res.AuthID = auth
		if err := d.PaymentGateway.Capture(auth, amount); err != nil {
			_ = d.PaymentGateway.Refund(auth, amount)
			if o.Kind == KindPhysical {
				_ = d.StockService.Release(o.ItemID, o.Qty)
			}
			return nil, err
		}
	} else if o.PayMethod == PayInvoice {
		if !cust.B2B {
			return nil, errors.New("legacy: invoice requires b2b")
		}
		lim, err := d.CreditService.Limit(o.CustomerID)
		if err != nil {
			return nil, err
		}
		if amount > lim {
			res.Status = "credit_review"
			_ = d.Notifier.Email(cust.Email, "Order on credit review")
		} else {
			if err := d.CreditService.Charge(o.CustomerID, amount); err != nil {
				return nil, err
			}
			inv, err := d.InvoiceService.Issue(o.CustomerID, amount)
			if err != nil {
				return nil, err
			}
			res.InvoiceID = inv
		}
	} else if o.PayMethod == PayCOD {
		if o.Kind != KindPhysical {
			return nil, errors.New("legacy: cod only for physical")
		}
		if o.Region != "RU" {
			return nil, errors.New("legacy: cod only in RU")
		}
		// cod: в гейтeway не ходим вообще, деньги соберёт курьер
	} else {
		return nil, fmt.Errorf("legacy: unknown pay method %s", string(o.PayMethod))
	}

	// ==================================================================
	// Финальный статус
	// ==================================================================
	if res.Status == "" {
		// legacy: cod и invoice тоже получают "paid", аналитика так настроена
		res.Status = "paid"
	}
	res.Total = amount

	// ==================================================================
	// Уведомления
	// ==================================================================
	if res.Status == "held" || res.Status == "credit_review" || res.Status == "backorder" {
		// письмо уже ушло в ветке, которая выставила статус
	} else {
		if err := d.Notifier.Email(cust.Email, "Order confirmed"); err != nil {
			return nil, err
		}
		if cust.Phone != "" {
			if err := d.Notifier.SMS(cust.Phone, "Order "+strconv.Itoa(o.ID)+" confirmed"); err != nil {
				// legacy: SMS упал — не критично, просто лог
				log.Printf("legacy: sms failed for order %d: %v", o.ID, err)
			}
		}
	}

	// ==================================================================
	// События
	// ==================================================================
	if err := d.EventBus.Publish("order.processed", res); err != nil {
		return nil, err
	}
	if o.Kind == KindSubscription {
		if err := d.EventBus.Publish("subscription.created", map[string]any{"order": o.ID, "customer": o.CustomerID}); err != nil {
			return nil, err
		}
	}

	// ==================================================================
	// Аудит
	// ==================================================================
	details := map[string]any{
		"order":    o.ID,
		"customer": o.CustomerID,
		"kind":     int(o.Kind),
		"amount":   amount,
		"status":   res.Status,
		"region":   o.Region,
		"tier":     int(tier),
	}
	if err := d.AuditLogger.Log("order.processed", details); err != nil {
		return nil, err
	}

	return res, nil
}
