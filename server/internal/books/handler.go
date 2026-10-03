package books

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type searchQuery struct {
	Topic         []string `form:"topic"`
	Genre         string   `form:"genre"`
	Provider      []string `form:"provider"`
	MinYear       *int     `form:"min_year"`
	MaxYear       *int     `form:"max_year"`
	Language      string   `form:"language"`
	MinPopularity *float64 `form:"min_popularity"`
	MinRating     *float64 `form:"min_rating"`
	Limit         int      `form:"limit"`
	Page          int      `form:"page"`
}

func (d *dependencies) HandleSearch(c *gin.Context) {
	var query searchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters", "details": err.Error()})
		return
	}
	if err := validateQuery(query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, provider := range query.Provider {
		if _, ok := d.byID[strings.ToLower(strings.TrimSpace(provider))]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown provider %q", provider)})
			return
		}
	}
	result, err := d.Search(c.Request.Context(), SearchRequest{
		Topics: query.Topic, Genre: query.Genre, Providers: query.Provider,
		MinYear: query.MinYear, MaxYear: query.MaxYear, Language: query.Language,
		MinPopularity: query.MinPopularity, MinRating: query.MinRating, Limit: query.Limit, Page: query.Page,
	})
	if err != nil {
		status := http.StatusServiceUnavailable
		if len(result.Providers) > 0 {
			allSkipped := true
			for _, provider := range result.Providers {
				if provider.Status != "skipped" {
					allSkipped = false
					break
				}
			}
			if allSkipped {
				status = http.StatusUnprocessableEntity
			}
		}
		c.JSON(status, gin.H{"error": err.Error(), "providers": result.Providers})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (d *dependencies) HandleProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"providers": d.Capabilities()})
}

func validateQuery(query searchQuery) error {
	for _, topic := range query.Topic {
		if strings.TrimSpace(topic) == "" {
			return errors.New("topic values must not be empty")
		}
		if len(strings.TrimSpace(topic)) > MaxParamLength {
			return fmt.Errorf("topic values must be at most %d characters", MaxParamLength)
		}
	}
	if len(query.Topic) > MaxSearchTopics {
		return fmt.Errorf("at most %d topic values are allowed", MaxSearchTopics)
	}
	if strings.TrimSpace(query.Genre) != "" && len(strings.TrimSpace(query.Genre)) > MaxParamLength {
		return fmt.Errorf("genre must be at most %d characters", MaxParamLength)
	}
	if strings.TrimSpace(query.Language) != "" && len(strings.TrimSpace(query.Language)) > MaxParamLength {
		return fmt.Errorf("language must be at most %d characters", MaxParamLength)
	}
	for _, provider := range query.Provider {
		if strings.TrimSpace(provider) == "" {
			return errors.New("provider values must not be empty")
		}
		if len(strings.TrimSpace(provider)) > MaxParamLength {
			return fmt.Errorf("provider values must be at most %d characters", MaxParamLength)
		}
	}
	if query.MinYear != nil && (*query.MinYear < -3000 || *query.MinYear > 3000) {
		return errors.New("min_year must be between -3000 and 3000")
	}
	if query.MaxYear != nil && (*query.MaxYear < -3000 || *query.MaxYear > 3000) {
		return errors.New("max_year must be between -3000 and 3000")
	}
	if query.MinYear != nil && query.MaxYear != nil && *query.MinYear > *query.MaxYear {
		return errors.New("min_year must be less than or equal to max_year")
	}
	if query.MinPopularity != nil && (*query.MinPopularity < 0 || math.IsNaN(*query.MinPopularity) || math.IsInf(*query.MinPopularity, 0)) {
		return errors.New("min_popularity must be non-negative")
	}
	if query.MinRating != nil && (*query.MinRating < 0 || *query.MinRating > 5 || math.IsNaN(*query.MinRating) || math.IsInf(*query.MinRating, 0)) {
		return errors.New("min_rating must be between 0 and 5")
	}
	if query.Limit < 0 || query.Limit > MaxResultLimit {
		return fmt.Errorf("limit must be between 0 and %d (0 uses the default of %d)", MaxResultLimit, DefaultResultLimit)
	}
	if query.Page < 0 || query.Page > MaxSearchPage {
		return fmt.Errorf("page must be between 1 and %d", MaxSearchPage)
	}
	return nil
}
