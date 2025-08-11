# CMS CISDI Test

Aplikasi Content Management System (CMS) berbasis REST API menggunakan Go, Gin, dan GORM. Mendukung manajemen artikel dengan multi-versi, sistem tagging, dan autentikasi JWT.

## 🚀 Fitur Utama

- **Autentikasi & Otorisasi**: Registrasi, login dengan JWT, role-based access control
- **Manajemen Artikel**: CRUD artikel dengan sistem scoring hubungan artikel-tag
- **Sistem Versi**: Multi-version artikel dengan status (draft/published/archived)
- **Manajemen Tag**: CRUD tag dengan trending score
- **Advanced Features**: Filtering, sorting, paginasi artikel
- **Role Management**: Akses berbasis role (admin, editor, writer)
- **Middleware**: Autentikasi dan otorisasi otomatis
- **Response Helper**: JSON response yang konsisten

## 📁 Struktur Folder

```
cms-cisdi/
├── config/                # Konfigurasi database dan JWT
├── handlers/              # HTTP handlers (controllers)
├── middleware/            # Middleware autentikasi dan otorisasi
├── models/                # Models dan Data Transfer Objects
├── repositories/          # Database query layer
├── services/              # Business logic layer
├── tests/                 # Integration dan unit tests
├── main.go                # Entry point aplikasi
├── docker-compose.yml     # Docker compose configuration
├── .env.example           # Template environment variables
├── .env                   # Environment variables (local)
└── .gitignore             # Git ignore rules
```

## 🛠️ Instalasi & Setup

### Metode 1: Local Development

1. **Clone repository:**
   ```bash
   git clone <repo-url>
   cd cms-cisdi/cisdi-test-cms
   ```

2. **Setup environment:**
   ```bash
   cp .env.example .env
   # Edit .env dengan konfigurasi database dan JWT Anda
   ```

3. **Install dependencies:**
   ```bash
   go mod tidy
   ```

4. **Jalankan aplikasi:**
   ```bash
   go run main.go
   ```

### Metode 2: Docker

```bash
docker compose up -d --build
```

Aplikasi akan berjalan di `http://localhost:8080`

## 🔗 Endpoint API

### Autentikasi
| Method | Endpoint | Deskripsi | Auth Required |
|--------|----------|-----------|---------------|
| `POST` | `/api/v1/auth/register` | Registrasi user baru | ❌ |
| `POST` | `/api/v1/auth/login` | Login user | ❌ |

### Artikel Management (Protected)
| Method | Endpoint | Deskripsi | Auth Required |
|--------|----------|-----------|---------------|
| `GET` | `/api/v1/articles` | List artikel dengan filter & paginasi | ✅ |
| `POST` | `/api/v1/articles` | Buat artikel baru | ✅ |
| `GET` | `/api/v1/articles/:id` | Detail artikel | ✅ |
| `DELETE` | `/api/v1/articles/:id` | Hapus artikel | ✅ |
| `POST` | `/api/v1/articles/:id/versions` | Tambah versi artikel | ✅ |
| `PUT` | `/api/v1/articles/:id/versions/:version_id/status` | Update status versi | ✅ |
| `GET` | `/api/v1/articles/:id/versions` | List versi artikel | ✅ |
| `GET` | `/api/v1/articles/:id/versions/:version_id` | Detail versi artikel | ✅ |

### Tag Management (Protected)
| Method | Endpoint | Deskripsi | Auth Required |
|--------|----------|-----------|---------------|
| `GET` | `/api/v1/tags` | List semua tag | ✅ |
| `POST` | `/api/v1/tags` | Buat tag baru | ✅ |
| `GET` | `/api/v1/tags/:id` | Detail tag | ✅ |

### Public API
| Method | Endpoint | Deskripsi | Auth Required |
|--------|----------|-----------|---------------|
| `GET` | `/api/v1/public/articles` | List artikel published | ❌ |
| `GET` | `/api/v1/public/articles/:id` | Detail artikel published | ❌ |

## 📝 Contoh Penggunaan

### Registrasi User
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "user1",
    "email": "user1@mail.com",
    "password": "pass123"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user1@mail.com",
    "password": "pass123"
  }'
```

### Get Articles dengan Filter
```bash
curl -X GET "http://localhost:8080/api/v1/articles?status=published&page=1&limit=10&sort_by=created_at&sort_order=desc" \
  -H "Authorization: Bearer <jwt_token>"
```

### Buat Artikel Baru
```bash
curl -X POST http://localhost:8080/api/v1/articles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt_token>" \
  -d '{
    "title": "Artikel Baru",
    "content": "Konten artikel...",
    "status": "draft",
    "tag_ids": [1, 2, 3]
  }'
```

## 🧪 Testing

Aplikasi dilengkapi dengan integration test suite yang komprehensif.

### Menjalankan Semua Test
```bash
go test -v -run ^TestIntegrationSuite/TestIntegrationSuite$
```

### Menjalankan Test Spesifik
```bash
# Test artikel-tag relationship scoring
go test -v -run ^TestIntegrationSuite/TestArticleTagRelationshipScore

# Test autentikasi
go test -v -run ^TestIntegrationSuite/TestAuthFlow

# Test manajemen artikel
go test -v -run ^TestIntegrationSuite/TestCreateAndGetArticle

# Test sistem versi
go test -v -run ^TestIntegrationSuite/TestArticleVersioning
```

### Test Coverage
```bash
go test -cover ./...
```

### Test Suite Features
- **Integration Tests**: End-to-end testing dari API endpoints
- **Authentication Testing**: Test flow register, login, dan JWT validation
- **Business Logic Testing**: Test artikel scoring, tag relationships
- **Database Testing**: Test database operations dan constraints
- **Permission Testing**: Test role-based access control

## 👥 Role & Akses Control

| Role | Artikel | Versi | Tag | Akses |
|------|---------|-------|-----|-------|
| **Admin** | Semua artikel | Semua versi | Semua tag | Full access |
| **Editor** | Semua artikel | Semua versi | Semua tag | Full access |
| **Writer** | Artikel sendiri | Versi artikel sendiri | Read-only | Limited |
| **User** | Artikel published | Versi published | Read-only | Read-only |

## 🏗️ Arsitektur

Aplikasi menggunakan arsitektur **Clean Architecture** dengan pemisahan layer:

1. **Handlers Layer**: HTTP request/response handling
2. **Services Layer**: Business logic dan validation
3. **Repositories Layer**: Database operations
4. **Models Layer**: Data structures dan DTOs

## 📊 Fitur Scoring

Sistem scoring hubungan artikel-tag (`article_tag_relationship_score`) yang menghitung:
- Relevansi tag terhadap artikel
- Trending score tag
- Engagement metrics

## 🔧 Environment Variables

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password
DB_NAME=cms_cisdi

# JWT
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRES_IN=24h

# Server
SERVER_PORT=8080
SERVER_HOST=localhost
```

## 🤝 Kontribusi

1. Fork repository
2. Buat feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push ke branch (`git push origin feature/amazing-feature`)
5. Buat Pull Request

## 📄 Lisensi

MIT License - lihat file [LICENSE](LICENSE) untuk detail.

---

**Dibuat dengan ❤️ menggunakan Go, Gin, dan GORM**