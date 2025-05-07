package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/thisisjab/gchat-go/internal/validator"
)

// envelope is used when returning JSON responses. Any JSON object must be enveloped.
type envelope map[string]any

// writeJSON writes a JSON response with the given status code and enveloped data.
func (s *APIServer) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.Marshal(data)

	if err != nil {
		return err
	}

	js = append(js, '\n')

	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

// readJSON is used when reading JSON requests. It reads the request body into the destination.
// If any expected error occurs, it is returned and can be handled as a bad request.
func (s *APIServer) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	const maxBytes = 1_048_576

	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)

	if err != nil {
		var (
			syntaxError           *json.SyntaxError
			unmarshalTypeError    *json.UnmarshalTypeError
			invalidUnmarshalError *json.InvalidUnmarshalError
			maxBytesError         *http.MaxBytesError
		)

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}

			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}

	}

	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must contain a single JSON object")
	}

	return nil
}

// readIntQuery reads an integer value from the query string.
func (s *APIServer) readIntQuery(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	value := qs.Get(key)

	if value == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		v.AddError(key, "must be an integer")
		return defaultValue
	}

	return i
}

// readUUIDParam reads a UUID value from the query string.
func (s *APIServer) readUUIDParam(key string, r *http.Request, v *validator.Validator) *uuid.UUID {
	params := httprouter.ParamsFromContext(r.Context())

	value, err := uuid.Parse(params.ByName(key))
	if err != nil {
		v.AddError(key, "invalid uuid")

		return nil
	}

	return &value
}
