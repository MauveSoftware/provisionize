package api

const (
	// StatusCodeOK is returned if the request was processed successfully
	StatusCodeOK = 200

	// StatusCodeRequestError is returned when the request did not pass the sanity checks prior processing
	StatusCodeRequestError = 400

	// StatusCodeProcessingError is returned when an error occured while processing
	StatusCodeProcessingError = 500
)
