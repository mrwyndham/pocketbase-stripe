# PocketBase + Stripe Subscription Payments Starter

The all-in-one starter kit for high-performance SaaS applications. That don't want a vendor buy in when it comes to backend and frontend. This is a front end agnostic template that you can use 100% with any SaaS application. If there are any issues please feel free to reach out to me on X - [@meinbiz](https://twitter.com/meinbiz)

## Features

- User management and authentication with [PocketBase](https://pocketbase.io/docs/authentication)
- Powerful data access & management tooling on top of SQL Lite with [PocketBase](https://Pocketbase.io/docs/guides/database)
- Integration with [Stripe Checkout](https://stripe.com/docs/payments/checkout) and the [Stripe customer portal](https://stripe.com/docs/billing/subscriptions/customer-portal)
- Automatic syncing of pricing plans and subscription statuses via [Stripe webhooks](https://stripe.com/docs/webhooks)

## Step-by-step setup

When deploying this template, the sequence of steps is important. Follow the steps below in order to get up and running.

### Configure Stripe

Next, we'll need to configure [Stripe](https://stripe.com/) to handle test payments. If you don't already have a Stripe account, create one now.

For the following steps, make sure you have the ["Test Mode" toggle](https://stripe.com/docs/testing) switched on.

#### Create a webhook

We need to create a webhook in the `Developers` section of Stripe.

1. Click the "Add Endpoint" button on the [test Endpoints page](https://dashboard.stripe.com/test/webhooks).
1. Enter your production deployment URL followed by `/api/webhooks` for the endpoint URL. (e.g. `https://your-deployment-url.vercel.app/api/webhooks`)
1. Click `Select events` under the `Select events to listen to` heading.
1. Click `Select all events` in the `Select events to send` section.
1. Copy `Signing secret` as we'll need that in the next step.

#### Create product and pricing information

Your application's webhook listens for product updates on Stripe and automatically propagates them to your Pocketbase database. So with your webhook listener running, you can now create your product and pricing information in the [Stripe Dashboard](https://dashboard.stripe.com/test/products).

Stripe Checkout currently supports pricing that bills a predefined amount at a specific interval. More complex plans (e.g., different pricing tiers or seats) are not yet supported.

For example, you can create business models with different pricing tiers, e.g.:

- Product 1: Hobby
  - Price 1: 10 USD per month
  - Price 2: 100 USD per year
- Product 2: Freelancer
  - Price 1: 20 USD per month
  - Price 2: 200 USD per year

Optionally, to speed up the setup, we have added a [fixtures file](stripe_bootstrap/stripe-fixtures.json) to bootstrap test product and pricing data in your Stripe account. The [Stripe CLI](https://stripe.com/docs/stripe-cli#install) `fixtures` command executes a series of API requests defined in this JSON file. Simply run `stripe fixtures stripe_bootstrap/stripe-fixtures.json`.

**Important:** Make sure that you've configured your Stripe webhook correctly and redeployed with all needed environment variables.

#### Configure the Stripe customer portal

1. Set your custom branding in the [settings](https://dashboard.stripe.com/settings/branding)
1. Configure the Customer Portal [settings](https://dashboard.stripe.com/test/settings/billing/portal)
1. Toggle on "Allow customers to update their payment methods"
1. Toggle on "Allow customers to update subscriptions"
1. Toggle on "Allow customers to cancel subscriptions"
1. Add the products and prices that you want
1. Set up the required business information and links

### Configure Pocketbase

1. Download this package
1. Run `go run main.go serve` from a command line in the root of the folder
1. Go to a webbrowser and browse to `https://127.0.0.1:8090/_/` and create new admin account and login
1. Click `Settings` on the left hand side bar and go to `Import Collections`
1. Click `Load from JSON file` and grab the schema file from `pb_bootstrap/pb_schema.json`
1. Exit the `go run main.go` command
1. Go to main.go in an IDE and search the file for `{YOUR_WEBHOOK_SECRET_HERE}` and replace this with your webhook secret which will look like `whsec_....`
1. Search the file for `{YOUR_STRIPE_SECRET_KEY_HERE}` and replace this with your stripe secret which will look like `sk_test....`
1. Re-run `go run main.go serve`
1. Configure your authentication settings (this is optional for testing but required for prod)
1. Finally you will need to host or provide a self-signed cert to use with stripe in dev or you will need to host **WEBHOOKS WILL NOT WORK WITHOUT HOSTING**

### Connect to Your Front End

1. You can add the pricing information and authentication to your front end app. You have a fully functioning backend subscription service that you can host and control.

### That's it

I know, that was quite a lot to get through, but it's worth it. You're now ready to earn recurring revenue from your customers. 🥳

### Use the Stripe CLI to test webhooks

[Install the Stripe CLI](https://stripe.com/docs/stripe-cli) and [link your Stripe account](https://stripe.com/docs/stripe-cli#login-account).

Next, start local webhook forwarding:

```bash
stripe listen --forward-to=127.0.0.1:8090/api/webhooks
```

Running this Stripe command will print a webhook secret (such as, `whsec_***`) to the console. Set `STRIPE_WEBHOOK_SECRET` to this value in your `.env.local` file.

Please note that stripe wont forward to http. You will need to ensure you are working in an environment where you have an SSL certificate installed

## Going live

### Archive testing products

Archive all test mode Stripe products before going live. Before creating your live mode products, make sure to follow the steps below to set up your live mode env vars and webhooks.

### Configure production environment variables

To run the project in live mode and process payments with Stripe, switch Stripe from "test mode" to "production mode." Your Stripe API keys will be different in production mode, and you will have to create a separate production mode webhook. Copy these values and paste them into Vercel, replacing the test mode values.

### Redeploy

Afterward, you will need to rebuild your production deployment for the changes to take effect. Within your project Dashboard, navigate to the "Deployments" tab, select the most recent deployment, click the overflow menu button (next to the "Visit" button) and select "Redeploy" (do NOT enable the "Use existing Build Cache" option).

To verify you are running in production mode, test checking out with the [Stripe test card](https://stripe.com/docs/testing). The test card should not work.

## A note on reliability

This template mirrors completed Stripe transactions to the Pocketbase database. This means that if the Pocketbase database is unavailable, the Stripe transaction will still succeed, but the Pocketbase database will not be updated, and the application will pass an error code back to Stripe. [By default](https://stripe.com/docs/webhooks/best-practices), Stripe will retry sending its response to the webhook for up to three days, or until the database update succeeds. This means that the Stripe transaction will eventually be reflected in the Pocketbase database as long as the database comes back online within three days. You may want to implement a process to automatically reconcile the Pocketbase database with Stripe in case of a prolonged outage.

## Inspiration and Possible Front End

This template is based on https://github.com/vercel/nextjs-subscription-payments/tree/main you could take the front end supplied there and adapt it to use PocketBase as a backend. Give it a try and submit a PR to this doc and I will add you as a contributor

## Note On The Docker Container

I have added a docker container that is not production ready but you can probably get running in google cloud run quite easily or whatever your preference is for hosting. Please note:

- You need to setup a mounted volume for the DB otherwise it will run in memory
- You need to setup your user collection permissions
- You need to deploy with run arguements

Here is my current build flow to upgrade and deploy the container to Google Artifact

### Update Packages

`go get -u github.com/pocketbase/pocketbase`

### Build Binary

`GOOS=linux GOARCH=amd64 go build -o bin/app-amd64-linux main.go`

### Build Image

`docker build -t myimage . --build-arg STRIPE_SECRET_KEY=sk_test_WHATEVER_YOUR_KEY_IS --build-arg HOST=api.sign365.com.au --build-arg STRIPE_RETURN_URL=https://sign365.com.au/account --build-arg PORT=8090 --build-arg DEVELOPMENT="" --platform linux/amd64`

### Tag Image

`docker tag myimage australia-southeast1-docker.pkg.dev/biz365-1569752078001/fastpocket/myimage:latest`

### Push Image

`docker push australia-southeast1-docker.pkg.dev/biz365-1569752078001/fastpocket/myimage`

I really hope this helps in building a fresh image.

## Sponsors

- [XAM Consulting](https://xam.com.au/)

## Contributors

- [Samuel Wyndham](https://twitter.com/meinbiz)
- [Suan Choi](https://github.com/suanTech)

## License

MIT License

Copyright (c) 2024 Sign365

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
