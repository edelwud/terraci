package costutil

// HoursPerMonth is the average number of hours in a month for cost calculations.
const HoursPerMonth = 730

// HourlyCost returns hourly and monthly costs from an hourly rate.
func HourlyCost(pricePerHour float64) (hourly, monthly float64) {
	return pricePerHour, pricePerHour * HoursPerMonth
}

// ScaledHourlyCost returns hourly and monthly costs for multiple units at an hourly rate.
func ScaledHourlyCost(pricePerHour float64, count int) (hourly, monthly float64) {
	h := pricePerHour * float64(count)
	return h, h * HoursPerMonth
}

// FixedMonthlyCost returns hourly and monthly costs from a fixed monthly rate.
func FixedMonthlyCost(monthly float64) (hourly, monthlyCost float64) {
	return monthly / HoursPerMonth, monthly
}
