
# Use the latest golang base image
FROM golang:latest


# Set the Current Working Directory inside the container
WORKDIR /app
ARG STRIPE_SECRET_KEY=""
ARG STRIPE_RETURN_URL=""
ARG HOST=""
ARG PORT="443"
ARG DEVELOPMENT=""

# Set Environment Variables
ENV DEBIAN_FRONTEND=noninteractive
ENV STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
ENV STRIPE_RETURN_URL=${STRIPE_RETURN_URL}
ENV HOST=${HOST}
ENV PORT=${PORT}
ENV DEVELOPMENT=${DEVELOPMENT}

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Stripe Pipeline
RUN curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | tee /usr/share/keyrings/stripe.gpg
RUN echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | tee -a /etc/apt/sources.list.d/stripe.list
RUN apt-get update
RUN apt-get install stripe


# Copy the script to the container
COPY ./script.sh /script.sh

# Make the script executable
RUN chmod +x /script.sh

# Command to run the executable
CMD ["/script.sh"]

