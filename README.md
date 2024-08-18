# jsonapi

[![Test](https://github.com/nisimpson/gibbon/actions/workflows/jsonapi-test.yml/badge.svg)](https://github.com/nisimpson/gibbon/actions/workflows/jsonapi-test.yml)
[![GoDoc](https://godoc.org/github.com/gonobo/jsonapi/v2?status.svg)](http://godoc.org/github.com/gonobo/jsonapi/v2)
[![Release](https://img.shields.io/github/release/gonobo/jsonapi.svg)](https://github.com/gonobo/releases)

**Yet Another JSON API library for Go.**

Package [`jsonapi`](http://godoc.org/github.com/gonobo/jsonapi/v2) provides structures and functions to implement [JSON API](http://jsonapi.org) compatible APIs. The library can be used with any framework and is built on top of the standard Go http library.

## Installation

Get the package using the go tool:

```bash
$ go get -u github.com/gonobo/jsonapi/v2
```

## Structures

This library uses [StructField](http://golang.org/pkg/reflect/#StructField)
tags to annotate the structs fields that you already have and use in
your app and then reads and writes [JSON API](http://jsonapi.org)
output based on the tag content.

```go
type Customer struct {
  ID   string `jsonapi:"primary,customers"`
  Name string `jsonapi:"attr,name"`
}

type Order struct {
  ID       string     `jsonapi:"primary,orders"`
  Customer *Customer  `jsonapi:"relation,customer"`
  Products []*Product `jsonapi:"relation,products,omitempty"`
}

type Product struct {
  ID string   `jsonapi:"primary,products"`
  Name string `jsonapi:"attr,name"`
}

// This object...
order := Order{
  ID:       "1",
  Customer: &Customer{ID: "2", Name: "John"},
  Products: []*Product{
    {ID: "42", Name: "Shoes"},
    {ID: "24", Name: "Socks"},
  }
}

// ...is transformed into this resource when marshaled.
resource := jsonapi.Resource{
  ID:   "1",
  Type: "orders",
  Relationships: jsonapi.Relationships{
    "customer": &jsonapi.Relationship{
      Data: jsonapi.One{
        Value: &jsonapi.Resource{
          ID:         "2",
          Type:       "customers",
          Attributes: map[string]any{"name": "John"},
        }}
    },
    "products": &jsonapi.Relationship{
      Data: jsonapi.Many{
        Values: []*jsonapi.Resource{
          {
            ID:         "42",
            Type:       "products",
            Attributes: map[string]any{"name": "Shoes"}
          },
          {
            ID:         "24",
            Type:       "products",
            Attributes: map[string]any{"name": "Socks"}
          },
        },
      },
    },
  }
}
```

### Permitted Tag Values

Struct tag values are equivalent to those found in the
[Google JSON API library](https://github.com/google/jsonapi):

#### `primary`

```go
`jsonapi:"primary,<type field output>"`
```

This indicates this is the primary key field for this struct type.
Tag value arguments are comma separated. The first argument must be,
`primary`, and the second must be the name that should appear in the
`type`\* field for all data objects that represent this type of model.

\* According the [JSON API](http://jsonapi.org) spec, the plural record
types are shown in the examples, but not required.

#### `attr`

```go
`jsonapi:"attr,<key name in attributes hash>,<optional: omitempty>"`
```

These fields' values will end up in the `attributes`hash for a record.
The first argument must be, `attr`, and the second should be the name
for the key to display in the `attributes` hash for that record. The optional
third argument is `omitempty` - if it is present the field will not be present
in the `"attributes"` if the field's value is equivalent to the field types
empty value (ie if the `count` field is of type `int`, `omitempty` will omit the
field when `count` has a value of `0`). Lastly, the spec indicates that
`attributes` key names should be dasherized for multiple word field names.

#### `relation`

```go
`jsonapi:"relation,<key name in relationships hash>,<optional: omitempty>"`
```

Relations are struct fields that represent a one-to-one or one-to-many
relationship with other structs. JSON API will traverse the graph of
relationships and marshal or unmarshal records. The first argument must
be, `relation`, and the second should be the name of the relationship,
used as the key in the `relationships` hash for the record. The optional
third argument is `omitempty` - if present will prevent non existent to-one and
to-many from being serialized.

## Marshaling and Unmarshaling

> All `Marshal` and `Unmarshal` methods expect pointers to struct
> instance or slices of the same type. Using values during marshaling and
> unmarshal is undefined behavior.

```go
import "github.com/gonobo/jsonapi/v2"

func createOrder(w *http.ResponseWriter, r *http.Request) {
  in, err := jsonapi.Decode(r.Body)
  order := Order{}
  err = jsonapi.Unmarshal(in, &order)

  newOrder, err := db.CreateNewOrder(order)
  out, err := jsonapi.Marshal(newOrder)
  w.WriteHeader(http.StatusCreated)
  err = jsonapi.Write(w, out)
}
```

## JSON:API Server Handlers

This library also provides custom handlers to remove some of the boilerplate needed to adhere to the
JSON:API specification.

```go
import (
  "github.com/gonobo/jsonapi/v2"
  "github.com/gonobo/jsonapi/v2/mux"
)

func getOrder(r *http.Request) jsonapi.Response {
  // ctx contains the JSON:API information extracted from the http request:
  // - request type
  // - request id
  // - relationship name
  // - unmarshaled payload
  ctx, ok := jsonapi.GetContext(r.Context())

  order, err := db.GetOrder(ctx.ResourceID)
  out, err := jsonapi.Marshal(order)
  res := jsonapi.NewResponse(http.StatusCreated)
  res.Body = out
  return res
}

func main() {
  m := mux.New(
    mux.WithRoute("orders", mux.Route{
      // GET requests to /orders/:id will be routed to this handler.
      Get: jsonapi.HandlerFunc(getOrder),
      // GET requests to /orders will be routed to this handler.
      List: jsonapi.HandlerFunc(listOrders),
      // all other requests will return 404.
    })
    mux.WithRoute("customers", ...),
    mux.WithRoute("products", ...),
  )

  // Handler(h, ...opts) provides the request context using a default
  // request context resolver. This behavior can be modified via options.
  log.Fatal(http.ListenAndServe(":3333", jsonapi.Handler(m)))
}

```

`Route` also implements `jsonapi.Handler`, so you skip using `Mux` if you don't need it.
This could be useful in serverless scenarios whose compute only serves a specific
resource or operation:

```go

jsonapi.Handler(mux.Route{
  Get: jsonapi.HandlerFunc(getOrder),
})

// OR

jsonapi.Handler(jsonapi.HandlerFunc(getOrder),
  func(h *jsonapi.H) {
    h.RequestContextResolver = MyCustomContextResolver()
  },
)
```

## Examples

TBD.

## License

The MIT License (MIT)

Copyright (c) 2024 Nathan Simpson
