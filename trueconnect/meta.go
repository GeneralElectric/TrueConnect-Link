package trueconnect

// FileMetadata stores a file reference and the associated metadata.
type FileMetadata struct {
	DataStoreRef string   `json:"data_store_ref"`
	Metadata     Metadata `json:"metadata"`
}

// Metadata is a map of the MetadataValues in FileMetadata.
type Metadata map[string]MetadataValue

// MetadataValue stores a value with optional immutable and notify flags.
type MetadataValue struct {
	// The metadata value.
	Value string `json:"value"`
	// True for immutable (write-once) values.
	Immutable bool `json:"immutable"`
	// True if this metadata is to be included in the notification message.
	Notify bool `json:"notify,omitempty"`
}

// Standard names for keys used in the APIs, such as query parameter keys.
const (
	// DataStoreRef is the unique reference for a stored file, as returned from
	// the backing store.
	// This reference will be used as a key for the stored metadata.
	DataStoreRef = "data_store_ref"

	// StartDate is used with EndDate to bound data searches to a time period
	// when querying metadata. Requires files to have DataStartDate and
	// DataEndDate metadata defined.
	//
	// This value must be ISO 8601 format (eg. 2016-07-04T15:03:43Z).
	// If a StartDate is given without an EndDate it will result in a Bad
	// Request (400).
	StartDate = "startdate"

	// EndDate is used with StartDate to bound data searches to a time period
	// when querying metadata. Requires files to have DataStartDate and
	// DataEndDate metadata defined.
	//
	// This value must be ISO 8601 format (eg. 2016-07-04T15:03:43Z).
	// If a EndDate is given without an StartDate it will result in a Bad
	// Request (400).
	EndDate = "enddate"
)

// Standard names for metadata keys to provide consistent naming of common
// and/or mandatory metadata.
// Metadata Keys are used for adding, updating or searching for metadata.
const (
	// TenantID is used to mark a file's access rights.
	//
	// This is a required metadata field that cannot be overwritten or deleted.
	TenantID = "tenant_id"

	// AssetRef is the asset that the file came from, such as the aircraft tail
	// number.
	//
	// This is an optional metadata field.
	AssetRef = "asset_ref"

	// DataType is the type of data, such as qar, eventlog, cmro.
	//
	// This is a required metadata field that cannot be overwritten or deleted.
	// This must only contain alphanumeric characters to conform to the
	// requirements of forming the routing key.
	DataType = "data_type"

	// FileFormat is the format of the file, such as csv, xml, json, sfd, a717.
	//
	// This is a required metadata field that cannot be overwritten or deleted.
	// This must only contain alphanumeric characters to conform to the
	// requirements of forming the routing key.
	FileFormat = "file_format"

	// CustomRoute is used as a component to create the MQ Exchange routing
	// key.
	//
	// This is an optional metadata field, but if provided it cannot be
	// overwritten or deleted.
	// This must only contain alphanumeric characters to conform to the
	// requirements of forming the routing key.
	CustomRoute = "custom_route"

	// FileArrivalDate is the date at which the file arrived.
	//
	// This value must be ISO 8601 format (eg. 2016-07-04T15:03:43Z).
	// If this is not given it will be set to current time at time of arrival.
	// Files are ordered by this value.
	FileArrivalDate = "file_arrival_date"

	// DataStartDate is the earliest date associated with the data within the
	// file.
	//
	// This value must be ISO 8601 format (eg. 2016-07-04T15:03:43Z).
	// Must be supplied to perform time-bounded searches using start and end
	// date query parameters.
	DataStartDate = "data_start_date"

	// DataEndDate is the latest date associated with the data within the file.
	//
	// This value must be ISO 8601 format (eg. 2016-07-04T15:03:43Z).
	// Must be supplied to perform time-bounded searches using start and end
	// date query parameters.
	DataEndDate = "data_end_date"

	// NotifiedMetadata is the metadata that has been passed into the
	// notification message.
	//
	// This is an optional field in the notification message, with a value of
	// a map of key-value pairs.
	NotifiedMetadata = "notified_metadata"

	// ConfigRef identifies the configuration used to read/interpret/decode the
	// file.
	ConfigRef = "config_ref"

	// OriginalFileName is the original name of the stored file.
	OriginalFileName = "original_file_name"
)
