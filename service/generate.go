package service

import "github.com/s3nkyh/arcticeroute/models"

var (
	dikson = models.Point{
		"Dikson",
		68.97,
		68.97,
	}
	arkhangelsk = models.Point{
		"Arkhangelsk",
		64.54,
		40.51,
	}
	kaninNos = models.Point{
		"Kanin Nos",
		68.65,
		43.26,
	}
	murmansk = models.Point{
		"Murmansk",
		68.97,
		33.07,
	}
)

func GetPoints() []models.Point {
	return []models.Point{dikson, arkhangelsk, kaninNos, murmansk}
}
