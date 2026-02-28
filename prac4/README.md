# Practice 4 (Docker + Compose + Postgres)

## Run
```bash
docker compose up --build
```

API will be on: http://localhost:8080

## Test quickly (curl)
Health:
```bash
curl http://localhost:8080/health
```

List movies:
```bash
curl http://localhost:8080/movies
```

Create:
```bash
curl -X POST http://localhost:8080/movies \
  -H "Content-Type: application/json" \
  -d '{"title":"Interstellar"}'
```

Update:
```bash
curl -X PUT http://localhost:8080/movies/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated title"}'
```

Delete:
```bash
curl -X DELETE http://localhost:8080/movies/1
```
