package validators

import (
	"encoding/csv"
	"os"
	"strconv"

	"cosmossdk.io/math"
)

type ChainInfo struct {
	Name          string
	blocksInMonth int64
	archiveNode   string
}

var Chains = []ChainInfo{
	{"dYdX", 2592000, "https://dydx-mainnet-archive-rpc.public.blastapi.io:443"},
}

type MissingValsData struct {
	Val   string
	Chain string
	Err   string
}

func (mvd MissingValsData) String() []string {
	return []string{mvd.Val, mvd.Chain, mvd.Err}
}

type ValsData struct {
	Validator string
	Chain     string
	Jailed    int64
	Downtime  math.LegacyDec
	VotePower math.LegacyDec
}

func (pd ValsData) String() []string {
	return []string{pd.Validator, pd.Chain, strconv.FormatInt(pd.Jailed, 10), pd.Downtime.String(), pd.VotePower.String()}
}

var Validators = []string{
	"validator_moniker",
}

// Export array of structs to CSV file of missing validators (validator,chain,err)
func ExportToCSVMissingValidators(filename string, data []MissingValsData) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV data
	values := [][]string{}
	for _, d := range data {
		values = append(values, d.String())
	}

	if err := writer.WriteAll(values); err != nil {
		return err
	}

	return nil

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
	values := [][]string{}
	for _, d := range data {
		values = append(values, d.String())
	}

	if err := writer.WriteAll(values); err != nil {
		return err
	}

	return nil

}
