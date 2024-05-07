package validators

import (
	"encoding/csv"
	"os"
	"strconv"

	"cosmossdk.io/math"
)

type ValsData struct {
	Validator string
	Jailed    int64
	Downtime  math.LegacyDec
	VotePower math.LegacyDec
}

func (pd ValsData) String() []string {
	return []string{pd.Validator, strconv.FormatInt(pd.Jailed, 10), pd.Downtime.String(), pd.VotePower.String()}
}

// Export array of structs to CSV file of validators (validator,chain,amount-of-times-jailed,downtime-percentage,vote-power)
func ExportToCSVValidators(filename string, data []ValsData) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV data
	values := [][]string{{"validator", "jailed", "downtime", "voting power"}}
	for _, d := range data {
		values = append(values, d.String())
	}

	if err := writer.WriteAll(values); err != nil {
		return err
	}

	return nil
}
