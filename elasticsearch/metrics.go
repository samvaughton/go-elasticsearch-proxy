package elasticsearch

import (
	"github.com/tidwall/gjson"
)

const MetricDateRange = "dateRange"
const MetricLocation = "location"
const MetricAgency = "agency"
const MetricFeatures = "features"

// Range types
const MetricGuests = "guests"
const MetricPets = "pets"
const MetricBedrooms = "bedrooms"
const MetricBathrooms = "bathrooms"
const MetricNightlyLowPrice = "nightlyLow"
const MetricNightlyHighPrice = "nightlyHigh"

const MetricUnknown = "unknown"

type MetricDateRangeData struct {
	ArrivalDate   string `json:"arrivalDate"`
	DepartureDate string `json:"departureDate"`
}

type MetricFeaturesData struct {
	SearchType string   `json:"searchType"`
	Items      []string `json:"items"`
}

type MetricLocationData struct {
	Distance  string  `json:"distance"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type MetricRangeData struct {
	Minimum float64 `json:"minimum"`
	Maximum float64 `json:"maximum"`
}

type MetricAgencyData struct {
	CompanyName string `json:"companyName"`
}

var MetricExtractionHandlersMap = map[string]func(metricType string, query gjson.Result) interface{}{
	MetricLocation: func(metricType string, query gjson.Result) interface{} {
		return MetricLocationData{
			Distance:  query.Get("distance").String(),
			Latitude:  query.Get("location.1").Float(),
			Longitude: query.Get("location.0").Float(),
		}
	},
	MetricDateRange: func(metricType string, query gjson.Result) interface{} {
		return MetricDateRangeData{
			ArrivalDate:   query.Get("script.params.arrivalDate").String(),
			DepartureDate: query.Get("script.params.departureDate").String(),
		}
	},
	MetricAgency: func(metricType string, query gjson.Result) interface{} {
		return MetricAgencyData{
			CompanyName: query.Get("query").String(),
		}
	},
	MetricFeatures: func(metricType string, query gjson.Result) interface{} {
		list := query.Get("query.bool.should")

		var items []string

		if list.IsArray() && len(list.Array()) > 0 {
			// Loop through the terms and extract them out
			for _, termContainer := range list.Array() {
				termObj := termContainer.Get("term")

				// Since the es query actually has a period delimited field in it we need to
				// simply match the first and only field in this object otherwise the gjson library
				// will never find it
				items = append(items, termObj.Get("*").String())
			}
		}

		return MetricFeaturesData{
			SearchType: "collection",
			Items:      items,
		}
	},
	MetricGuests:           ParseStandardRangeQuery,
	MetricPets:             ParseStandardRangeQuery,
	MetricBedrooms:         ParseStandardRangeQuery,
	MetricBathrooms:        ParseStandardRangeQuery,
	MetricNightlyLowPrice:  ParseStandardRangeQuery,
	MetricNightlyHighPrice: ParseStandardRangeQuery,
}

func ParseStandardRangeQuery(metricType string, query gjson.Result) interface{} {
	return MetricRangeData{
		Minimum: query.Get("gte").Float(),
		Maximum: query.Get("lte").Float(),
	}
}

// This will attempt to extract meaning data from the query into easier to understand format
func ExtractQueryMetrics(query gjson.Result) map[string]interface{} {
	// Things we want to extract
	// Location bool.must[].bool.must[].bool.should[].geo_distance { distance, location }
	// Dates bool.must[].bool.must[].bool.must[].bool.must[].script.script { id "lycan_availability_filter_advanced", params.arrivalDate, params.departureDate }

	// We need to recursively drill down into this query and pull out all useful metrics
	metrics := make(map[string]interface{})

	ExtractQueryMetricsRecursive(query, metrics)

	return metrics
}

type RecursiveQueryParserResult struct {
	FoundMetric bool
	Data        gjson.Result
}

// This will drill down all paths and find the actual queries
func ExtractQueryMetricsRecursive(query gjson.Result, metrics map[string]interface{}) {
	if !query.Exists() {
		return
	}

	if query.IsArray() || query.IsObject() {
		query.ForEach(func(key, value gjson.Result) bool {
			if CanDescend(key) {
				ExtractQueryMetricsRecursive(value, metrics)
			}

			metricType := DetermineMetric(key.Str, query)

			if metricType != MetricUnknown {
				metricData := ExtractMetric(metricType, value)

				if metricType == MetricFeatures {

					if metrics[MetricFeatures] == nil {
						metrics[MetricFeatures] = make([]interface{}, 0)
					}

					metrics[MetricFeatures] = append(metrics[MetricFeatures].([]interface{}), metricData)

				} else {
					metrics[metricType] = metricData
				}

			}

			return true
		})
	}
}

func CanDescend(key gjson.Result) bool {
	return key.Str == "bool" ||
		key.Str == "filter" ||
		key.Str == "must" ||
		key.Str == "should" ||
		key.Str == "range" ||
		key.Str == "match" ||
		key.Str == ""
}

func ExtractMetric(metricType string, query gjson.Result) interface{} {
	// Since not all metrics can be located by the key values of the starting object we need refer to the handler map
	if metricType != MetricUnknown {
		if fn, exists := MetricExtractionHandlersMap[metricType]; exists {
			return fn(metricType, query)
		}
	}

	return nil
}

func DetermineMetric(key string, query gjson.Result) string {
	// Handle key based metrics first
	switch key {
	case "geo_distance":
		return MetricLocation
	case "listing.sleeps":
		return MetricGuests
	case "listing.maxPets":
		return MetricPets
	case "listing.bedrooms":
		return MetricBedrooms
	case "listing.bathrooms":
		return MetricBathrooms
	case "pricing.visual.nightlyLow":
		return MetricNightlyLowPrice
	case "pricing.visual.nightlyHigh":
		return MetricNightlyHighPrice
	case "$marketing.agent.brandCompanyName":
		return MetricAgency
	}

	// Script has a nested script so access that and see if it exists
	if scriptId := query.Get("script.script.id"); scriptId.Exists() {
		switch scriptId.String() {
		case "lycan_availability_filter_advanced":
			return MetricDateRange
		}
	}

	if featuresObj := query.Get("nested"); featuresObj.Exists() && featuresObj.Get("path").String() == "features" {
		list := featuresObj.Get("query.bool.should")

		if list.IsArray() && len(list.Array()) > 0 {
			return MetricFeatures
		}
	}

	return MetricUnknown
}
