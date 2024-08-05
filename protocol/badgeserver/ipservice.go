package badgeserver

import (
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/praserx/ipconv"

	"github.com/lavanet/lava/v2/utils"
)

type IpService struct {
	DefaultGeolocation int
	CountryCsvFilePath string
	IpTsvFilePath      string
	IpCountryData      *[]*IpData
}

func InitIpService(defaultGeolocation int, countriesFilePath, ipFilePath string) (*IpService, error) {
	service := IpService{DefaultGeolocation: defaultGeolocation, CountryCsvFilePath: countriesFilePath, IpTsvFilePath: ipFilePath}

	err := service.ReadIpTsvFileData()
	if err != nil {
		utils.LavaFormatError("failed to setup the ip service.", err)
	}
	return &service, err
}

func (service *IpService) readCountryCsvFileData() (*map[string]int, error) {
	countries := make(map[string]int)
	if len(service.CountryCsvFilePath) == 0 {
		utils.LavaFormatWarning("badge is not configured correctly- missing country csv file path.", nil)
		return &countries, nil
	}
	file, err := os.Open(service.CountryCsvFilePath)
	if err != nil {
		utils.LavaFormatError("error opening country file", err)
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'

	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		countries[row[0]], _ = strconv.Atoi(row[len(row)-1])
	}
	return &countries, nil
}

func (service *IpService) ReadIpTsvFileData() error {
	counties, err := service.readCountryCsvFileData()
	if err != nil {
		utils.LavaFormatError("error reading country data.", err)
		return err
	}
	if len(service.IpTsvFilePath) == 0 {
		utils.LavaFormatWarning("badge is not configured correctly- missing ip tsv file path.", nil)
		service.IpCountryData = &[]*IpData{}
		return nil
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
	sort.Slice(result, func(i, j int) bool {
		return result[i].FromIp < result[j].FromIp
	})
	service.IpCountryData = &result
	return nil
}

func (service *IpService) SearchForIp(toSearchIp string) (*IpData, error) {
	if len(*service.IpCountryData) == 0 {
		return nil, fmt.Errorf("ip servive not configured correctly")
	}
	needle, err := convertStringToIpInt(toSearchIp)
	if err != nil {
		return nil, err
	}
	haystack := *service.IpCountryData // for simplicity
	low := 0
	high := len(haystack) - 1

	for low <= high {
		median := (low + high) / 2
		if haystack[median].FromIp < needle {
			low = median + 1
		} else {
			high = median - 1
		}
	}

	if low == len(haystack) {
		return nil, fmt.Errorf("ip not found")
	}
	if needle >= haystack[low].FromIp && needle <= haystack[low].ToIP {
		return haystack[low], nil
	}
	if needle >= haystack[high].FromIp && needle <= haystack[high].ToIP {
		return haystack[high], nil
	}

	// theoretically we should never come here but, I added this to be on the safe side
	for low <= high {
		if needle >= haystack[low].FromIp && needle <= haystack[low].ToIP {
			return haystack[low], nil
		}
		low++
	}
	return nil, fmt.Errorf("ip not found")
}

func convertRowToIpModel(rowData string) (*IpData, error) {
	convertRowWith4tabs := func(ipStringDatas []string) (*IpData, error) {
		ipSorce := strings.Split(ipStringDatas[0], " ")
		if len(ipSorce) != 2 {
			return nil, fmt.Errorf("unexpeted ip range on  tsv data format. expected 2 separated with space(' ')")
		}
		fromIpData, err := convertStringToIpInt(ipSorce[0])
		if err != nil {
			return nil, err
		}
		toIpData, err := convertStringToIpInt(ipSorce[1])
		if err != nil {
			return nil, err
		}
		return &IpData{
			FromIp:      fromIpData,
			ToIP:        toIpData,
			CountryCode: ipStringDatas[2],
		}, nil
	}
	convertRowWith5tabs := func(ipStringDatas []string) (*IpData, error) {
		fromIpData, err := convertStringToIpInt(ipStringDatas[0])
		if err != nil {
			return nil, err
		}
		toIpData, err := convertStringToIpInt(ipStringDatas[1])
		if err != nil {
			return nil, err
		}
		return &IpData{
			FromIp:      fromIpData,
			ToIP:        toIpData,
			CountryCode: ipStringDatas[3],
		}, nil
	}

	ipStringDatas := strings.Split(rowData, "\t")
	if len(ipStringDatas) == 4 {
		return convertRowWith4tabs(ipStringDatas)
	} else if len(ipStringDatas) == 5 {
		return convertRowWith5tabs(ipStringDatas)
	} else {
		return nil, fmt.Errorf("invalid tsv data format. expected 4")
	}
}

func convertStringToIpInt(sc string) (int64, error) {
	if len(sc) == 0 {
		return 0, fmt.Errorf("invalid ip length to convert to number")
	}
	ip := net.ParseIP(sc)
	if ip == nil {
		return 0, fmt.Errorf("converting ip didn't succeed")
	}
	number, err := ipconv.IPv4ToInt(ip)
	return int64(number), err
}
