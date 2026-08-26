package decodo

// ResultEntry represents a single result from a scrape operation.
type ResultEntry struct {
	Content                interface{}              `json:"content"`
	StatusCode             int                      `json:"status_code"`
	URL                    string                   `json:"url,omitempty"`
	TaskID                 string                   `json:"task_id"`
	Headers                map[string]string        `json:"headers,omitempty"`
	Cookies                []map[string]interface{} `json:"cookies,omitempty"`
	CreatedAt              string                   `json:"created_at"`
	UpdatedAt              string                   `json:"updated_at"`
	Help                   string                   `json:"help,omitempty"`
	BrowserActionsWarnings []map[string]string      `json:"browser_actions_warnings,omitempty"`
	BrowserActionsError    []map[string]string      `json:"browser_actions_error,omitempty"`
	DeliveryZip            string                   `json:"delivery_zip,omitempty"`
}

// SyncResponse is returned from synchronous scrape requests.
type SyncResponse struct {
	Results []ResultEntry `json:"results"`
}

// AsyncTaskResponse is returned when creating an async task.
type AsyncTaskResponse struct {
	ID                     string              `json:"id"`
	Target                 string              `json:"target,omitempty"`
	URL                    string              `json:"url,omitempty"`
	Query                  string              `json:"query,omitempty"`
	Status                 string              `json:"status"`
	CreatedAt              string              `json:"created_at,omitempty"`
	UpdatedAt              string              `json:"updated_at,omitempty"`
	PageFrom               int                 `json:"page_from,omitempty"`
	Limit                  int                 `json:"limit,omitempty"`
	Geo                    *string             `json:"geo,omitempty"`
	DeviceType             string              `json:"device_type,omitempty"`
	Headless               *string             `json:"headless,omitempty"`
	Parse                  bool                `json:"parse,omitempty"`
	Locale                 *string             `json:"locale,omitempty"`
	Domain                 string              `json:"domain,omitempty"`
	OutputSchema           *string             `json:"output_schema,omitempty"`
	ContentEncoding        string              `json:"content_encoding,omitempty"`
	PageCount              int                 `json:"page_count,omitempty"`
	Adults                 int                 `json:"adults,omitempty"`
	Children               int                 `json:"children,omitempty"`
	CallbackURL            string              `json:"callback_url,omitempty"`
	BrowserActionsError    []map[string]string `json:"browser_actions_error,omitempty"`
	BrowserActionsWarnings []map[string]string `json:"browser_actions_warnings,omitempty"`
}

// BatchResponse is returned from batch scrape requests.
type BatchResponse struct {
	ID      int64               `json:"id,omitempty"`
	Queries []AsyncTaskResponse `json:"queries,omitempty"`
	Errors  []map[string]string `json:"errors,omitempty"`
}

// TaskStatus represents the status of an async task.
type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusFaulted TaskStatus = "faulted"
	TaskStatusDone    TaskStatus = "done"
)

// TaskMetadata contains the status of an async task.
type TaskMetadata struct {
	Status TaskStatus `json:"status"`
}

// TaskResultsResponse contains results from a completed async task.
type TaskResultsResponse struct {
	Results []ResultEntry `json:"results"`
}
