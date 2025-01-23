package services

func GetLocations() ([]Location, error) {
	return _getLocations(), nil
}

func _getLocations() []Location {
	var result []Location

	result = append(result, Location{Id: "houston", Name: "Houston", Lat: "5454545", Lon: "123456"})
	result = append(result, Location{Id: "cypress", Name: "Cypress", Lat: "5454545", Lon: "123456"})
	result = append(result, Location{Id: "katy", Name: "Katy", Lat: "5454545", Lon: "123456"})

	return result
}
