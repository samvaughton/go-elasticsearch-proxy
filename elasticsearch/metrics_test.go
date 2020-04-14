package elasticsearch

import (
	"fmt"
	"github.com/tidwall/gjson"
	"strconv"
	"testing"
)

func TestExtractQueryMetrics(t *testing.T) {

	/*
	 * If we want the test to fail, then Fail() on a valid query
	 */

	jsonOne := gjson.Parse(`{"bool":{"must":[{"bool":{"must":[{"bool":{"should":[{"geo_bounding_box":{"location":{"top_right":{"lat":50.93127,"lon":-4.1649445},"bottom_left":{"lat":49.9554143,"lon":-5.747107}},"boost":4}},{"geo_distance":{"distance":"49.59miles","location":[-4.95602575,50.44334215]}}]}},{"bool":{"must":[{"bool":{"must":[{"script":{"script":{"id":"lycan_availability_filter_advanced","params":{"arrivalDate":"2020-04-10","departureDate":"2020-04-17"}}}}]}}]}},{"bool":{"must":[{"range":{"listing.sleeps":{"gte":3,"boost":2}}},{"range":{"listing.maxPets":{"gte":1,"boost":2}}}]}},{"match":{"$marketing.agent.brandCompanyName":{"query":"FBM Holidays","operator":"and"}}},{"range":{"pricing.visual.nightlyLow":{"gte":50,"lte":210,"boost":2}}},{"range":{"listing.bedrooms":{"gte":2,"lte":3}}},{"range":{"listing.bathrooms":{"gte":2}}},{"bool":{"must":[{"nested":{"path":"features","query":{"bool":{"should":[{"term":{"features.type.keyword":"GENERAL_PET_FRIENDLY"}},{"term":{"features.type.keyword":"SUITABILITY_PETS_ALLOWED"}}]}}}},{"nested":{"path":"features","query":{"bool":{"should":[{"term":{"features.type.keyword":"GENERAL_PARKING"}},{"term":{"features.type.keyword":"GENERAL_DEDICATED_PRIVATE_PARKING"}},{"term":{"features.type.keyword":"GENERAL_SHARED_PRIVATE_PARKING"}},{"term":{"features.type.keyword":"GENERAL_ONSTREET_FREE_PARKING"}},{"term":{"features.type.keyword":"ACCESSIBILITY_DISABLED_PARKING_SPOT"}}]}}}}]}}]}}]}}`)

	t.Run("test recursive function works", func(t *testing.T) {
		metrics := ExtractQueryMetrics(jsonOne)

		if len(metrics) != 9 {
			t.Fail()
		}
	})

	jsonTwo := gjson.Parse(`{"bool":{"must":[{"bool":{"must":[{"range":{"pricing.visual.nightlyLow":{"gte":0,"lte":9999,"boost":2}}},{"bool":{"must":[{"nested":{"path":"features","query":{"bool":{"should":[{"term":{"features.type.keyword":"GENERAL_PET_FRIENDLY"}},{"term":{"features.type.keyword":"SUITABILITY_PETS_ALLOWED"}}]}}}},{"nested":{"path":"features","query":{"bool":{"should":[{"term":{"features.type.keyword":"OUTDOOR_GARDEN"}},{"term":{"features.type.keyword":"OUTDOOR_BALCONY"}},{"term":{"features.type.keyword":"OUTDOOR_BARBECUE"}},{"term":{"features.type.keyword":"OUTDOOR_DECK_PATIO"}},{"term":{"features.type.keyword":"OUTDOOR_GARDEN_FURNITURE"}},{"term":{"features.type.keyword":"OUTDOOR_HAMMOCK"}},{"term":{"features.type.keyword":"OUTDOOR_OUTDOOR_DINING"}},{"term":{"features.type.keyword":"OUTDOOR_SUMMER_HOUSE"}},{"term":{"features.type.keyword":"OUTDOOR_TABLE_AND_CHAIRS"}},{"term":{"features.type.keyword":"OUTDOOR_TERRACE"}}]}}}},{"nested":{"path":"features","query":{"bool":{"should":[{"term":{"features.type.keyword":"GENERAL_PARKING"}},{"term":{"features.type.keyword":"GENERAL_DEDICATED_PRIVATE_PARKING"}},{"term":{"features.type.keyword":"GENERAL_SHARED_PRIVATE_PARKING"}},{"term":{"features.type.keyword":"GENERAL_ONSTREET_FREE_PARKING"}},{"term":{"features.type.keyword":"ACCESSIBILITY_DISABLED_PARKING_SPOT"}}]}}}}]}}]}}]}}`)

	t.Run("test recursive function works", func(t *testing.T) {
		metrics := ExtractQueryMetrics(jsonTwo)

		if len(metrics) != 2 {
			t.Log("Metrics Array Count: " + strconv.Itoa(len(metrics)))
			t.Fail()
		}

		featuresArray, ok := metrics[MetricFeatures].([]interface{})

		if ok {
			if len(featuresArray) != 3 {
				t.Log("Features Array Count: " + strconv.Itoa(len(featuresArray)))
				fmt.Println(featuresArray)
				t.Fail()
			}
		} else {
			t.Error("Incorrect interface for features array")
			t.Fail()
		}

	})

}
