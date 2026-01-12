# Blog API Example

A complete blog platform API demonstrating entity relationships, authentication, and role-based access control.

## Overview

This example demonstrates:
- Multiple entities with relationships (User, Post, Comment, Category)
- Foreign key references (`ref(Entity)`)
- Role-based access control (RBAC)
- Soft delete for data preservation
- Full-text search on content
- Comment threading and moderation
- Event publishing for integrations

## File Structure

```
02-blog-api/
├── blog-api.cai    # Main DSL file
├── README.md          # This file
└── test.sh            # Sample curl commands for testing
```

## Entity Relationships

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│   Category  │◀──────│    Post     │──────▶│    User     │
│             │       │             │       │  (author)   │
└─────────────┘       └─────────────┘       └─────────────┘
      │                      │
      │ (parent)             │
      ▼                      ▼
┌─────────────┐       ┌─────────────┐
│  Category   │       │   Comment   │──────▶ User (optional)
│  (child)    │       │             │
└─────────────┘       └─────────────┘
                            │
                            │ (parent)
                            ▼
                      ┌─────────────┐
                      │   Comment   │
                      │  (reply)    │
                      └─────────────┘
```

## Entities

### User

| Field | Type | Description |
|-------|------|-------------|
| `id` | uuid | Primary key |
| `email` | string | Unique email (login) |
| `username` | string | Unique username |
| `display_name` | string | Public display name |
| `role` | enum | reader, author, editor, admin |
| `active` | boolean | Account status |

**Roles:**
- `reader` - Can read posts, comment
- `author` - Can create/edit own posts
- `editor` - Can edit any post, moderate comments
- `admin` - Full access

### Category

| Field | Type | Description |
|-------|------|-------------|
| `id` | uuid | Primary key |
| `name` | string | Category name |
| `slug` | string | URL-friendly identifier |
| `parent_id` | ref(Category) | Parent category (hierarchical) |

### Post

| Field | Type | Description |
|-------|------|-------------|
| `id` | uuid | Primary key |
| `title` | string | Post title (searchable) |
| `slug` | string | URL-friendly identifier |
| `content` | text | Post content (searchable) |
| `author_id` | ref(User) | Post author |
| `category_id` | ref(Category) | Post category |
| `tags` | list(string) | Post tags |
| `status` | enum | draft, pending_review, published, archived |

### Comment

| Field | Type | Description |
|-------|------|-------------|
| `id` | uuid | Primary key |
| `post_id` | ref(Post) | Related post |
| `author_id` | ref(User) | Comment author (optional for guests) |
| `parent_id` | ref(Comment) | Parent comment (for threading) |
| `status` | enum | pending, approved, spam, rejected |

## API Endpoints

### Authentication

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | `/auth/register` | Register new user | None |
| POST | `/auth/login` | Login, get JWT | None |
| GET | `/auth/me` | Get current user | Required |
| PUT | `/auth/me` | Update profile | Required |

### Users

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/users` | List all users | Admin |
| GET | `/users/{id}` | Get user profile | Optional |

### Categories

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/categories` | List categories | Optional |
| GET | `/categories/{slug}` | Get by slug | Optional |
| POST | `/categories` | Create category | Editor+ |
| PUT | `/categories/{id}` | Update category | Editor+ |
| DELETE | `/categories/{id}` | Delete category | Admin |

### Posts

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/posts` | List published posts | Optional |
| GET | `/posts/{slug}` | Get post by slug | Optional |
| POST | `/posts` | Create post | Author+ |
| PUT | `/posts/{id}` | Update post | Author+ |
| DELETE | `/posts/{id}` | Delete post | Author+ |
| POST | `/posts/{id}/publish` | Publish draft | Editor+ |
| POST | `/posts/{id}/like` | Like post | Required |
| GET | `/me/posts` | List own posts | Author+ |

### Comments

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/posts/{id}/comments` | List comments | Optional |
| POST | `/posts/{id}/comments` | Add comment | Required |
| POST | `/posts/{id}/comments/guest` | Guest comment | None |
| PUT | `/comments/{id}` | Update comment | Owner |
| DELETE | `/comments/{id}` | Delete comment | Owner/Mod |
| POST | `/comments/{id}/moderate` | Moderate comment | Editor+ |
| GET | `/admin/comments/pending` | List pending | Editor+ |

## Step-by-Step Instructions

### 1. Generate the API

```bash
codeai generate examples/02-blog-api/blog-api.cai
```

### 2. Configure Environment

```bash
export DATABASE_URL="postgres://localhost:5432/blog"
export REDIS_URL="redis://localhost:6379"
export JWT_ISSUER="blog-api"
export JWT_SECRET="your-secret-key"
```

### 3. Run Migrations

```bash
codeai migrate up
```

### 4. Start the Server

```bash
codeai run
```

## Sample Requests

### Register a User

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword",
    "username": "johndoe",
    "display_name": "John Doe"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securepassword"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "...",
    "username": "johndoe",
    "role": "reader"
  },
  "expires_at": "2026-01-13T10:00:00Z"
}
```

### Create a Category (Editor+)

```bash
curl -X POST http://localhost:8080/categories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <editor-token>" \
  -d '{
    "name": "Technology",
    "slug": "tech",
    "description": "Tech news and tutorials"
  }'
```

### Create a Post (Author+)

```bash
curl -X POST http://localhost:8080/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <author-token>" \
  -d '{
    "title": "Getting Started with CodeAI",
    "slug": "getting-started-codeai",
    "excerpt": "Learn how to build APIs with CodeAI DSL",
    "content": "CodeAI is a powerful DSL for generating backend APIs...",
    "category_id": "category-uuid",
    "tags": ["codeai", "tutorial", "backend"],
    "status": "draft"
  }'
```

### Publish a Post (Editor+)

```bash
curl -X POST http://localhost:8080/posts/<post-id>/publish \
  -H "Authorization: Bearer <editor-token>"
```

### List Published Posts

```bash
# All published posts
curl http://localhost:8080/posts

# Filter by category
curl "http://localhost:8080/posts?category=tech"

# Search posts
curl "http://localhost:8080/posts?search=CodeAI"

# Filter by tag
curl "http://localhost:8080/posts?tag=tutorial"
```

### Add a Comment

```bash
# Authenticated comment
curl -X POST http://localhost:8080/posts/<post-id>/comments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "content": "Great article! Very helpful."
  }'

# Reply to comment
curl -X POST http://localhost:8080/posts/<post-id>/comments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "content": "Thanks for the feedback!",
    "parent_id": "<parent-comment-id>"
  }'
```

### Guest Comment

```bash
curl -X POST http://localhost:8080/posts/<post-id>/comments/guest \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Nice post!",
    "guest_name": "Visitor",
    "guest_email": "visitor@example.com"
  }'
```

### Moderate Comments (Editor+)

```bash
# Approve comment
curl -X POST http://localhost:8080/comments/<id>/moderate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <editor-token>" \
  -d '{"status": "approved"}'

# Mark as spam
curl -X POST http://localhost:8080/comments/<id>/moderate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <editor-token>" \
  -d '{"status": "spam"}'
```

## Key Concepts Demonstrated

1. **Entity Relationships**:
   - `ref(Entity)` for foreign keys
   - Self-referencing (Category parent, Comment threads)
   - Optional vs required relationships

2. **Role-Based Access Control**:
   - `roles: [admin, editor]` on endpoints
   - Different access levels per role

3. **Soft Delete**:
   - `soft_delete` modifier preserves data
   - `deleted_at` timestamp for audit

4. **Full-Text Search**:
   - `searchable` modifier on title/content
   - Search query parameter on list endpoints

5. **Enum Types**:
   - Post status workflow
   - Comment moderation states
   - User roles

6. **Caching**:
   - Redis cache configuration
   - TTL and prefix settings

7. **Events**:
   - Event emission on state changes
   - Webhook and Kafka publishing

## Next Steps

- See [E-commerce](../03-ecommerce/) for workflows and sagas
- Check [Integrations](../04-integrations/) for external API patterns
- Try [Scheduled Jobs](../05-scheduled-jobs/) for background tasks
