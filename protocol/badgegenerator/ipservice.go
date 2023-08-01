package badgegenerator

import (
	"encoding/csv"
	"fmt"
	"github.com/lavanet/lava/utils"
	"io"
	"os"
	"strconv"
	"strings"
)

type IpService struct {
	DefaultGeolocation int
	CountryCsvFilePath string
	IpTsvFilePath      string
	IpCountryData      *[]*IpData
}

func InitIpService(defaultGeolocation int, countriesFilePath string, ipFilePath string) (*IpService, error) {
	service := IpService{DefaultGeolocation: defaultGeolocation, CountryCsvFilePath: countriesFilePath, IpTsvFilePath: ipFilePath}

	err := service.ReadIpTsvFileData()
	if err != nil {
		utils.LavaFormatError("failed to setup the ip service.", err)
	}
	return &service, err
}
func (service *IpService) readCountryCsvFileData() (*map[string]int, error) {
	countries := make(map[string]int)
	file, err := os.Open(service.CountryCsvFilePath)
	if err != nil {
		utils.LavaFormatError("error opening country file", err)
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		for _, rowData := range row {
			countryData := strings.Split(rowData, ";")
			countries[countryData[0]], _ = strconv.Atoi(countryData[len(countryData)-1])
		}
	}
	return &countries, nil
}

func (service *IpService) ReadIpTsvFileData() error {
	counties, err := service.readCountryCsvFileData()
	if err != nil {
		utils.LavaFormatError("error reading country data.", err)
		return err
	}
	file, err := os.Open(service.IpTsvFilePath)
	if err != nil {
		utils.LavaFormatError("error opening ip file.", err)
		return err
	}
	defer file.Close()
	var result []*IpData
	// Create a new CSV reader
	reader := csv.NewReader(file)
	// Read and print each row
	for {
		row, err := reader.Read()
		if err != nil {
			if len(row) > 1 {
				row = []string{strings.Join(row, "")} // this is a necessary correction
			} else if err == io.EOF {
				break
			} else {
				return err
			}

		}
		for _, rowData := range row {
			ipData, err := convertRowToIpModel(rowData)
			if err != nil {
				utils.LavaFormatWarning("error reading ip data", err)
				continue
			}
			geolocation, exist := (*counties)[ipData.CountryCode]
			if !exist {
				geolocation = service.DefaultGeolocation
			}
			ipData.Geolocation = geolocation
			result = append(result, ipData)
		}
	}
	service.IpCountryData = &result
	return nil
}

func (service *IpService) SearchForIp(toSearchIp string) (*IpData, error) {
	ipData := convertStringToIpModel(toSearchIp)
	if ipData == nil {
		return nil, fmt.Errorf("invalid ip format")
	}
	if len(*service.IpCountryData) == 0 {
		return nil, fmt.Errorf("invalid service configuration")
	}

	for _, ip := range *service.IpCountryData {
		//for better readability this is done in different ifs
		if ipData.Group1 >= ip.FromIp.Group1 && ipData.Group1 <= ip.ToIP.Group1 {
			//group 2
			if ipData.Group2 >= ip.FromIp.Group2 && ipData.Group2 <= ip.ToIP.Group2 {
				//gr 3
				if ipData.Group3 >= ip.FromIp.Group3 && ipData.Group3 <= ip.ToIP.Group3 {
					//gr 4
					if ipData.Group4 >= ip.FromIp.Group4 && ipData.Group4 <= ip.ToIP.Group4 {
						return ip, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("ip not found")
}

func convertRowToIpModel(rowData string) (*IpData, error) {

	convertRowWith4tabs := func(ipStringDatas []string) (*IpData, error) {
		ipSorce := strings.Split(ipStringDatas[0], " ")
		if len(ipSorce) != 2 {
			return nil, fmt.Errorf("Unexpeted ip range on  tsv data format. expected 2 seperated with space(' ')")
		}
		return &IpData{
			FromIp:      convertStringToIpModel(ipSorce[0]),
			ToIP:        convertStringToIpModel(ipSorce[1]),
			CountryCode: ipStringDatas[2],
		}, nil
	}
	convertRowWith5tabs := func(ipStringDatas []string) (*IpData, error) {
		return &IpData{
			FromIp:      convertStringToIpModel(ipStringDatas[0]),
			ToIP:        convertStringToIpModel(ipStringDatas[1]),
			CountryCode: ipStringDatas[3],
		}, nil
	}

	ipStringDatas := strings.Split(rowData, "\t")
	if len(ipStringDatas) == 4 {
		return convertRowWith4tabs(ipStringDatas)
	} else if len(ipStringDatas) == 5 {
		return convertRowWith5tabs(ipStringDatas)
	} else {
		return nil, fmt.Errorf("Invalid tsv data format. expected 4")
	}

}

func convertStringToIpModel(sc string) *Ip {
	if len(sc) == 0 {
		return nil
	}
	ipGroups := strings.Split(sc, ".")
	if len(ipGroups) != 4 {
		return nil
	}
	group1, _ := strconv.Atoi(ipGroups[0])
	group2, _ := strconv.Atoi(ipGroups[1])
	group3, _ := strconv.Atoi(ipGroups[2])
	group4, _ := strconv.Atoi(ipGroups[3])

	return &Ip{
		Group1: group1,
		Group2: group2,
		Group3: group3,
		Group4: group4,
	}
}
