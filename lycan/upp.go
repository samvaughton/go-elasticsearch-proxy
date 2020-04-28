package lycan

import (
	"crypto/sha1"
	"fmt"
	"github.com/tidwall/gjson"
	"net/url"
	"strconv"
	"time"
)

type PriceRequestData struct {
	Property PropertyData      `json:"property"`
	Context  PriceContextData  `json:"context"`
	Response PriceResponseData `json:"response"`
}

type PropertyData struct {
	Name string `json:"name"`
	Uuid string `json:"uuid"`
}

type PriceContextData struct {
	Currency    string        `json:"currency"`
	Guests      GuestData     `json:"guests"`
	DateRange   DateRangeData `json:"dateRange"`
	Fingerprint string        `json:"fingerprint"`
}

type GuestData struct {
	Total    int `json:"total"`
	Adults   int `json:"adults"`
	Children int `json:"children"`
	Infants  int `json:"infants"`
	Pets     int `json:"pets"`
}

type DateRangeData struct {
	ArrivalDate   string `json:"arrivalDate"`
	DepartureDate string `json:"departureDate"`
	Nights        int    `json:"nights"`
}

type PriceResponseData struct {
	IsAvailable   bool    `json:"isAvailable"`
	IsPriced      bool    `json:"isPriced"`
	BookableType  string  `json:"bookableType"`
	Total         float64 `json:"total"`
	BasePrice     float64 `json:"basePrice"`
	DamageDeposit float64 `json:"damageDeposit"`
	Currency      string  `json:"currency"`
	StatusCode    int     `json:"statusCode"`
}

func CalculateNights(arrivalDate string, departureDate string) int {
	aParsed, _ := time.Parse("2006-01-02", arrivalDate)
	dParsed, _ := time.Parse("2006-01-02", departureDate)

	var nights float64
	if arrivalDate == departureDate {
		nights = 0
	} else {
		nights = (dParsed.Sub(aParsed)).Hours() / 24
	}

	return int(nights)
}

/*
 * This function generates a fingerprint based on the unique vectors of a price request.
 * These vectors being property ID, arrival & departure date, total guests & pets
 */
func GenerateFingerprint(params url.Values) string {
	sha := sha1.New()

	sha.Write([]byte(GetQueryString("propertyUuid", params)))
	sha.Write([]byte(GetQueryString("arrivalDate", params)))
	sha.Write([]byte(GetQueryString("departureDate", params)))
	sha.Write([]byte(GetQueryString("adults", params)))
	sha.Write([]byte(GetQueryString("pets", params)))

	return fmt.Sprintf("%x", sha.Sum(nil))
}

func ExtractPriceRequestData(params url.Values, priceResponse gjson.Result, statusCode int) PriceRequestData {
	return PriceRequestData{
		Property: PropertyData{
			Name: GetQueryString("propertyName", params),
			Uuid: GetQueryString("propertyUuid", params),
		},
		Context: PriceContextData{
			Currency: GetQueryString("currency", params),
			Fingerprint: GenerateFingerprint(params),
			Guests: GuestData{
				Pets:     GetQueryInteger("pets", params),
				Adults:   GetQueryInteger("adults", params),
				Children: GetQueryInteger("children", params),
				Infants:  GetQueryInteger("infants", params),
				Total:    GetQueryInteger("adults", params) + GetQueryInteger("children", params) + GetQueryInteger("infants", params),
			},
			DateRange: DateRangeData{
				ArrivalDate:   GetQueryString("arrival", params),
				DepartureDate: GetQueryString("departure", params),
				Nights:        CalculateNights(GetQueryString("arrival", params), GetQueryString("departure", params)),
			},
		},
		Response: PriceResponseData{
			IsAvailable:   priceResponse.Get("isAvailable").Bool(),
			IsPriced:      priceResponse.Get("isPriced").Bool(),
			BookableType:  priceResponse.Get("bookableType").String(),
			Total:         priceResponse.Get("total").Float(),
			BasePrice:     priceResponse.Get("basePrice").Float(),
			DamageDeposit: priceResponse.Get("damageDeposit").Float(),
			Currency:      priceResponse.Get("currency").String(),
			StatusCode:    statusCode,
		},
	}
}

func GetQueryString(key string, params url.Values) string {
	return params.Get(key)
}

func GetQueryInteger(key string, params url.Values) int {
	string := GetQueryString(key, params)

	if string == "" {
		return 0
	}

	number, err := strconv.Atoi(string)

	if err != nil {
		return 0
	}

	return number
}
