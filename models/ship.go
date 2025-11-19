package models

type Ship struct {
	MMSI      int32   `json:"mmsi"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
