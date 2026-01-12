#!/bin/bash
# =============================================================================
# Blog API Test Script
# =============================================================================
# Run this script to test the Blog API endpoints
# Usage: ./test.sh [BASE_URL]
# =============================================================================

BASE_URL="${1:-http://localhost:8080}"

echo "============================================="
echo "Blog API Test Script"
echo "Base URL: $BASE_URL"
echo "============================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_header() {
    echo -e "${BLUE}>>> $1${NC}"
}

section_header() {
    echo ""
    echo -e "${YELLOW}=============================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}=============================================${NC}"
    echo ""
}

# Store tokens and IDs
ADMIN_TOKEN=""
EDITOR_TOKEN=""
AUTHOR_TOKEN=""
READER_TOKEN=""
CATEGORY_ID=""
POST_ID=""
COMMENT_ID=""

# =============================================================================
section_header "1. User Registration & Authentication"
# =============================================================================

# Register Admin
test_header "Register Admin User"
ADMIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@blog.com",
    "password": "adminpass123",
    "username": "admin",
    "display_name": "Blog Admin"
  }')
echo "$ADMIN_RESPONSE" | jq .
ADMIN_ID=$(echo "$ADMIN_RESPONSE" | jq -r '.id')
echo "Admin ID: $ADMIN_ID"
echo ""

# Register Editor
test_header "Register Editor User"
EDITOR_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "editor@blog.com",
    "password": "editorpass123",
    "username": "editor",
    "display_name": "Blog Editor"
  }')
echo "$EDITOR_RESPONSE" | jq .
echo ""

# Register Author
test_header "Register Author User"
AUTHOR_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "author@blog.com",
    "password": "authorpass123",
    "username": "authorjane",
    "display_name": "Jane Author"
  }')
echo "$AUTHOR_RESPONSE" | jq .
echo ""

# Register Reader
test_header "Register Reader User"
READER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "reader@blog.com",
    "password": "readerpass123",
    "username": "booklover",
    "display_name": "Book Lover"
  }')
echo "$READER_RESPONSE" | jq .
echo ""

# Login as Admin
test_header "Login as Admin"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@blog.com",
    "password": "adminpass123"
  }')
echo "$LOGIN_RESPONSE" | jq .
ADMIN_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
echo "Admin Token: ${ADMIN_TOKEN:0:50}..."
echo ""

# Login as Editor
test_header "Login as Editor"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "editor@blog.com", "password": "editorpass123"}')
EDITOR_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
echo "Editor Token: ${EDITOR_TOKEN:0:50}..."
echo ""

# Login as Author
test_header "Login as Author"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "author@blog.com", "password": "authorpass123"}')
AUTHOR_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
echo "Author Token: ${AUTHOR_TOKEN:0:50}..."
echo ""

# Login as Reader
test_header "Login as Reader"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "reader@blog.com", "password": "readerpass123"}')
READER_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
echo "Reader Token: ${READER_TOKEN:0:50}..."
echo ""

# =============================================================================
section_header "2. Category Management"
# =============================================================================

# Create parent category (Editor)
test_header "Create Category: Technology (Editor)"
CATEGORY_RESPONSE=$(curl -s -X POST "$BASE_URL/categories" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $EDITOR_TOKEN" \
  -d '{
    "name": "Technology",
    "slug": "tech",
    "description": "Technology news and tutorials"
  }')
echo "$CATEGORY_RESPONSE" | jq .
CATEGORY_ID=$(echo "$CATEGORY_RESPONSE" | jq -r '.id')
echo "Category ID: $CATEGORY_ID"
echo ""

# Create child category
test_header "Create Sub-Category: Programming (under Technology)"
SUBCATEGORY_RESPONSE=$(curl -s -X POST "$BASE_URL/categories" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $EDITOR_TOKEN" \
  -d "{
    \"name\": \"Programming\",
    \"slug\": \"programming\",
    \"description\": \"Programming tutorials and tips\",
    \"parent_id\": \"$CATEGORY_ID\"
  }")
echo "$SUBCATEGORY_RESPONSE" | jq .
echo ""

# List categories
test_header "List All Categories"
curl -s "$BASE_URL/categories" | jq .
echo ""

# Get category by slug
test_header "Get Category by Slug: tech"
curl -s "$BASE_URL/categories/tech" | jq .
echo ""

# =============================================================================
section_header "3. Post Management"
# =============================================================================

# Create a draft post (Author)
test_header "Create Draft Post (Author)"
POST_RESPONSE=$(curl -s -X POST "$BASE_URL/posts" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AUTHOR_TOKEN" \
  -d "{
    \"title\": \"Introduction to CodeAI DSL\",
    \"slug\": \"intro-codeai-dsl\",
    \"excerpt\": \"Learn the basics of CodeAI DSL for building APIs\",
    \"content\": \"CodeAI DSL is a domain-specific language designed for defining backend APIs. It provides a declarative way to specify entities, endpoints, and business logic.\",
    \"category_id\": \"$CATEGORY_ID\",
    \"tags\": [\"codeai\", \"dsl\", \"tutorial\", \"backend\"],
    \"status\": \"draft\",
    \"meta_title\": \"CodeAI DSL Tutorial\",
    \"meta_description\": \"A comprehensive guide to CodeAI DSL\"
  }")
echo "$POST_RESPONSE" | jq .
POST_ID=$(echo "$POST_RESPONSE" | jq -r '.id')
POST_SLUG=$(echo "$POST_RESPONSE" | jq -r '.slug')
echo "Post ID: $POST_ID"
echo "Post Slug: $POST_SLUG"
echo ""

# List author's posts (including drafts)
test_header "List Author's Own Posts (includes drafts)"
curl -s "$BASE_URL/me/posts" \
  -H "Authorization: Bearer $AUTHOR_TOKEN" | jq .
echo ""

# Try to list public posts (should be empty - post is draft)
test_header "List Public Posts (should not include draft)"
curl -s "$BASE_URL/posts" | jq .
echo ""

# Publish the post (Editor)
test_header "Publish Post (Editor)"
curl -s -X POST "$BASE_URL/posts/$POST_ID/publish" \
  -H "Authorization: Bearer $EDITOR_TOKEN" | jq .
echo ""

# Now list public posts (should include published post)
test_header "List Public Posts (now includes published)"
curl -s "$BASE_URL/posts" | jq .
echo ""

# Get post by slug
test_header "Get Post by Slug"
curl -s "$BASE_URL/posts/$POST_SLUG" | jq .
echo ""

# Search posts
test_header "Search Posts for 'CodeAI'"
curl -s "$BASE_URL/posts?search=CodeAI" | jq .
echo ""

# Filter by tag
test_header "Filter Posts by Tag: tutorial"
curl -s "$BASE_URL/posts?tag=tutorial" | jq .
echo ""

# Like the post (Reader)
test_header "Like Post (Reader)"
curl -s -X POST "$BASE_URL/posts/$POST_ID/like" \
  -H "Authorization: Bearer $READER_TOKEN" | jq '.like_count'
echo ""

# =============================================================================
section_header "4. Comment System"
# =============================================================================

# Add comment (Reader - authenticated)
test_header "Add Comment (Authenticated Reader)"
COMMENT_RESPONSE=$(curl -s -X POST "$BASE_URL/posts/$POST_ID/comments" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $READER_TOKEN" \
  -d '{
    "content": "Great introduction! Very clear and helpful."
  }')
echo "$COMMENT_RESPONSE" | jq .
COMMENT_ID=$(echo "$COMMENT_RESPONSE" | jq -r '.id')
echo "Comment ID: $COMMENT_ID"
echo ""

# Reply to comment (Author)
test_header "Reply to Comment (Author)"
REPLY_RESPONSE=$(curl -s -X POST "$BASE_URL/posts/$POST_ID/comments" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AUTHOR_TOKEN" \
  -d "{
    \"content\": \"Thank you! Glad you found it useful.\",
    \"parent_id\": \"$COMMENT_ID\"
  }")
echo "$REPLY_RESPONSE" | jq .
echo ""

# Add guest comment (requires moderation)
test_header "Add Guest Comment (Requires Moderation)"
GUEST_COMMENT=$(curl -s -X POST "$BASE_URL/posts/$POST_ID/comments/guest" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "This is a guest comment",
    "guest_name": "Anonymous Visitor",
    "guest_email": "visitor@example.com"
  }')
echo "$GUEST_COMMENT" | jq .
GUEST_COMMENT_ID=$(echo "$GUEST_COMMENT" | jq -r '.id')
echo "Guest Comment ID (pending): $GUEST_COMMENT_ID"
echo ""

# List pending comments (Editor)
test_header "List Pending Comments for Moderation (Editor)"
curl -s "$BASE_URL/admin/comments/pending" \
  -H "Authorization: Bearer $EDITOR_TOKEN" | jq .
echo ""

# Approve guest comment (Editor)
test_header "Approve Guest Comment (Editor)"
curl -s -X POST "$BASE_URL/comments/$GUEST_COMMENT_ID/moderate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $EDITOR_TOKEN" \
  -d '{"status": "approved"}' | jq .
echo ""

# List all approved comments for the post
test_header "List Approved Comments for Post"
curl -s "$BASE_URL/posts/$POST_ID/comments" | jq .
echo ""

# =============================================================================
section_header "5. User Management (Admin)"
# =============================================================================

# List all users (Admin only)
test_header "List All Users (Admin)"
curl -s "$BASE_URL/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
echo ""

# Filter users by role
test_header "List Users with Role: author"
curl -s "$BASE_URL/users?role=author" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
echo ""

# Get user public profile
test_header "Get User Public Profile"
curl -s "$BASE_URL/users/$ADMIN_ID" | jq .
echo ""

# =============================================================================
section_header "6. Update Operations"
# =============================================================================

# Update post (Author)
test_header "Update Post Content (Author)"
curl -s -X PUT "$BASE_URL/posts/$POST_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AUTHOR_TOKEN" \
  -d '{
    "title": "Introduction to CodeAI DSL (Updated)",
    "tags": ["codeai", "dsl", "tutorial", "backend", "api"]
  }' | jq '.title, .tags'
echo ""

# Update user profile
test_header "Update User Profile"
curl -s -X PUT "$BASE_URL/auth/me" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $READER_TOKEN" \
  -d '{
    "display_name": "Avid Book Lover",
    "bio": "I love reading about technology and programming."
  }' | jq '.display_name, .bio'
echo ""

# Update comment (Owner)
test_header "Update Own Comment"
curl -s -X PUT "$BASE_URL/comments/$COMMENT_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $READER_TOKEN" \
  -d '{
    "content": "Great introduction! Very clear and helpful. Looking forward to more!"
  }' | jq '.content'
echo ""

# =============================================================================
section_header "7. Cleanup (Delete Operations)"
# =============================================================================

# Delete comment (soft delete)
test_header "Delete Comment"
curl -s -X DELETE "$BASE_URL/comments/$GUEST_COMMENT_ID" \
  -H "Authorization: Bearer $EDITOR_TOKEN"
echo "Guest comment deleted"
echo ""

# Delete category (Admin only, soft delete)
test_header "Delete Category (Admin)"
curl -s -X DELETE "$BASE_URL/categories/$CATEGORY_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
echo "Category soft-deleted"
echo ""

# Delete post (soft delete)
test_header "Delete Post (Author)"
curl -s -X DELETE "$BASE_URL/posts/$POST_ID" \
  -H "Authorization: Bearer $AUTHOR_TOKEN"
echo "Post soft-deleted"
echo ""

# =============================================================================
section_header "Test Complete!"
# =============================================================================

echo "Summary:"
echo "  - Users created: admin, editor, author, reader"
echo "  - Categories created: Technology, Programming"
echo "  - Posts created: Introduction to CodeAI DSL"
echo "  - Comments: 3 (1 authenticated, 1 reply, 1 guest)"
echo ""
echo "Note: Delete operations use soft delete - data is preserved"
echo "      with a deleted_at timestamp for audit purposes."
