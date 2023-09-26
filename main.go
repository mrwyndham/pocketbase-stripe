package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"

	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/billingportal/session"
	checkoutSession "github.com/stripe/stripe-go/v75/checkout/session"
	"github.com/stripe/stripe-go/v75/customer"
	"github.com/stripe/stripe-go/v75/webhook"
)

func coalesce(value *string, defaultValue string) string {
	if value != nil {
		return *value
	}
	return defaultValue
}

func int64ToISODate(timestamp int64) string {
	// Convert the Unix timestamp to a time.Time
	t := time.Unix(timestamp, 0)

	// Format the time as an ISO 8601 date string (in UTC)
	return t.Format(time.RFC3339)
}

func main() {
	app := pocketbase.New()
	stripe.Key = "{YOUR_STRIPE_SECRET_KEY_HERE}"
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.POST("/create-checkout-session", func(c echo.Context) error {
			// 1. Destructure the price and quantity from the POST body
			body := c.Request().Body
			defer body.Close()
			payload, err := io.ReadAll(body)
			var data map[string]interface{}
			json.Unmarshal([]byte(payload), &data)

			price, _ := data["price"].(map[string]interface{})
			quantity, _ := data["quantity"].(float64)

			// 2. Get the user from pocketbase auth
			token := c.Request().Header.Get("Authorization")
			record, err := app.Dao().FindAuthRecordByToken(token, app.Settings().RecordAuthToken.Secret)

			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"failure": "Could not get user"})
			}

			// 3. Retrieve or create the customer in Stripe
			existingCustomerRecord, err := app.Dao().FindFirstRecordByData("customer", "user_id", record.Id)
			if err != nil {
				//create new customer if none exists
				customerParams := &stripe.CustomerParams{
					Metadata: map[string]string{
						"pocketbaseUUID": record.GetString("id"),
					},
				}

				stripeCustomer, _ := customer.New(customerParams)

				//upload customer to pocketbase
				collection, err := app.Dao().FindCollectionByNameOrId("customer")
				if err != nil {
					return err
				}
				var form *forms.RecordUpsert
				newCustomerRecord := models.NewRecord(collection)

				if err == nil {
					form = forms.NewRecordUpsert(app, newCustomerRecord)
				}

				form.LoadData(map[string]any{
					"user_id":            record.Id,
					"stripe_customer_id": stripeCustomer.ID,
				})

				if err := form.Submit(); err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "Could not create new customer"})
				}

				//Do Pricing New Customer
				if price["type"] == "recurring" {
					//create new session
					lineParams := []*stripe.CheckoutSessionLineItemParams{
						{
							Price:    stripe.String(price["id"].(string)),
							Quantity: stripe.Int64(int64(quantity)),
						},
					}
					customerUpdateParams := &stripe.CheckoutSessionCustomerUpdateParams{
						Address: stripe.String("auto"),
					}
					subscriptionParams := &stripe.CheckoutSessionSubscriptionDataParams{
						Metadata: map[string]string{},
					}

					sessionParams := &stripe.CheckoutSessionParams{
						Customer:                 &stripeCustomer.ID,
						PaymentMethodTypes:       stripe.StringSlice([]string{"card"}),
						BillingAddressCollection: stripe.String("required"),
						CustomerUpdate:           customerUpdateParams,
						Mode:                     stripe.String("subscription"),
						AllowPromotionCodes:      stripe.Bool(true),
						SuccessURL:               stripe.String("https://www.sign365.com.au/account"),
						CancelURL:                stripe.String("https://www.sign365.com.au/"),
						LineItems:                lineParams,
						SubscriptionData:         subscriptionParams,
					}
					sesh, _ := checkoutSession.New(sessionParams)
					return c.JSON(http.StatusOK, sesh)
				} else if price["type"] == "one_time" {
					//create new session
					lineParams := []*stripe.CheckoutSessionLineItemParams{
						{
							Price:    stripe.String(price["id"].(string)),
							Quantity: stripe.Int64(int64(quantity)),
						},
					}
					customerUpdateParams := &stripe.CheckoutSessionCustomerUpdateParams{
						Address: stripe.String("auto"),
					}

					sessionParams := &stripe.CheckoutSessionParams{
						Customer:                 &stripeCustomer.ID,
						PaymentMethodTypes:       stripe.StringSlice([]string{"card"}),
						BillingAddressCollection: stripe.String("required"),
						CustomerUpdate:           customerUpdateParams,
						Mode:                     stripe.String("payment"),
						AllowPromotionCodes:      stripe.Bool(true),
						SuccessURL:               stripe.String("https://www.sign365.com.au/account"),
						CancelURL:                stripe.String("https://www.sign365.com.au/"),
						LineItems:                lineParams,
					}
					sesh, _ := checkoutSession.New(sessionParams)
					return c.JSON(http.StatusOK, sesh)
				} else {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "Could not create new session"})
				}
			} else {
				//Do Pricing Existing
				if price["type"] == "recurring" {
					//create new session
					lineParams := []*stripe.CheckoutSessionLineItemParams{
						{
							Price:    stripe.String(price["id"].(string)),
							Quantity: stripe.Int64(int64(quantity)),
						},
					}
					customerUpdateParams := &stripe.CheckoutSessionCustomerUpdateParams{
						Address: stripe.String("auto"),
					}
					subscriptionParams := &stripe.CheckoutSessionSubscriptionDataParams{
						Metadata: map[string]string{},
					}

					sessionParams := &stripe.CheckoutSessionParams{
						Customer:                 stripe.String(existingCustomerRecord.GetString("stripe_customer_id")),
						PaymentMethodTypes:       stripe.StringSlice([]string{"card"}),
						BillingAddressCollection: stripe.String("required"),
						CustomerUpdate:           customerUpdateParams,
						Mode:                     stripe.String("subscription"),
						AllowPromotionCodes:      stripe.Bool(true),
						SuccessURL:               stripe.String("https://www.sign365.com.au/account"),
						CancelURL:                stripe.String("https://www.sign365.com.au/"),
						LineItems:                lineParams,
						SubscriptionData:         subscriptionParams,
					}
					sesh, _ := checkoutSession.New(sessionParams)
					return c.JSON(http.StatusOK, sesh)
				} else if price["type"] == "one_time" {
					//create new session
					lineParams := []*stripe.CheckoutSessionLineItemParams{
						{
							Price:    stripe.String(price["id"].(string)),
							Quantity: stripe.Int64(int64(quantity)),
						},
					}
					customerUpdateParams := &stripe.CheckoutSessionCustomerUpdateParams{
						Address: stripe.String("auto"),
					}

					sessionParams := &stripe.CheckoutSessionParams{
						Customer:                 stripe.String(existingCustomerRecord.GetString("stripe_customer_id")),
						PaymentMethodTypes:       stripe.StringSlice([]string{"card"}),
						BillingAddressCollection: stripe.String("required"),
						CustomerUpdate:           customerUpdateParams,
						Mode:                     stripe.String("payment"),
						AllowPromotionCodes:      stripe.Bool(true),
						SuccessURL:               stripe.String("https://www.sign365.com.au/account"),
						CancelURL:                stripe.String("https://www.sign365.com.au/"),
						LineItems:                lineParams,
					}
					sesh, _ := checkoutSession.New(sessionParams)
					return c.JSON(http.StatusOK, sesh)
				} else {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "Could not create new session for stripe"})
				}
			}
		})
		return nil
	})
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.POST("/create-portal-link", func(c echo.Context) error {
			// 1. Get the user from pocketbase auth
			token := c.Request().Header.Get("Authorization")
			record, err := app.Dao().FindAuthRecordByToken(token, app.Settings().RecordAuthToken.Secret)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"failure": "Could not get user"})
			}

			// 2. Retrieve or create the customer in Stripe
			existingCustomerRecord, err := app.Dao().FindFirstRecordByData("customer", "user_id", record.Id)
			if err != nil {
				//create new customer if none exists
				customerParams := &stripe.CustomerParams{
					Metadata: map[string]string{
						"pocketbaseUUID": record.GetString("id"),
					},
				}

				stripeCustomer, _ := customer.New(customerParams)

				//upload customer to pocketbase
				collection, err := app.Dao().FindCollectionByNameOrId("customer")
				if err != nil {
					return err
				}
				var form *forms.RecordUpsert
				newCustomerRecord := models.NewRecord(collection)

				if err == nil {
					form = forms.NewRecordUpsert(app, newCustomerRecord)
				}

				form.LoadData(map[string]any{
					"user_id":            record.Id,
					"stripe_customer_id": stripeCustomer.ID,
				})

				if err := form.Submit(); err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "Could not create new customer"})
				}

				//create new session
				sessionParams := &stripe.BillingPortalSessionParams{
					Customer:  stripe.String(stripeCustomer.ID),
					ReturnURL: stripe.String("https://sign365.com.au/account"),
				}
				sesh, err := session.New(sessionParams)
				if err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "Could not create new session"})
				} else {
					return c.JSON(http.StatusOK, sesh)
				}

			} else {
				//create new session
				sessionParams := &stripe.BillingPortalSessionParams{
					Customer:  stripe.String(existingCustomerRecord.GetString("stripe_customer_id")),
					ReturnURL: stripe.String("https://sign365.com.au/account"),
				}
				sesh, err := session.New(sessionParams)
				if err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "Could not create new session"})
				} else {
					return c.JSON(http.StatusOK, sesh)
				}
			}
		})
		return nil
	})

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.POST("/stripe", func(c echo.Context) error {
			// Read the request body into a byte slice
			body := c.Request().Body
			defer body.Close() // Close the body when done
			payload, err := io.ReadAll(body)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"failure": "failed to read body"})
			}

			event := stripe.Event{}
			err = json.Unmarshal(payload, &event)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"failure": "failed to parse JSON"})
			}

			endpointSecret := "{YOUR_WEBHOOK_SECRET_HERE}"
			signatureHeader := c.Request().Header.Get("Stripe-Signature")
			event, err = webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"failure": "webhook verification failed"})
			}

			switch event.Type {
			case "product.created", "product.updated":
				var product stripe.Product
				err := json.Unmarshal(event.Data.Raw, &product)
				if err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "failed to marshall the stripe event"})
				}
				// Then define and call a func to handle the successful payment intent.

				collection, err := app.Dao().FindCollectionByNameOrId("product")
				if err != nil {
					return err
				}

				existingRecord, err := app.Dao().FindFirstRecordByData("product", "product_id", product.ID)
				record := models.NewRecord(collection)

				var form *forms.RecordUpsert

				if err == nil && existingRecord != nil {
					// Existing record found, update it
					// You might need to map data from product to your record
					// Assuming UpdateRecord updates the existing record with new data
					form = forms.NewRecordUpsert(app, existingRecord)
				} else {
					// Existing record not found, insert a new record
					// You might need to map data from product to your record
					// Assuming InsertRecord inserts a new record
					form = forms.NewRecordUpsert(app, record)
				}

				form.LoadData(map[string]any{
					"product_id":  product.ID,
					"active":      product.Active,
					"name":        product.Name,
					"description": coalesce(&product.Description, ""),
					"metadata":    product.Metadata,
				})

				// validate and submit (internally it calls app.Dao().SaveRecord(record) in a transaction)
				if err := form.Submit(); err != nil {
					return err
				}
			case "price.created", "price.updated":
				var price stripe.Price
				err := json.Unmarshal(event.Data.Raw, &price)
				if err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "failed to marshall the stripe event"})
				}
				// Then define and call a func to handle the successful payment intent.

				collection, err := app.Dao().FindCollectionByNameOrId("price")
				if err != nil {
					return err
				}

				existingRecord, err := app.Dao().FindFirstRecordByData("product", "price_id", price.ID)
				record := models.NewRecord(collection)

				var form *forms.RecordUpsert

				if err == nil && existingRecord != nil {
					// Existing record found, update it
					// You might need to map data from product to your record
					// Assuming UpdateRecord updates the existing record with new data
					form = forms.NewRecordUpsert(app, existingRecord)
				} else {
					// Existing record not found, insert a new record
					// You might need to map data from product to your record
					// Assuming InsertRecord inserts a new record
					form = forms.NewRecordUpsert(app, record)
				}

				form.LoadData(map[string]any{
					"price_id":          price.ID,
					"product_id":        price.Product.ID,
					"active":            price.Active,
					"currency":          price.Currency,
					"description":       price.Nickname,
					"type":              price.Type,
					"unit_amount":       price.UnitAmount,
					"interval":          price.Recurring.Interval,
					"interval_count":    price.Recurring.IntervalCount,
					"trial_period_days": price.Recurring.TrialPeriodDays,
					"metadata":          price.Metadata,
				})

				// validate and submit (internally it calls app.Dao().SaveRecord(record) in a transaction)
				if err := form.Submit(); err != nil {
					return err
				}
			case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
				var subscription stripe.Subscription
				err := json.Unmarshal(event.Data.Raw, &subscription)
				if err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "failed to marshall the stripe event"})
				}
				//Get customer's UUID from mapping table in order to update users billing address and payment method
				existingCustomer, err := app.Dao().FindFirstRecordByData("customer", "stripe_customer_id", subscription.Customer.ID)
				if err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "no customer"})
				}

				var uuid = existingCustomer.GetString("user_id")
				collection, err := app.Dao().FindCollectionByNameOrId("subscription")
				if err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "collection doesn't exist"})
				}

				//Update Subscription Details
				existingRecord, err := app.Dao().FindFirstRecordByData("subscription", "subscription_id", subscription.ID)
				record := models.NewRecord(collection)

				var form *forms.RecordUpsert

				if err == nil && existingRecord != nil {
					// Existing record found, update it
					// You might need to map data from product to your record
					// Assuming UpdateRecord updates the existing record with new data
					form = forms.NewRecordUpsert(app, existingRecord)
				} else {
					// Existing record not found, insert a new record
					// You might need to map data from product to your record
					// Assuming InsertRecord inserts a new record
					form = forms.NewRecordUpsert(app, record)
				}

				form.LoadData(map[string]any{
					"subscription_id":      subscription.ID,
					"user_id":              uuid,
					"metadata":             subscription.Metadata,
					"status":               subscription.Status,
					"price_id":             subscription.Items.Data[0].Price.ID,
					"quantity":             subscription.Items.Data[0].Quantity,
					"cancel_at_period_end": subscription.CancelAtPeriodEnd,
					"cancel_at":            int64ToISODate(subscription.CancelAt),
					"canceled_at":          int64ToISODate(subscription.CanceledAt),
					"current_period_start": int64ToISODate(subscription.CurrentPeriodStart),
					"current_period_end":   int64ToISODate(subscription.CurrentPeriodEnd),
					"created":              int64ToISODate(subscription.Items.Data[0].Created),
					"ended_at":             int64ToISODate(subscription.EndedAt),
					"trial_start":          int64ToISODate(subscription.TrialStart),
					"trial_end":            int64ToISODate(subscription.TrialEnd),
				})
				if err := form.Submit(); err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "couldn't submit subscription update"})
				}

				//Update User Details If Subscription Created
				if event.Type == "customer.subscription.created" {
					existingUserRecord, _ := app.Dao().FindFirstRecordByData("user", "id", uuid)
					var userForm = forms.NewRecordUpsert(app, existingUserRecord)

					userForm.LoadData(map[string]any{
						"billing_address": subscription.DefaultPaymentMethod.Customer.Address,
						"payment_method":  subscription.DefaultPaymentMethod.Type,
					})
					// validate and submit (internally it calls app.Dao().SaveRecord(record) in a transaction)
					if err := userForm.Submit(); err != nil {
						return c.JSON(http.StatusBadRequest, map[string]string{"failure": "couldn't submit user update"})
					}
				}
			case "checkout.session.completed":
				var session stripe.CheckoutSession
				err := json.Unmarshal(event.Data.Raw, &session)
				if err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"failure": "failed to marshall the stripe event"})
				}
				if session.Mode == "subscription" {
					//Get customer's UUID from mapping table in order to update users billing address and payment method
					existingCustomer, err := app.Dao().FindFirstRecordByData("customer", "stripe_customer_id", session.Subscription.Customer.ID)
					if err != nil {
						return c.JSON(http.StatusBadRequest, map[string]string{"failure": "no customer"})
					}

					var uuid = existingCustomer.GetString("user_id")
					collection, err := app.Dao().FindCollectionByNameOrId("subscription")
					if err != nil {
						return c.JSON(http.StatusBadRequest, map[string]string{"failure": "collection doesn't exist"})
					}

					//Update Subscription Details
					existingRecord, err := app.Dao().FindFirstRecordByData("subscription", "subscription_id", session.Subscription.ID)
					record := models.NewRecord(collection)

					var form *forms.RecordUpsert

					if err == nil && existingRecord != nil {
						// Existing record found, update it
						// You might need to map data from product to your record
						// Assuming UpdateRecord updates the existing record with new data
						form = forms.NewRecordUpsert(app, existingRecord)
					} else {
						// Existing record not found, insert a new record
						// You might need to map data from product to your record
						// Assuming InsertRecord inserts a new record
						form = forms.NewRecordUpsert(app, record)
					}

					form.LoadData(map[string]any{
						"subscription_id":      session.Subscription.ID,
						"user_id":              uuid,
						"metadata":             session.Subscription.Metadata,
						"status":               session.Subscription.Status,
						"price_id":             session.Subscription.Items.Data[0].Price.ID,
						"quantity":             session.Subscription.Items.Data[0].Quantity,
						"cancel_at_period_end": session.Subscription.CancelAtPeriodEnd,
						"cancel_at":            int64ToISODate(session.Subscription.CancelAt),
						"canceled_at":          int64ToISODate(session.Subscription.CanceledAt),
						"current_period_start": int64ToISODate(session.Subscription.CurrentPeriodStart),
						"current_period_end":   int64ToISODate(session.Subscription.CurrentPeriodEnd),
						"created":              int64ToISODate(session.Subscription.Items.Data[0].Created),
						"ended_at":             int64ToISODate(session.Subscription.EndedAt),
						"trial_start":          int64ToISODate(session.Subscription.TrialStart),
						"trial_end":            int64ToISODate(session.Subscription.TrialEnd),
					})
					if err := form.Submit(); err != nil {
						return c.JSON(http.StatusBadRequest, map[string]string{"failure": "couldn't submit subscription update"})
					}

					//Update User Details
					existingUserRecord, err := app.Dao().FindFirstRecordByData("user", "id", uuid)
					var userForm = forms.NewRecordUpsert(app, existingUserRecord)

					userForm.LoadData(map[string]any{
						"billing_address": session.Subscription.DefaultPaymentMethod.Customer.Address,
						"payment_method":  session.Subscription.DefaultPaymentMethod.Type,
					})

					// validate and submit (internally it calls app.Dao().SaveRecord(record) in a transaction)
					if err := userForm.Submit(); err != nil {
						return c.JSON(http.StatusBadRequest, map[string]string{"failure": "couldn't submit user update"})
					}
				}
			default:
				return c.JSON(http.StatusBadRequest, map[string]string{"failure": "didn't receive a valid event"})
			}

			return c.JSON(http.StatusOK, map[string]interface{}{"success": "data was received"})
		} /* optional middlewares */)

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
