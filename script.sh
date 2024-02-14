#!/bin/bash

echo "STRIPE_SECRET_KEY = $STRIPE_SECRET_KEY"
echo "STRIPE_RETURN_URL = $STRIPE_RETURN_URL"
echo "HOST = $HOST"
echo "PORT = $PORT"
echo "DEVELOPMENT = $DEVELOPMENT"

# Serve Offline
if [[ -n "$DEVELOPMENT" ]]; then
    echo "Serving Offline..."
    stripe listen --print-secret --api-key "$STRIPE_SECRET_KEY" > secret.txt &
    wait $!
    nohup stripe listen --forward-to "http://0.0.0.0:8090/stripe" --api-key "$STRIPE_SECRET_KEY" --live > stripe.out 2>&1 &
    nohup ./bin/app-amd64-linux serve --http "0.0.0.0:8090"
# Serve Online
elif [[ -n "$HOST" && -n "$STRIPE_SECRET_KEY" && -n "$PORT" ]]; then
    echo "Serving Online..."
    nohup stripe listen --print-secret --api-key "$STRIPE_SECRET_KEY" > secret.txt &
    wait $!
    echo "WHSEC = $(<secret.txt)"
    nohup stripe listen --forward-to "https://$HOST/stripe" --api-key "$STRIPE_SECRET_KEY" --live > stripe.out 2>&1 &
    nohup ./bin/app-amd64-linux serve --http "0.0.0.0:8090"
# Error
else
    # Handle the case where the environment variables are not set
    echo "Environment variables HOST, STRIPE_SECRET_KEY, and PORT must be set or else you should use the local environment variable LOCAL to serve http on port 8090"
    exit 1
fi