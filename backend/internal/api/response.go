package api

import "github.com/gin-gonic/gin"

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type responseBody struct {
	Data  any        `json:"data"`
	Error *errorBody `json:"error"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, responseBody{
		Data:  data,
		Error: nil,
	})
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, responseBody{
		Data: nil,
		Error: &errorBody{
			Code:    code,
			Message: message,
		},
	})
}
