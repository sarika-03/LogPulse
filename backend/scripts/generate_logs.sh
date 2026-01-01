#!/bin/bash

# generate_logs.sh - Generate fake logs for testing LokiLite

LOKILITE_URL="${LOKILITE_URL:-http://localhost:8080}"
COUNT="${1:-100}"

SERVICES=("api-gateway" "auth-service" "user-service" "payment-service" "notification-service")
ENVS=("prod" "staging" "dev")
LEVELS=("error" "warn" "info" "debug")

MESSAGES_ERROR=(
    "Connection timeout to database after 30s"
    "Failed to process payment: Invalid card number"
    "Authentication failed: Token expired"
    "Service unavailable: upstream connection refused"
    "Memory limit exceeded: OOMKilled"
)

MESSAGES_WARN=(
    "High memory usage detected: 85% utilized"
    "Slow query detected: 2.5s execution time"
    "Rate limit approaching: 950/1000 requests"
    "Certificate expires in 7 days"
)

MESSAGES_INFO=(
    "Request processed successfully in 45ms"
    "User logged in: user_id=12345"
    "Payment completed: amount=99.99"
    "Service started on port 8080"
    "Health check passed"
)

MESSAGES_DEBUG=(
    "Parsing request body: content-type=application/json"
    "Cache hit for key: user:12345:profile"
    "SQL query executed: SELECT * FROM users"
    "HTTP response: status=200, duration=12ms"
)

get_random_element() {
    local array=("$@")
    echo "${array[$RANDOM % ${#array[@]}]}"
}

get_message_for_level() {
    case $1 in
        error) get_random_element "${MESSAGES_ERROR[@]}" ;;
        warn)  get_random_element "${MESSAGES_WARN[@]}" ;;
        info)  get_random_element "${MESSAGES_INFO[@]}" ;;
        debug) get_random_element "${MESSAGES_DEBUG[@]}" ;;
    esac
}

echo "Generating $COUNT logs to $LOKILITE_URL..."

for i in $(seq 1 $COUNT); do
    SERVICE=$(get_random_element "${SERVICES[@]}")
    ENV=$(get_random_element "${ENVS[@]}")
    LEVEL=$(get_random_element "${LEVELS[@]}")
    MESSAGE=$(get_message_for_level $LEVEL)
    TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")

    PAYLOAD=$(cat <<EOF
{
    "streams": [{
        "labels": {
            "service": "$SERVICE",
            "env": "$ENV",
            "level": "$LEVEL"
        },
        "entries": [{
            "ts": "$TIMESTAMP",
            "line": "$MESSAGE"
        }]
    }]
}
EOF
)

    curl -s -X POST "$LOKILITE_URL/ingest" \
        -H "Content-Type: application/json" \
        -d "$PAYLOAD" > /dev/null

    if [ $((i % 10)) -eq 0 ]; then
        echo "Sent $i / $COUNT logs"
    fi
done

echo "Done! Generated $COUNT logs."
echo ""
echo "Test query:"
echo "curl '$LOKILITE_URL/query?query={service=\"api-gateway\"}&limit=10'"
