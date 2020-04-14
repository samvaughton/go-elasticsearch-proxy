package elasticsearch

import (
	"github.com/tidwall/gjson"
	"testing"
)

func TestParseJsonBodyLines(t *testing.T) {
	testJson := `{"preference":"price"}
		{"track_total_hits":20000,"query":{"function_score":{"functions":[{"random_score":{"seed":[2020,15],"field":"_seq_no"}}],"query":{"bool":{"must":[{"bool":{"must":[{"range":{"pricing.visual.nightlyLow":{"gte":0,"lte":9999,"boost":2}}}]}}]}},"score_mode":"multiply"}}}
		{"preference":"price"}
		{"track_total_hits":20000,"query":{"function_score":{"functions":[{"random_score":{"seed":[2020,15],"field":"_seq_no"}}],"query":{"bool":{"must":[{"bool":{"must":[{"range":{"pricing.visual.nightlyLow":{"gte":0,"lte":9999,"boost":2}}}]}}]}},"score_mode":"multiply"}}}`

	lines := ParseJsonBodyLines(testJson)

	if len(lines) != 4 {
		t.Fail()
	}
}

func TestIsValidElasticsearchQuery(t *testing.T) {

	/*
	 * If we want the test to fail, then Fail() on a valid query
	 */

	t.Run("preference:price fails", func(t *testing.T) {
		if IsValidQuery(gjson.Parse(`{"preference":"price"}`)) {
			t.Fail()
		}
	})

	t.Run("preference:page fails", func(t *testing.T) {
		if IsValidQuery(gjson.Parse(`{"preference":"page"}`)) {
			t.Fail()
		}
	})

	t.Run("query that is a bool fails", func(t *testing.T) {
		if IsValidQuery(gjson.Parse(`{"query":true}`)) {
			t.Fail()
		}
	})

	t.Run("query exists passes", func(t *testing.T) {
		if !IsValidQuery(gjson.Parse(`{"query":{}}`)) {
			t.Fail()
		}
	})
}

func TestDeDuplicateJsonLines(t *testing.T) {
	t.Run("duplicate lines are removed", func(t *testing.T) {
		lines := []gjson.Result{
			gjson.Parse(`{"foo": "bar"}`),
			gjson.Parse(`{"foo": "bar"}`),
		}

		if len(DeDuplicateJsonLines(lines)) != 1 {
			t.Fail()
		}
	})

	t.Run("duplicate lines are removed with other unique lines", func(t *testing.T) {
		lines := []gjson.Result{
			gjson.Parse(`{"line": "1"}`),
			gjson.Parse(`{"line": "1"}`),
			gjson.Parse(`{"line": "2"}`),
			gjson.Parse(`{"line": "3"}`),
			gjson.Parse(`{"line": "4"}`),
			gjson.Parse(`{"line": "4"}`),
			gjson.Parse(`{"line": "4"}`),
		}

		if len(DeDuplicateJsonLines(lines)) != 4 {
			t.Fail()
		}
	})

	t.Run("duplicate json lines are removed", func(t *testing.T) {
		lines := []gjson.Result{
			gjson.Parse(`{"bool":{"must":[{"bool":{"must":[{"bool":{"should":[{"geo_bounding_box":{"location":{"top_right":{"lat":50.93127,"lon":-4.1649445},"bottom_left":{"lat":49.9554143,"lon":-5.747107}},"boost":4}},{"geo_distance":{"distance":"49.59miles","location":[-4.95602575,50.44334215]}}]}},{"range":{"pricing.visual.nightlyLow":{"gte":0,"lte":9999,"boost":2}}},{"range":{"listing.bedrooms":{"gte":4,"lte":5}}},{"bool":{"must":[{"nested":{"path":"features","query":{"bool":{"should":[{"term":{"features.type.keyword":"GENERAL_PET_FRIENDLY"}},{"term":{"features.type.keyword":"SUITABILITY_PETS_ALLOWED"}}]}}}},{"nested":{"path":"features","query":{"bool":{"must":[{"term":{"features.type.keyword":"SPA_POOL_JACUZZI_HOT_TUB"}}]}}}}]}}]}}]}}`),
			gjson.Parse(`{"bool":{"must":[{"bool":{"must":[{"bool":{"should":[{"geo_bounding_box":{"location":{"top_right":{"lat":50.93127,"lon":-4.1649445},"bottom_left":{"lat":49.9554143,"lon":-5.747107}},"boost":4}},{"geo_distance":{"distance":"49.59miles","location":[-4.95602575,50.44334215]}}]}},{"range":{"pricing.visual.nightlyLow":{"gte":0,"lte":9999,"boost":2}}},{"range":{"listing.bedrooms":{"gte":4,"lte":5}}},{"bool":{"must":[{"nested":{"path":"features","query":{"bool":{"should":[{"term":{"features.type.keyword":"GENERAL_PET_FRIENDLY"}},{"term":{"features.type.keyword":"SUITABILITY_PETS_ALLOWED"}}]}}}},{"nested":{"path":"features","query":{"bool":{"must":[{"term":{"features.type.keyword":"SPA_POOL_JACUZZI_HOT_TUB"}}]}}}}]}}]}}]}}`),
		}

		if len(DeDuplicateJsonLines(lines)) != 1 {
			t.Fail()
		}
	})
}
