# Validation Package Design

A Laravel-inspired, type-safe validation package for Go. No struct tags, no reflection.

## Core Concept

Validation rules are defined as a function that receives a `Validator` and returns a typed struct. The function acts as both schema definition and deserialization — if validation passes, you get a fully populated, correctly typed result.

```go
func CreateUserRules(v val.Validator) CreateUserRequest {
    return CreateUserRequest{
        Name:  v.String("name", val.Required(), val.Min(3), val.Max(255)),
        Email: v.String("email", val.Required(), val.Email()),
        Age:   v.Int("age", val.Required(), val.IntMin(18)),
    }
}
```

### Handler Usage

```go
func CreateUser(r *request.Request) response.Response {
    user, err := val.Validate(r, CreateUserRules)
    if err != nil {
        return response.ValidationError(err)
    }

    // user is CreateUserRequest — fully typed, validated
    db.Create(user)
}
```

### Entry Point

```go
func Validate[T any](source Source, rules func(Validator) T) (T, *ValidationError)
```

`Source` is an interface that provides the raw `map[string]any` from JSON, query params, URI, etc.

---

## Validator Interface

```go
type Validator interface {
    // Primitives
    String(key string, rules ...Rule[string]) string
    Int(key string, rules ...Rule[int]) int
    Float(key string, rules ...Rule[float64]) float64
    Bool(key string, rules ...Rule[bool]) bool
    Date(key string, rules ...Rule[time.Time]) time.Time

    // Primitive arrays
    Strings(key string, rules ...ArrayRule[string]) []string
    Ints(key string, rules ...ArrayRule[int]) []int

    // Nullable — returns pointer (nil if absent)
    Nullable() NullableValidator
}

type NullableValidator interface {
    String(key string, rules ...Rule[string]) *string
    Int(key string, rules ...Rule[int]) *int
    Float(key string, rules ...Rule[float64]) *float64
    Bool(key string, rules ...Rule[bool]) *bool
    Date(key string, rules ...Rule[time.Time]) *time.Time
}
```

---

## Rule Interface

```go
type Rule[T any] interface {
    Validate(value T, ctx *Context) error
    Message() string
}
```

The `Context` gives rules access to the full validation state for cross-field validation (see below).

---

## Validation Errors

Errors are returned as a map of field → list of messages, identical to Laravel:

```go
type ValidationError struct {
    Message string
    Errors  map[string][]string
}
```

```json
{
    "message": "The name field must be at least 3 characters long and 2 more errors",
    "errors": {
        "name": ["The name field must be at least 3 characters long"],
        "email": ["The email field must be a valid email address"],
        "address.city": ["The address city field is required"]
    }
}
```

All errors are collected — validation does **not** stop at the first failure.

---

## Features

### 1. Dot Notation (Nested Objects)

Nested fields use dot paths, just like Laravel. No separate `Sub()` method needed.

**Laravel:**
```php
'address.street' => 'required|string|max:255',
'address.city'   => 'required|string',
```

**Go:**
```go
func Rules(v val.Validator) CreateUserRequest {
    return CreateUserRequest{
        Name: v.String("name", val.Required()),
        Address: Address{
            Street: v.String("address.street", val.Required(), val.Max(255)),
            City:   v.String("address.city", val.Required()),
            Zip:    v.String("address.zip", val.Required(), val.Regex(`^\d{4}[A-Z]{2}$`)),
        },
    }
}
```

Internally, `"address.street"` splits on `.` and walks the nested `map[string]any`. Errors are keyed as `address.street`.

---

### 2. Primitive Arrays (`tags.*`)

**Laravel:**
```php
'tags'   => 'required|array|min:1',
'tags.*' => 'string|max:50',
```

**Go:**
```go
Tags: v.Strings("tags", val.Required(), val.ArrayMin(1), val.Each(val.Max(50))),
```

- `v.Strings()` returns `[]string`, `v.Ints()` returns `[]int`
- `val.Each()` applies element-level rules to every item
- `val.ArrayMin()` / `val.ArrayMax()` validate array length
- Errors are keyed per element: `tags.0`, `tags.1`, etc.

---

### 3. Object Arrays (`items.*`)

**Laravel:**
```php
'items'        => 'required|array',
'items.*.name' => 'required|string',
'items.*.price' => 'required|integer|min:0',
```

**Go:**
```go
Items: val.Array(v, "items", val.Required(), func(v val.Validator) Item {
    return Item{
        Name:  v.String("name", val.Required()),
        Price: v.Int("price", val.Required(), val.IntMin(0)),
    }
}),
```

`val.Array` is a generic free function (Go doesn't allow generic methods on interfaces):

```go
func Array[T any](v Validator, key string, rules ...any) []T
// The last argument that is a func(Validator) T is used as the element rules function.
// Other arguments are array-level rules (Required, ArrayMin, etc.)
```

Each element's rules function receives a scoped validator. Errors are keyed as `items.0.name`, `items.1.price`, etc.

---

### 4. Nullable Fields

**Laravel:**
```php
'bio' => 'nullable|string|max:1000',
```

**Go:**
```go
Bio: v.Nullable().String("bio", val.Max(1000)),
```

- Returns `*string` (or `*int`, `*time.Time`, etc.)
- `nil` if the field is absent or `null`
- Validators are only applied if a value is present
- No error if missing (unlike `Required()` which errors on absence)

---

### 5. Cross-Field Validation

**Laravel:**
```php
'start_date' => 'required|date|before:end_date',
'end_date'   => 'required|date',
```

**Go:**
```go
StartDate: v.Date("start_date", val.Required(), val.Before("end_date")),
EndDate:   v.Date("end_date", val.Required()),
```

Cross-field rules like `val.Before("end_date")` register a deferred hook via `Context`. All hooks run after all fields have been processed, so field ordering in the rules function doesn't matter.

Internally:

```go
func (b *before) Validate(value time.Time, ctx *Context) error {
    ctx.OnResolved(b.otherKey, func(other any) error {
        otherTime, ok := other.(time.Time)
        if !ok {
            return fmt.Errorf("field %s is not a date", b.otherKey)
        }
        if !value.Before(otherTime) {
            return fmt.Errorf("must be before %s", b.otherKey)
        }
        return nil
    })
    return nil
}
```

Errors from cross-field hooks are attributed to the field that registered them (`start_date`, not `end_date`).

---

### 6. Optional vs Required vs Nullable

| Syntax | Absent | Null | Present |
|---|---|---|---|
| `v.String("x", val.Required())` | error | error | validates & returns value |
| `v.String("x")` | returns `""` | returns `""` | validates & returns value |
| `v.Nullable().String("x")` | returns `nil` | returns `nil` | validates & returns `*string` |

---

## Built-in Rules

### String Rules

| Rule | Laravel Equivalent | Description |
|---|---|---|
| `val.Required()` | `required` | Field must be present and non-empty |
| `val.Min(n)` | `min:n` | Minimum length |
| `val.Max(n)` | `max:n` | Maximum length |
| `val.Email()` | `email` | Must be valid email |
| `val.URL()` | `url` | Must be valid URL |
| `val.UUID()` | `uuid` | Must be valid UUID |
| `val.In("a", "b")` | `in:a,b` | Must be one of the values |
| `val.Regex(pattern)` | `regex:pattern` | Must match regex |
| `val.Alpha()` | `alpha` | Only alphabetic characters |
| `val.AlphaNum()` | `alpha_num` | Only alphanumeric characters |
| `val.Username()` | — | Custom: `^[a-zA-Z0-9._-]{1,255}$` |
| `val.ExternalURL()` | — | Custom: must start with `https://` |

### Number Rules

| Rule | Laravel Equivalent | Description |
|---|---|---|
| `val.Required()` | `required` | Field must be present |
| `val.IntMin(n)` | `min:n` | Minimum value |
| `val.IntMax(n)` | `max:n` | Maximum value |
| `val.Between(a, b)` | `between:a,b` | Value in range |

### Date Rules

| Rule | Laravel Equivalent | Description |
|---|---|---|
| `val.Required()` | `required` | Field must be present |
| `val.Before(field)` | `before:field` | Must be before another date field |
| `val.After(field)` | `after:field` | Must be after another date field |
| `val.BeforeNow()` | `before:now` | Must be in the past |
| `val.AfterNow()` | `after:now` | Must be in the future |

### Array Rules

| Rule | Laravel Equivalent | Description |
|---|---|---|
| `val.Required()` | `required` | Array must be present |
| `val.ArrayMin(n)` | `min:n` | Minimum number of items |
| `val.ArrayMax(n)` | `max:n` | Maximum number of items |
| `val.Each(rules...)` | `*` | Apply rules to each element |

---

## Custom Rules

Implement the `Rule[T]` interface:

```go
type lowercase struct{}

func Lowercase() Rule[string] { return &lowercase{} }

func (l *lowercase) Validate(value string, ctx *Context) error {
    if value != strings.ToLower(value) {
        return fmt.Errorf("must be lowercase")
    }
    return nil
}

func (l *lowercase) Message() string {
    return "The {field} field must be lowercase"
}
```

Usage:

```go
Slug: v.String("slug", val.Required(), Lowercase()),
```

---

## Error Messages

Every built-in rule has a default message using `{field}` and `{param}` placeholders, matching Laravel's style:

```
"The {field} field is required"
"The {field} field must be at least {param} characters long"
"The {field} field must be a valid email address"
"The {field} field must be one of [{param}]"
```

Field names are auto-formatted from snake_case to readable: `created_at` → `created at`.

---

## Full Example

```go
type CreatePostRequest struct {
    Title       string
    Body        string
    Slug        string
    CategoryID  int
    Tags        []string
    PublishDate *time.Time
    Author      Author
    Images      []Image
}

type Author struct {
    Name  string
    Email string
}

type Image struct {
    URL     string
    Caption *string
}

func CreatePostRules(v val.Validator) CreatePostRequest {
    return CreatePostRequest{
        Title:      v.String("title", val.Required(), val.Min(5), val.Max(200)),
        Body:       v.String("body", val.Required(), val.Min(50)),
        Slug:       v.String("slug", val.Required(), val.Regex(`^[a-z0-9-]+$`), val.Max(200)),
        CategoryID: v.Int("category_id", val.Required(), val.IntMin(1)),
        Tags:       v.Strings("tags", val.ArrayMin(1), val.ArrayMax(10), val.Each(val.Min(1), val.Max(50))),
        PublishDate: v.Nullable().Date("publish_date", val.AfterNow()),
        Author: Author{
            Name:  v.String("author.name", val.Required(), val.Max(100)),
            Email: v.String("author.email", val.Required(), val.Email()),
        },
        Images: val.Array(v, "images", val.ArrayMax(10), func(v val.Validator) Image {
            return Image{
                URL:     v.String("url", val.Required(), val.URL()),
                Caption: v.Nullable().String("caption", val.Max(500)),
            }
        }),
    }
}
```

### Compared to Laravel

```php
$validated = $request->validate([
    'title'            => 'required|string|min:5|max:200',
    'body'             => 'required|string|min:50',
    'slug'             => 'required|string|regex:/^[a-z0-9-]+$/|max:200',
    'category_id'      => 'required|integer|min:1',
    'tags'             => 'array|min:1|max:10',
    'tags.*'           => 'string|min:1|max:50',
    'publish_date'     => 'nullable|date|after:now',
    'author.name'      => 'required|string|max:100',
    'author.email'     => 'required|email',
    'images'           => 'array|max:10',
    'images.*.url'     => 'required|url',
    'images.*.caption' => 'nullable|string|max:500',
]);
```

Nearly identical readability, but with compile-time type safety and zero reflection.

---

## Comparison to v10

| Aspect | v10 (struct tags) | This package |
|---|---|---|
| Type safety | Runtime (tags are strings) | Compile-time |
| Reflection | Yes | No |
| Error discovery | Runtime panics on typos | Compiler errors |
| Nested validation | Automatic via struct recursion | Dot notation |
| Custom validators | Global registration ceremony | Implement `Rule[T]` interface |
| Error messages | Generic, hard to customize | Laravel-style with `{field}` placeholders |
| Cross-field rules | Limited (`eqfield`, `gtfield`) | Deferred hooks on any field |
| Result type | Populated struct you pass in | Returned from rules function |
| Array elements | Struct tags on slice elements | `Each()` / `Array()` inline |
