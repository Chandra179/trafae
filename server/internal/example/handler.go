package example

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type createExampleRequest struct {
	Name string `json:"name" binding:"required"`
}

// @Summary      Create example
// @Description  Creates and persists an example record
// @Tags         example
// @Accept       json
// @Produce      json
// @Param        request  body      createExampleRequest  true  "example payload"
// @Success      201  {object}  Example
// @Router       /example [post]
func (d *dependencies) HandleExample(c *gin.Context) {
	var req createExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(fmt.Errorf("bind create example request: %w", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ex, err := d.CreateExample(c.Request.Context(), req.Name)
	if err != nil {
		_ = c.Error(err)

		if errors.Is(err, ErrReservedName) {
			c.JSON(http.StatusConflict, gin.H{"error": "name is reserved"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, ex)
}
