package main

import (
	"fmt"
	"strconv"
	"strings"
)

type GPSCoordinates struct {
	Latitude    float64
	Longitude   float64
	HasLocation bool
}

// parseRational converti un chaine number/denominateur en float64
func parseRational(rational string) (float64, error) {
	parts := strings.Split(rational, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid rational format: %s", rational)
	}

	num, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	denom, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}

	if denom == 0 {
		return 0, fmt.Errorf("division by zero")
	}

	return num / denom, nil
}

// GPS: "48/1, 51/1, 2964/100" je le recupere sous ce format donc fonction pour parser
func parseRationalList(rationalList string) ([]float64, error) {
	parts := strings.Split(rationalList, ", ")
	var values []float64

	for _, part := range parts {
		val, err := parseRational(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		values = append(values, val)
	}

	return values, nil
}

// dmsToDecimal convertit des coordonnées DMS (Degrees, Minutes, Seconds) en decimal
// objectif passer de  [degrer, minutes, secondes] a decimal  formule decimal = deg + (min/60) + (sec/3600)
func dmsToDecimal(dms []float64) float64 {
	if len(dms) != 3 {
		return 0
	}
	return dms[0] + (dms[1] / 60.0) + (dms[2] / 3600.0)
}

func extractGPSCoordinates(tags map[string]string) GPSCoordinates {
	gps := GPSCoordinates{HasLocation: false}

	latStr, hasLat := tags["GPSLatitude"]
	lonStr, hasLon := tags["GPSLongitude"]
	latRef, hasLatRef := tags["GPSLatitudeRef"]
	lonRef, hasLonRef := tags["GPSLongitudeRef"]

	if !hasLat || !hasLon || !hasLatRef || !hasLonRef {
		return gps
	}

	// recupere les 3 valeurs GPS (degrer, minutes, secondes)
	latDMS, err := parseRationalList(latStr)
	if err != nil || len(latDMS) != 3 {
		return gps
	}

	lonDMS, err := parseRationalList(lonStr)
	if err != nil || len(lonDMS) != 3 {
		return gps
	}

	// je converti DMS en Decimal
	lat := dmsToDecimal(latDMS)
	lon := dmsToDecimal(lonDMS)

	// N/S/E/W
	if latRef == "S" {
		lat = -lat
	}
	if lonRef == "W" {
		lon = -lon
	}

	gps.Latitude = lat
	gps.Longitude = lon
	gps.HasLocation = true

	return gps
}

func formatGPSCoordinates(gps GPSCoordinates) string {
	if !gps.HasLocation {
		return ""
	}
	return fmt.Sprintf("%.6f, %.6f", gps.Latitude, gps.Longitude)
}

func getOpenStreetMapURL(gps GPSCoordinates) string {
	if !gps.HasLocation {
		return ""
	}
	// Example de lien: https://www.openstreetmap.org/?mlat=48.0&mlon=2.0#map=15/48.0/2.0
	return fmt.Sprintf("https://www.openstreetmap.org/?mlat=%.6f&mlon=%.6f#map=15/%.6f/%.6f",
		gps.Latitude, gps.Longitude, gps.Latitude, gps.Longitude)
}

func printGPSInfo(tags map[string]string) {
	gps := extractGPSCoordinates(tags)

	if !gps.HasLocation {
		return
	}

	coords := formatGPSCoordinates(gps)
	PrintMetadata("GPS Coordinates", coords)
	osmURL := getOpenStreetMapURL(gps)
	PrintMetadata("OpenStreetMap", osmURL)
}
