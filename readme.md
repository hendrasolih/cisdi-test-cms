# CMS CISDI Test

Aplikasi Content Management System (CMS) berbasis REST API menggunakan Go, Gin, dan GORM. Mendukung manajemen artikel, versi, tag, dan autentikasi JWT.

## Fitur Utama
- Registrasi & Login user (JWT)
- Manajemen artikel (CRUD)
- Versi artikel (multi-version, status: draft/published/archived)
- Tag artikel (CRUD, trending score)
- Filtering, sorting, dan paginasi artikel
- Role-based access (admin, editor, user)
- Middleware autentikasi dan otorisasi
- Helper response JSON konsisten

## Struktur Folder
```
├── config/                # Konfigurasi DB, JWT
├── handlers/              # HTTP handler (controller)
├── middleware/            # Middleware (auth, role)
├── models/                # Model & DTO
├── repositories/          # Query DB
├── services/              # Bisnis logic
├── main.go                # Entry point
├── .env                   # Environment variable
├── .gitignore             # File/folder yang diabaikan git
```

## Instalasi & Menjalankan
1. Clone repo:
   ```bash
   git clone <repo-url>
   cd cms-cisdi/cisdi-test-cms
   ```
2. Copy `.env.example` ke `.env` dan isi konfigurasi DB/JWT.
3. Install dependency:
   ```bash
   go mod tidy
   ```
4. Jalankan migrasi & server:
   ```bash
   go run main.go
   ```

## Endpoint API
### Auth
- `POST /api/v1/auth/register` — Registrasi user
- `POST /api/v1/auth/login` — Login user

### Artikel (protected)
- `GET /api/v1/articles` — List artikel (filter, sort, paginasi)
- `POST /api/v1/articles` — Buat artikel
- `GET /api/v1/articles/:id` — Detail artikel
- `DELETE /api/v1/articles/:id` — Hapus artikel
- `POST /api/v1/articles/:id/versions` — Tambah versi artikel
- `PUT /api/v1/articles/:id/versions/:version_id/status` — Update status versi
- `GET /api/v1/articles/:id/versions` — List versi artikel
- `GET /api/v1/articles/:id/versions/:version_id` — Detail versi artikel

### Tag (protected)
- `GET /api/v1/tags` — List tag
- `POST /api/v1/tags` — Buat tag
- `GET /api/v1/tags/:id` — Detail tag

### Artikel Publik
- `GET /api/v1/public/articles` — List artikel published
- `GET /api/v1/public/articles/:id` — Detail artikel published

## Contoh Curl
```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register -H "Content-Type: application/json" -d '{"username":"user1","email":"user1@mail.com","password":"pass123"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d '{"email":"user1@mail.com","password":"pass123"}'

# Get Articles (protected)
curl -X GET "http://localhost:8080/api/v1/articles?status=published&page=1&limit=10&sort_by=created_at&sort_order=desc" -H "Authorization: Bearer <jwt_token>"
```

## Role & Akses
- **admin/editor**: akses semua artikel, versi, tag
- **user**: hanya artikel sendiri & published

## Kontribusi
1. Fork repo
2. Buat branch fitur
3. Pull request

## Lisensi
MIT
