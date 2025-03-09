package utils

import (
	"sort"
	"strings"

	"github.com/lemnispace/shop-api/internal/models"
)

// MatchesCollectionFilters checks if a collection matches the specified filters
func MatchesCollectionFilters(collection models.Collection, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "title":
			if strValue, ok := value.(string); ok && !strings.Contains(strings.ToLower(collection.Title), strings.ToLower(strValue)) {
				return false
			}
		case "description":
			if strValue, ok := value.(string); ok && !strings.Contains(strings.ToLower(collection.Description), strings.ToLower(strValue)) {
				return false
			}
			// Add more filter types as needed
		}
	}
	return true
}

// SortCollections sorts the collections based on the provided sort key and order
func SortCollections(collections []models.Collection, sortKey, sortOrder string) {
	sort.Slice(collections, func(i, j int) bool {
		// Default to ascending order
		isAscending := sortOrder != "desc"

		// Compare based on sort key
		switch sortKey {
		case "title":
			if isAscending {
				return collections[i].Title < collections[j].Title
			}
			return collections[i].Title > collections[j].Title
		case "createdAt":
			if isAscending {
				return collections[i].CreatedAt.Before(collections[j].CreatedAt)
			}
			return collections[i].CreatedAt.After(collections[j].CreatedAt)
		case "updatedAt":
			if isAscending {
				return collections[i].UpdatedAt.Before(collections[j].UpdatedAt)
			}
			return collections[i].UpdatedAt.After(collections[j].UpdatedAt)
		default:
			// Default sort by ID
			if isAscending {
				return collections[i].ID < collections[j].ID
			}
			return collections[i].ID > collections[j].ID
		}
	})
}
