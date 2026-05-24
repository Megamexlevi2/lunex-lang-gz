// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"fmt"
	"lunex/internal/runtime"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/invoice"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/price"
	"github.com/stripe/stripe-go/v76/product"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
)

func stripeClientObj() *runtime.Value {
	createPaymentIntent := runtime.FuncVal(&runtime.Function{
		Name: "createPaymentIntent",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("createPaymentIntent({amount, currency, ...})")
			}
			opts := args[0].ObjVal
			amount := int64(0)
			currency := "usd"
			if v, ok := opts["amount"]; ok && v.Tag == runtime.TypeNumber {
				amount = int64(v.NumVal)
			}
			if v, ok := opts["currency"]; ok {
				currency = v.ToString()
			}
			params := &stripe.PaymentIntentParams{
				Amount:   stripe.Int64(amount),
				Currency: stripe.String(currency),
			}
			if v, ok := opts["customer"]; ok {
				params.Customer = stripe.String(v.ToString())
			}
			if v, ok := opts["description"]; ok {
				params.Description = stripe.String(v.ToString())
			}
			pi, err := paymentintent.New(params)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":           runtime.StringVal(pi.ID),
				"amount":       runtime.NumberVal(float64(pi.Amount)),
				"currency":     runtime.StringVal(string(pi.Currency)),
				"status":       runtime.StringVal(string(pi.Status)),
				"clientSecret": runtime.StringVal(pi.ClientSecret),
			}), nil
		},
	})

	getPaymentIntent := runtime.FuncVal(&runtime.Function{
		Name: "getPaymentIntent",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("getPaymentIntent(id)")
			}
			pi, err := paymentintent.Get(args[0].ToString(), nil)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":       runtime.StringVal(pi.ID),
				"amount":   runtime.NumberVal(float64(pi.Amount)),
				"currency": runtime.StringVal(string(pi.Currency)),
				"status":   runtime.StringVal(string(pi.Status)),
			}), nil
		},
	})

	createCustomer := runtime.FuncVal(&runtime.Function{
		Name: "createCustomer",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("createCustomer({email, name, ...})")
			}
			opts := args[0].ObjVal
			params := &stripe.CustomerParams{}
			if v, ok := opts["email"]; ok {
				params.Email = stripe.String(v.ToString())
			}
			if v, ok := opts["name"]; ok {
				params.Name = stripe.String(v.ToString())
			}
			if v, ok := opts["phone"]; ok {
				params.Phone = stripe.String(v.ToString())
			}
			c, err := customer.New(params)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":    runtime.StringVal(c.ID),
				"email": runtime.StringVal(c.Email),
				"name":  runtime.StringVal(c.Name),
			}), nil
		},
	})

	getCustomer := runtime.FuncVal(&runtime.Function{
		Name: "getCustomer",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("getCustomer(id)")
			}
			c, err := customer.Get(args[0].ToString(), nil)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":    runtime.StringVal(c.ID),
				"email": runtime.StringVal(c.Email),
				"name":  runtime.StringVal(c.Name),
			}), nil
		},
	})

	createProduct := runtime.FuncVal(&runtime.Function{
		Name: "createProduct",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("createProduct({name, description, ...})")
			}
			opts := args[0].ObjVal
			params := &stripe.ProductParams{}
			if v, ok := opts["name"]; ok {
				params.Name = stripe.String(v.ToString())
			}
			if v, ok := opts["description"]; ok {
				params.Description = stripe.String(v.ToString())
			}
			p, err := product.New(params)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":          runtime.StringVal(p.ID),
				"name":        runtime.StringVal(p.Name),
				"description": runtime.StringVal(p.Description),
				"active":      runtime.BoolVal(p.Active),
			}), nil
		},
	})

	createPrice := runtime.FuncVal(&runtime.Function{
		Name: "createPrice",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("createPrice({product, unitAmount, currency, recurring?})")
			}
			opts := args[0].ObjVal
			params := &stripe.PriceParams{}
			if v, ok := opts["product"]; ok {
				params.Product = stripe.String(v.ToString())
			}
			if v, ok := opts["unitAmount"]; ok && v.Tag == runtime.TypeNumber {
				params.UnitAmount = stripe.Int64(int64(v.NumVal))
			}
			if v, ok := opts["currency"]; ok {
				params.Currency = stripe.String(v.ToString())
			}
			if v, ok := opts["recurring"]; ok && v.Tag == runtime.TypeObject {
				r := v.ObjVal
				interval := "month"
				if iv, ok := r["interval"]; ok {
					interval = iv.ToString()
				}
				params.Recurring = &stripe.PriceRecurringParams{
					Interval: stripe.String(interval),
				}
			}
			pr, err := price.New(params)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":         runtime.StringVal(pr.ID),
				"unitAmount": runtime.NumberVal(float64(pr.UnitAmount)),
				"currency":   runtime.StringVal(string(pr.Currency)),
			}), nil
		},
	})

	createCheckoutSession := runtime.FuncVal(&runtime.Function{
		Name: "createCheckoutSession",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("createCheckoutSession({lineItems, successUrl, cancelUrl, mode?})")
			}
			opts := args[0].ObjVal
			params := &stripe.CheckoutSessionParams{}
			if v, ok := opts["successUrl"]; ok {
				params.SuccessURL = stripe.String(v.ToString())
			}
			if v, ok := opts["cancelUrl"]; ok {
				params.CancelURL = stripe.String(v.ToString())
			}
			mode := "payment"
			if v, ok := opts["mode"]; ok {
				mode = v.ToString()
			}
			params.Mode = stripe.String(mode)
			if v, ok := opts["customer"]; ok {
				params.Customer = stripe.String(v.ToString())
			}
			if v, ok := opts["lineItems"]; ok && v.Tag == runtime.TypeArray {
				for _, item := range v.ArrVal {
					if item.Tag != runtime.TypeObject {
						continue
					}
					li := &stripe.CheckoutSessionLineItemParams{}
					if p, ok := item.ObjVal["price"]; ok {
						li.Price = stripe.String(p.ToString())
					}
					if q, ok := item.ObjVal["quantity"]; ok && q.Tag == runtime.TypeNumber {
						li.Quantity = stripe.Int64(int64(q.NumVal))
					}
					params.LineItems = append(params.LineItems, li)
				}
			}
			s, err := session.New(params)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":  runtime.StringVal(s.ID),
				"url": runtime.StringVal(s.URL),
			}), nil
		},
	})

	createSubscription := runtime.FuncVal(&runtime.Function{
		Name: "createSubscription",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("createSubscription({customer, items})")
			}
			opts := args[0].ObjVal
			params := &stripe.SubscriptionParams{}
			if v, ok := opts["customer"]; ok {
				params.Customer = stripe.String(v.ToString())
			}
			if v, ok := opts["items"]; ok && v.Tag == runtime.TypeArray {
				for _, item := range v.ArrVal {
					if item.Tag != runtime.TypeObject {
						continue
					}
					si := &stripe.SubscriptionItemsParams{}
					if p, ok := item.ObjVal["price"]; ok {
						si.Price = stripe.String(p.ToString())
					}
					params.Items = append(params.Items, si)
				}
			}
			sub, err := subscription.New(params)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":     runtime.StringVal(sub.ID),
				"status": runtime.StringVal(string(sub.Status)),
			}), nil
		},
	})

	cancelSubscription := runtime.FuncVal(&runtime.Function{
		Name: "cancelSubscription",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("cancelSubscription(id)")
			}
			sub, err := subscription.Cancel(args[0].ToString(), nil)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":     runtime.StringVal(sub.ID),
				"status": runtime.StringVal(string(sub.Status)),
			}), nil
		},
	})

	createRefund := runtime.FuncVal(&runtime.Function{
		Name: "createRefund",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("createRefund({paymentIntent, amount?})")
			}
			opts := args[0].ObjVal
			params := &stripe.RefundParams{}
			if v, ok := opts["paymentIntent"]; ok {
				params.PaymentIntent = stripe.String(v.ToString())
			}
			if v, ok := opts["amount"]; ok && v.Tag == runtime.TypeNumber {
				params.Amount = stripe.Int64(int64(v.NumVal))
			}
			r, err := refund.New(params)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"id":     runtime.StringVal(r.ID),
				"amount": runtime.NumberVal(float64(r.Amount)),
				"status": runtime.StringVal(string(r.Status)),
			}), nil
		},
	})

	listInvoices := runtime.FuncVal(&runtime.Function{
		Name: "listInvoices",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			params := &stripe.InvoiceListParams{}
			if len(args) > 0 && args[0].Tag == runtime.TypeObject {
				if v, ok := args[0].ObjVal["customer"]; ok {
					params.Customer = stripe.String(v.ToString())
				}
			}
			params.Limit = stripe.Int64(10)
			iter := invoice.List(params)
			var invoices []*runtime.Value
			for iter.Next() {
				inv := iter.Invoice()
				invoices = append(invoices, runtime.ObjectVal(map[string]*runtime.Value{
					"id":     runtime.StringVal(inv.ID),
					"amount": runtime.NumberVal(float64(inv.AmountDue)),
					"status": runtime.StringVal(string(inv.Status)),
				}))
			}
			if invoices == nil {
				invoices = []*runtime.Value{}
			}
			return runtime.ArrayVal(invoices), iter.Err()
		},
	})

	constructEvent := runtime.FuncVal(&runtime.Function{
		Name: "constructEvent",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.Null, fmt.Errorf("constructEvent(payload, sigHeader, secret)")
			}
			payload := []byte(args[0].ToString())
			sig := args[1].ToString()
			secret := args[2].ToString()
			event, err := webhook.ConstructEvent(payload, sig, secret)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"type": runtime.StringVal(string(event.Type)),
				"id":   runtime.StringVal(event.ID),
				"data": runtime.StringVal(string(event.Data.Raw)),
			}), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"createPaymentIntent":    createPaymentIntent,
		"getPaymentIntent":       getPaymentIntent,
		"createCustomer":         createCustomer,
		"getCustomer":            getCustomer,
		"createProduct":          createProduct,
		"createPrice":            createPrice,
		"createCheckoutSession":  createCheckoutSession,
		"createSubscription":     createSubscription,
		"cancelSubscription":     cancelSubscription,
		"createRefund":           createRefund,
		"listInvoices":           listInvoices,
		"constructEvent":         constructEvent,
	})
}

func StripeModule() *runtime.Value {
	init := runtime.FuncVal(&runtime.Function{
		Name: "init",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("init(apiKey)")
			}
			stripe.Key = args[0].ToString()
			return stripeClientObj(), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"init": init,
	})
}
