package query

// Sort defines a sort request made by JSON:API clients.
type Sort struct {
	Property   string // The name of the property to sort by.
	Descending bool   // If true, sort in descending order.
}

// Page defines a pagination request made by JSON:API clients.
type Page struct {
	PageNumber int    // The requested page number.
	Cursor     string // The start page cursor.
	Limit      int    // The maximum number of resources to return.
}

// Fieldset defines a sparse fieldset request made by JSON:API clients.
type Fieldset struct {
	Property string // The name of the resource property to include in the return document.
}
