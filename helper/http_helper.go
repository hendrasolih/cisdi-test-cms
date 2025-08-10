package helper

import (
	"math"
	"net/http"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
)

const (
	textError             = `error`
	textOk                = `ok`
	codeSuccess           = 200
	codeBadRequestError   = 400
	codeUnauthorizedError = 401
	codeDatabaseError     = 402
	codeValidationError   = 403
	codeNotFound          = 404
)

// ResponseHelper ...
type ResponseHelper struct {
	C        *gin.Context
	Status   string
	Message  string
	Data     interface{}
	Code     int // not the http code
	CodeType string
}

// HTTPHelper ...
type HTTPHelper struct {
	Validate   *validator.Validate
	Translator ut.Translator
}

func (u *HTTPHelper) getTypeData(i interface{}) string {
	v := reflect.ValueOf(i)
	v = reflect.Indirect(v)

	return v.Type().String()
}

// GetStatusCode ...
func (u *HTTPHelper) GetStatusCode(err error) int {
	statusCode := http.StatusOK
	if err != nil {
		switch u.getTypeData(err) {
		case "models.ErrorUnauthorized":
			statusCode = http.StatusUnauthorized
		case "models.ErrorNotFound":
			statusCode = http.StatusNotFound
		case "models.ErrorConflict":
			statusCode = http.StatusConflict
		case "models.ErrorInternalServer":
			statusCode = http.StatusInternalServerError
		default:
			statusCode = http.StatusInternalServerError
		}
	}

	return statusCode
}

// SetResponse ...
// Set response data.
func (u *HTTPHelper) SetResponse(c *gin.Context, status string, message string, data interface{}, code int, codeType string) ResponseHelper {
	return ResponseHelper{c, status, message, data, code, codeType}
}

// SendError ...
// Send error response to consumers.
func (u *HTTPHelper) SendError(c *gin.Context, message string, data interface{}, code int, codeType string) error {
	res := u.SetResponse(c, textError, message, data, code, codeType)

	return u.SendResponse(res)
}

func (u *HTTPHelper) SendErrorV2(c *gin.Context, message string, data interface{}, code int, codeType string) error {
	res := u.SetResponse(c, textError, message, data, code, codeType)

	return u.SendResponseV2(res)
}

// SendBadRequest ...
// Send bad request response to consumers.
func (u *HTTPHelper) SendBadRequest(c *gin.Context, message string, data interface{}) error {
	res := u.SetResponse(c, textError, message, data, codeBadRequestError, `badRequest`)

	return u.SendResponse(res)
}

// SendValidationError ...
// Send validation error response to consumers.
func (u *HTTPHelper) SendValidationError(c *gin.Context, validationErrors validator.ValidationErrors) error {
	errorResponse := map[string][]string{}
	errorTranslation := validationErrors.Translate(u.Translator)
	for _, err := range validationErrors {
		errKey := Underscore(err.StructField())
		errorResponse[errKey] = append(errorResponse[errKey], errorTranslation[err.Namespace()])
	}

	c.JSON(400, map[string]interface{}{
		"code":         codeValidationError,
		"code_type":    "[Shipment] validationError",
		"code_message": errorResponse,
		"data":         u.EmptyJsonMap(),
	})
	return nil
}

// SendDatabaseError ...
// Send database error response to consumers.
func (u *HTTPHelper) SendDatabaseError(c *gin.Context, message string, data interface{}) error {
	return u.SendError(c, message, data, codeDatabaseError, `databaseError`)
}

// SendUnauthorizedError ...
// Send unauthorized response to consumers.
func (u *HTTPHelper) SendUnauthorizedError(c *gin.Context, message string, data interface{}) error {
	return u.SendError(c, message, data, codeUnauthorizedError, `unAuthorized`)
}

// SendNotFoundError ...
// Send not found response to consumers.
func (u *HTTPHelper) SendNotFoundError(c *gin.Context, message string, data interface{}) error {
	return u.SendError(c, message, data, codeNotFound, `notFound`)
}

func (u *HTTPHelper) SendNotFoundErrorV2(c *gin.Context, message string, data interface{}) error {
	return u.SendErrorV2(c, message, data, codeNotFound, `notFound`)
}

// SendSuccess ...
// Send success response to consumers.
func (u *HTTPHelper) SendSuccess(c *gin.Context, message string, data interface{}) error {
	res := u.SetResponse(c, textOk, message, data, codeSuccess, `success`)

	return u.SendResponse(res)
}

// SendResponse ...
// Send response
func (u *HTTPHelper) SendResponse(res ResponseHelper) error {
	if len(res.Message) == 0 {
		res.Message = `success`
	}

	var resCode int
	if res.Code != 200 {
		resCode = http.StatusBadRequest
	} else {
		resCode = http.StatusOK
	}

	res.C.JSON(resCode, map[string]interface{}{
		"code":         res.Code,
		"code_type":    res.CodeType,
		"code_message": res.Message,
		"data":         res.Data,
	})
	return nil
}

func (u *HTTPHelper) SendResponseV2(res ResponseHelper) error {
	var resCode int
	if res.Code == 404 {
		resCode = http.StatusNotFound
	} else if res.Code == 400 {
		resCode = http.StatusBadRequest
	} else {
		resCode = http.StatusOK
	}

	res.C.JSON(resCode, map[string]interface{}{
		"code":         res.Code,
		"code_type":    res.CodeType,
		"code_message": res.Message,
		"data":         res.Data,
	})
	return nil
}

func (u *HTTPHelper) EmptyJsonMap() map[string]interface{} {
	return make(map[string]interface{})
}

// get pagination URL
func (u *HTTPHelper) GetPagingUrl(c *gin.Context, page, limit int) string {
	r := c.Request
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	currentURL := scheme + "://" + r.Host + r.URL.Path + "?page=" + strconv.Itoa(page) + "&limit=" + strconv.Itoa(limit)
	return currentURL
}

// Set paginantion response
func (u *HTTPHelper) GeneratePaging(c *gin.Context, prev, next, limit, page, totalRecord int) map[string]interface{} {

	prevURL, nextURL, firstURL, lastURL := "", "", "", ""

	totalPages := int(math.Ceil(float64(totalRecord) / float64(limit)))

	if page > 1 {
		prev = page - 1
		if page < totalPages {
			next = page + 1
		} else {
			next = totalPages
		}
	}

	if totalPages >= page && page > 1 {
		prevURL = u.GetPagingUrl(c, prev, limit)
	}

	if totalPages > page {
		nextURL = u.GetPagingUrl(c, next, limit)
	}

	if totalPages >= page && page > 1 {
		firstURL = u.GetPagingUrl(c, 1, limit)
	}

	if totalPages >= page && totalPages != page {
		lastURL = u.GetPagingUrl(c, totalPages, limit)
	}

	links := map[string]interface{}{
		"previous": prevURL,
		"next":     nextURL,
		"first":    firstURL,
		"last":     lastURL,
	}

	pagination := map[string]interface{}{
		"total_records": totalRecord,
		"per_page":      limit,
		"current_page":  page,
		"total_pages":   totalPages,
		"links":         links,
	}

	return pagination
}
