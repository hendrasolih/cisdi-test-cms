#!/bin/bash

# Token sama untuk semua request
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6ImpvaG5fZG9lIiwicm9sZSI6IndyaXRlciIsImV4cCI6MTc1NDkwOTY3MywibmJmIjoxNzU0ODIzMjczLCJpYXQiOjE3NTQ4MjMyNzN9.x7S0JHSWWtvsOiuUt1cMKqF0oZS5euNUmsED603Au2c"

# Daftar tags (satu set tags per request)
TAGS_LIST=(
    '["javascript", "frontend", "react"]'
    '["go", "backend", "api"]'
    '["python", "data-science", "machine-learning"]'
    '["java", "spring", "microservices"]'
    '["rust", "systems", "performance"]'
)

# Loop setiap set tags
for ((i=0; i<${#TAGS_LIST[@]}; i++))
do
    echo "Request ke-$((i+1)) dengan tags: ${TAGS_LIST[$i]}"
    
    curl --silent --location 'http://localhost:8080/api/v1/articles' \
    --header 'Content-Type: application/json' \
    --header "Authorization: Bearer $TOKEN" \
    --data "{
        \"title\": \"Artikel ke-$((i+1))\",
        \"content\": \"<p>Konten artikel ke-$((i+1))...</p>\",
        \"tags\": ${TAGS_LIST[$i]}
    }"
    
    echo -e "\n---"
    sleep 1
done
