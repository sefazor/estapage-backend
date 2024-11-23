// pkg/utils/location/location.go
package location

import (
	"encoding/json"
	"os"
)

type Country struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	ISO2         string `json:"iso2"` // TR, US gibi
	ISO3         string `json:"iso3"` // TUR, USA gibi
	PhoneCode    string `json:"phonecode"`
	Capital      string `json:"capital"`
	Currency     string `json:"currency"`
	CurrencyName string `json:"currency_name"`
	Region       string `json:"region"`
}

type State struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	CountryID   uint   `json:"country_id"`
	CountryCode string `json:"country_code"` // TR, US gibi
	StateCode   string `json:"state_code"`   // 34, CA gibi
}

type City struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	StateID   uint   `json:"state_id"`
	StateCode string `json:"state_code"`
}

var (
	countries []Country
	states    []State
	cities    []City
)

// Init başlangıçta JSON dosyalarını yükler
func Init() error {
	// Ülkeleri yükle
	cData, err := os.ReadFile("pkg/data/countries.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(cData, &countries); err != nil {
		return err
	}

	// Eyaletleri yükle
	sData, err := os.ReadFile("pkg/data/states.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(sData, &states); err != nil {
		return err
	}

	// Şehirleri yükle
	ciData, err := os.ReadFile("pkg/data/cities.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(ciData, &cities)
}

// GetCountries tüm ülkeleri döner
func GetCountries() []Country {
	return countries
}

// GetStatesByCountry belirli bir ülkenin eyaletlerini/illerini döner
func GetStatesByCountry(countryCode string) []State {
	var countryStates []State
	for _, state := range states {
		if state.CountryCode == countryCode {
			countryStates = append(countryStates, state)
		}
	}
	return countryStates
}

// GetCitiesByState belirli bir eyaletin/ilin şehirlerini/ilçelerini döner
func GetCitiesByState(stateCode string) []City {
	var stateCities []City
	for _, city := range cities {
		if city.StateCode == stateCode {
			stateCities = append(stateCities, city)
		}
	}
	return stateCities
}
