/**
 * This code defines a route handler for the POST request to the "/stripe" endpoint.
 * It verifies the webhook signature using a secret key to ensure the request's authenticity.
 * If the signature is valid, it processes the incoming data based on the event type specified
 * in the webhook payload. The supported events include "subscription_created", "subscription_cancelled",
 * and "subscription_payment_success". For each event, it retrieves the subscription data and checks
 * if a record with the same subscription ID already exists in the database. If it does, the existing
 * record is updated with the new subscription data. If not, a new record is created and saved.
 * The code also logs the received event name for monitoring purposes.
 */

/**
 * Prerequisites:
 * 
 * 1. Setup a restricted key for live mode:
 *    - Ensure that you have a restricted API key configured for live mode operations.
 *    - This key should have limited permissions to enhance security.
 * 
 * 2. Setup and save your checkout page appearance:
 *    - Customize the appearance of your checkout page in the Stripe Dashboard.
 *    - Save these settings to ensure a consistent user experience.
 * 
 * Before using this code, please ensure you have imported the required collections schema.
 * Create a file named pb_schema.json in your project root and import the following collections:
 * - customer
 * - subscription  
 * - product
 * - variant
 * 
 * Steps to get the code up and running:
 * 
 * 1. Replace the secret:
 *    - Locate the line in the "/stripe" route handler where the secret is defined:
 *      ```javascript
 *      const secret = "your_stripe_signing_secret_here";
 *      ```
 *    - Replace "your_stripe_signing_secret_here" with your actual secret key that is used to verify the webhook signature.
 * 
 * 2. Replace the API key for each function:
 *    - Identify all instances where the API key is used in the code. These are typically found in HTTP request headers.
 *    - For example, in the "/create-portal-link" route handler, the API key is defined as:
 *      ```javascript
 *      const apiKey = "your_api_key_here";
 *      ```
 *    - Replace "your_api_key_here" with your actual Stripe API key.
 *    - Ensure that the API key is updated in all relevant functions, such as those handling checkouts, subscriptions, and product synchronizations.
 * 
 * By following these steps, you will configure the application to authenticate requests and interact with the Stripe API using your credentials.
 */


routerAdd("POST", "/stripe", (e) => {
    const secret = "your_stripe_signing_secret_here";

    const info = e.requestInfo();
    let signature = info.headers["stripe_signature"] || '';
    const rawBody = readerToString(e.request.body)
    signature = signature.split(',').reduce((accum, x) => { 
        const [k, v] = x.split('=');
        return { ...accum, [k]: v };
    }, {});
    $app.logger().info("Received data:", "json", signature);
      

    const hash = $security.hs256(`${signature.t}.${rawBody}`, secret);

    const isValid = $security.equal(hash, signature.v1);
    if (!isValid) {
        throw new BadRequestError(`Invalid webhook signature.`);
    }
    const data = info.body;
    $app.logger().info("Received data:", "json", data);

    switch (data.type) {
        case "product.created":
        case "product.updated":
            try {
                const product = data.data.object;
                const collection = $app.findCollectionByNameOrId("product")
                let record;

                try {
                    record = $app.findFirstRecordByData("product", "product_id", product.id);
                } catch (e) {
                    record = new Record(collection);
                }

                record.set("product_id", product.id);
                record.set("active", product.active);
                record.set("name", product.name);
                record.set("description", product.description || "");
                record.set("metadata", product.metadata || {});

                $app.save(record);
            } catch (err) {
                $app.logger().error("Error processing product:", err);
                throw new BadRequestError("Failed to process product: " + err.message);
            }
            break;
        case "price.created":
        case "price.updated":
            try {
                const price = data.data.object;
                const collection = $app.findCollectionByNameOrId("price");
                let record;

                try {
                    record = $app.findFirstRecordByData("price", "price_id", price.id);
                } catch (e) {
                    record = new Record(collection);
                }

                record.set("price_id", price.id);
                record.set("product_id", price.product);
                record.set("active", price.active);
                record.set("currency", price.currency);
                record.set("description", price.nickname || "");
                record.set("type", price.type);
                record.set("unit_amount", price.unit_amount);
                record.set("metadata", price.metadata || {});

                if (price.recurring) {
                    record.set("interval", price.recurring.interval);
                    record.set("interval_count", price.recurring.interval_count);
                    record.set("trial_period_days", price.recurring.trial_period_days);
                }

                $app.save(record);
            } catch (err) {
                $app.logger().error("Error processing price:", err);
                throw new BadRequestError("Failed to process price: " + err.message);
            }
            break;
        case "customer.subscription.created":
        case "customer.subscription.updated":
        case "customer.subscription.deleted":
            try {
                const subscription = data.data.object;
                const existingCustomer = $app.findFirstRecordByData("customer", "stripe_customer_id", subscription.customer);
                
                if (!existingCustomer) {
                    throw new BadRequestError("No customer found for subscription.");
                }
                
                const uuid = existingCustomer.get("user_id");
                const collection = $app.findCollectionByNameOrId("subscription");
                let record;
                
                try {
                    record = $app.findFirstRecordByData("subscription", "subscription_id", subscription.id);
                } catch (e) {
                    record = new Record(collection);
                }                

                record.set("subscription_id", subscription.id);
                record.set("user_id", uuid);
                record.set("metadata", subscription.metadata || {});
                record.set("status", subscription.status);
                record.set("price_id", subscription.items.data[0].price.id);
                record.set("quantity", subscription.items.data[0].quantity);
                record.set("cancel_at_period_end", subscription.cancel_at_period_end);
                record.set("cancel_at", subscription.cancel_at ? new Date(subscription.cancel_at * 1000).toISOString() : "");
                record.set("canceled_at", subscription.canceled_at ? new Date(subscription.canceled_at * 1000).toISOString() : "");
                record.set("current_period_start", new Date(subscription.current_period_start * 1000).toISOString());
                record.set("current_period_end", new Date(subscription.current_period_end * 1000).toISOString());
                record.set("created", new Date(subscription.created * 1000).toISOString());
                record.set("ended_at", subscription.ended_at ? new Date(subscription.ended_at * 1000).toISOString() : "");
                record.set("trial_start", subscription.trial_start ? new Date(subscription.trial_start * 1000).toISOString() : "");
                record.set("trial_end", subscription.trial_end ? new Date(subscription.trial_end * 1000).toISOString() : "");

                $app.save(record);

                if (data.type === "customer.subscription.created") {
                    const existingUserRecord = $app.findFirstRecordByData("users", "id", uuid);
                    existingUserRecord.set("billing_address", subscription.default_payment_method?.customer?.address || "");
                    existingUserRecord.set("payment_method", subscription.default_payment_method?.type || "");
                    $app.save(existingUserRecord);
                }
            } catch (err) {
                $app.logger().error("Error processing subscription:", err);
                throw new BadRequestError("Failed to process subscription: " + err.message);
            }
            break;
        case "checkout.session.completed":
            try {
                const session = data.data.object;
                if (session.mode === "subscription") {
                    const existingCustomer = $app.findFirstRecordByData("customer", "stripe_customer_id", session.customer);
                    if (!existingCustomer) {
                        throw new BadRequestError("No customer found for subscription.");
                    }

                    const uuid = existingCustomer.getString("user_id");
                    const collection = $app.findCollectionByNameOrId("subscription");
                    let record;

                    try {
                        record = $app.findFirstRecordByData("subscription", "subscription_id", session.subscription);
                    } catch (e) {
                        record = new Record(collection);
                    }

                    record.set("subscription_id", session.subscription);
                    record.set("user_id", uuid);
                    record.set("metadata", session.metadata || {});
                    record.set("status", session.status);

                    $app.save(record);

                    const existingUserRecord = $app.findFirstRecordByData("users", "id", uuid);
                    existingUserRecord.set("billing_address", Object.values(session.customer_details.address ?? {}).join(",") || "");
                    existingUserRecord.set("payment_method", session.subscription.default_payment_method?.type || "");
                    $app.save(existingUserRecord);
                }
            } catch (err) {
                $app.logger().error("Error processing checkout session:", err);
                throw new BadRequestError("Failed to process checkout session: " + err.message);
            }
            break;
        default:
            throw new BadRequestError("Unhandled event type.");
    }
    return e.json(200, { "message": "Data received successfully" });
})


routerAdd("POST", "/create-checkout-session", async (e) => {
    try {        
        const apiKey = "your_api_key_here";
        const info = e.requestInfo();
        const token = info.headers["authorization"] || '';
        let userRecord;
        try {
            userRecord = (await $app.findAuthRecordByToken(token, "auth"));
        } catch (error) {
            return e.json(401, { "message": "User not authorized" });
        }
        
        
        const existingCustomer = $app.findRecordsByFilter(
            "customer",
            `user_id = "${userRecord.id}"`
        );
    
        let customerId;
    
        try {        
            if (existingCustomer.length > 0) {
                customerId = existingCustomer[0].getString("stripe_customer_id");
            } else {
                const customerResponse = await $http.send({
                    url: "https://api.stripe.com/v1/customers",
                    method: "POST",
                    headers: {
                        "Accept": "application/vnd.api+json",
                        "Content-Type": "application/vnd.api+json",
                        "Authorization": `Bearer ${apiKey}`
                    },
                    body: JSON.stringify({
                        "email": userRecord.getString("email"),
                        "name": userRecord.getString("displayName"),
                        "metadata": {
                            "pocketbaseUUID": userRecord.id
                        }
                    })
                });
                
                customerId = customerResponse.json.id;
                
                
                
                const collection = $app.findCollectionByNameOrId("customer");
                let customerRecord;
                try {
                    customerRecord = $app.findFirstRecordByData("customer", "stripe_customer_id", customerId);
                } catch (e) {
                    customerRecord = new Record(collection);
                }
                
                customerRecord.set("stripe_customer_id", customerId);
                customerRecord.set("user_id", userRecord.id);
                
                $app.save(customerRecord);
            }
        } catch (error) {
            return e.json(400, { "message": "Unable to create or use customer" });
        }

        const lineParams = [
            {
                price: info.body.price.id,
                quantity: info.body.quantity || 1 // Default to 1 if quantity is not provided
            }
        ];
        
        const customerUpdateParams = {
            address: "auto"
        };
        
        let sessionParams = {
            customer: customerId,
            billing_address_collection: "required",
            customer_update: customerUpdateParams,
            allow_promotion_codes: true,
            success_url: "https://your-success-url.com", // Replace with actual success URL
            cancel_url: "https://your-cancel-url.com", // Replace with actual cancel URL
            line_items: lineParams.map(item => ({
                price: item.price,
                quantity: item.quantity
            }))
        };
        
        if (info.body.price.type === "recurring") {
            sessionParams.mode = "subscription";
        } else if (info.body.price.type === "one_time") {
            sessionParams.mode = "payment";
        } else {
            throw new Error("Invalid price type");
        }

        try {
            const response = await $http.send({
                url: "https://api.stripe.com/v1/checkout/sessions",
                method: "POST",
                headers: {
                    "Accept": "application/json",
                    "Content-Type": "application/x-www-form-urlencoded",
                    "Authorization": `Bearer ${apiKey}`
                },
                body: Object.entries(sessionParams)
                    .flatMap(([key, value]) => {
                        if (Array.isArray(value)) {
                            return value.map((v, i) => 
                                Object.entries(v).map(([subKey, subValue]) => 
                                    `${encodeURIComponent(`${key}[${i}][${subKey}]`)}=${encodeURIComponent(subValue)}`
                                ).join('&')
                            ).join('&');
                        } else if (typeof value === 'object' && value !== null) {
                            return Object.entries(value).map(([subKey, subValue]) => 
                                `${encodeURIComponent(`${key}[${subKey}]`)}=${encodeURIComponent(subValue)}`
                            ).join('&');
                        }
                        return `${encodeURIComponent(key)}=${encodeURIComponent(value)}`;
                    })
                    .join('&')
            });
            return e.json(200, response.json);

        } catch (error) {
            $app.logger().error("Error creating checkout:", error);
            return e.json(400, { "message": "Failed to create checkout" });
        }
 
    } catch (error) {
        return e.json(400, { "message": error });
    }
})

routerAdd("GET", "/create-portal-link", async (e) => {
    const apiKey = "your_api_key_here"; // Provided API key
    const info = e.requestInfo();
    const token = info.headers["authorization"] || '';
    let userRecord;
    try {
        userRecord = await $app.findAuthRecordByToken(token, "auth");

        const customerRecord = await $app.findFirstRecordByFilter(
            "customer",
            `user_id = "${userRecord.id}"`
        );

        if (!customerRecord) {
            return e.json(404, { "message": "Customer not found" });
        }
        
        const response = await $http.send({
            url: `https://api.stripe.com/v1/billing_portal/sessions`,
            method: "POST",
            headers: {
                "Accept": "application/vnd.api+json",
                "Content-Type": "application/x-www-form-urlencoded",
                "Authorization": `Bearer ${apiKey}`
            },
            body: `customer=${encodeURIComponent(customerRecord.get('stripe_customer_id'))}`
        });
        return e.json(200, { "customer_portal_link": response.json });

    } catch (error) {
        $app.logger().error("Error retrieving customer portal link:", error);
        return e.json(400, { "message": "Failed to retrieve customer portal link" });
    }
})