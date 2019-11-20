package fio

var maxFees = map[string]float64{
	"regaddress":   5.0,
	"addaddress":   1.0,
	"regdomain":    50.0,
	"renewdomain":  50.0,
	"renewaddress": 5.0,
	"burnexpired":  0.3,
	"setdomainpub": 0.3,
	"transfer":     0.3,
	"trnsfiopubky": 0.3,
	"recordsend":   0.3,
	"newfundsreq":  0.3,
	"rejectfndreq": 0.3,
}

// ConvertAmount is a convenience function for converting from a float for human readability
func ConvertAmount(tokens float64) uint64 {
	return uint64(tokens * 1000000000.0)
}

