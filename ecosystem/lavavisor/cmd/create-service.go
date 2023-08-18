package lavavisor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	processmanager "github.com/lavanet/lava/ecosystem/lavavisor/pkg/process"
	lvutil "github.com/lavanet/lava/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func CreateLavaVisorCreateServiceCobraCommand() *cobra.Command {
	cmdLavavisorCreateService := &cobra.Command{
		Use:   `create-service [service-config-folder]`,
		Short: "generates service files for each provider/consumer in the config.yml.",
		Long: `The 'create-service' command generates system service files for each provider 
		and consumer specified in the config.yml file. Once these service files are created, 
		the 'lavavisor start' command can utilize them to manage (enable, restart, and check the status of) 
		each service using the 'systemctl' tool. This ensures that each service is properly integrated with 
		the system's service manager, allowing for robust management and monitoring of the LavaVisor services.
		Each service file inside [service-config-folder] must be named exactly the same with corresponsing service name
		defined in config.yml`,
		Args: cobra.ExactArgs(1),
		Example: `required flags: --geolocation | --from
			optional flags: --log-level  | --node  | --keyring-backend
			lavavisor create-service ./config --geolocation 1 --from alice --log-level warn
			lavavisor create-service ./config --geolocation 1 --from bob --log-level info`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. read config.yml -> this will tell us what service files this command will create
			dir, _ := cmd.Flags().GetString("directory")
			// Build path to ./lavavisor
			lavavisorPath, err := processmanager.ValidateLavavisorDir(dir)
			if err != nil {
				return err
			}
			// create services folder in lavavisorPath
			lavavisorServicesDir := lavavisorPath + "/services"
			err = os.MkdirAll(lavavisorServicesDir, 0755) // 0755 is a common permission for directories
			if err != nil {
				return utils.LavaFormatError("failed to create services directory", err)
			}
			// Read config.yml -> it must be under ./lavavisor , we don't accept alternative paths
			configPath := filepath.Join(lavavisorPath, "/config.yml")
			configData, err := os.ReadFile(configPath)
			if err != nil {
				return utils.LavaFormatError("failed to read config.yaml", err)
			}
			var config Config
			err = yaml.Unmarshal(configData, &config)
			if err != nil {
				return utils.LavaFormatError("failed to unmarshal config.yaml", err)
			}

			// service logs dir:
			logsDir := lavavisorServicesDir + "/logs"
			err = os.MkdirAll(logsDir, 0755) // 0755 is a common permission for directories
			if err != nil {
				return utils.LavaFormatError("failed to create service logs directory", err)
			}

			// service logs dir:
			lavavisorServiceConfigDir := lavavisorServicesDir + "/service_configs"
			err = os.MkdirAll(lavavisorServiceConfigDir, 0755) // 0755 is a common permission for directories
			if err != nil {
				return utils.LavaFormatError("failed to create service logs directory", err)
			}
			// Iterate over the list of services and start them
			for _, process := range config.Services {
				utils.LavaFormatInfo("Creating the service file", utils.Attribute{Key: "Process", Value: process})
				CreateServiceFile(cmd, args, process, lavavisorServicesDir, logsDir, lavavisorServiceConfigDir)
			}
			// 3. if sanity check passes on command inputs, create the service file inside ./lavavisor/upgrades/<version-no>/ directory

			// 4. wherever the service config file was initially, copy it to ./lavavisor/upgrades/<version-no>/ as well

			// 5. fix 'start' command and only validate if service file exist & return error if not found
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmdLavavisorCreateService)
	cmdLavavisorCreateService.MarkFlagRequired(flags.FlagFrom)
	cmdLavavisorCreateService.Flags().Uint64(common.GeolocationFlag, 0, "geolocation to run from")
	cmdLavavisorCreateService.MarkFlagRequired(common.GeolocationFlag)
	cmdLavavisorCreateService.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorCreateService.Flags().String(flags.FlagLogLevel, "debug", "log level")
	return cmdLavavisorCreateService
}

func CreateServiceFile(cmd *cobra.Command, args []string, processName string, servicesDir string, logsDir string, lavavisorServiceConfigDir string) error {
	// working dir:
	out, err := exec.LookPath("lava-protocol")
	if err != nil {
		return utils.LavaFormatError("could not detect a linked lava-protocol binary", err)
	}
	workingDir := strings.TrimSpace(filepath.Dir(out) + "/")
	fmt.Println("workingDir: ", workingDir)
	// service type - provider or consumer
	serviceType, err := determineServiceType(processName)
	if err != nil {
		return err
	}
	// service config yml file path
	serviceConfigPath := args[0]
	serviceConfigFile := filepath.Join(serviceConfigPath, processName+".yml")
	if _, err := os.Stat(serviceConfigFile); err == nil {
		fmt.Println("Found service config file for", processName, "at", serviceConfigPath)
		// You can now read the YAML file, create the service, etc.
	} else {
		fmt.Println("Service config file not found for", processName)
	}
	lavavisorServiceConfig := filepath.Join(lavavisorServiceConfigDir, processName+".yml")
	err = lvutil.Copy(serviceConfigFile, lavavisorServiceConfig)
	if err != nil {
		return utils.LavaFormatError("couldn't copy binary to system path", err)
	}
	// from user
	fromUser, _ := cmd.Flags().GetString(flags.FlagFrom)
	// keyring backend
	keyringBackend, _ := cmd.Flags().GetString(flags.FlagKeyringBackend)
	// chainId
	chainID := strings.Split(processName, "-")[1]
	// geolocation
	geoLocation, _ := cmd.Flags().GetUint64(lavasession.GeolocationFlag)
	// log-level
	logLevel, _ := cmd.Flags().GetString(flags.FlagLogLevel)
	// log-level
	node, _ := cmd.Flags().GetString(flags.FlagNode)

	content := "[Unit]\n"
	content += "\tDescription=" + processName + " " + serviceType + " daemon\n"
	content += "\tAfter=network-online.target\n\n"
	content += "[Service]\n"
	content += "\tWorkingDirectory=" + workingDir + "\n"
	if serviceType == "consumer" {
		content += "\tExecStart=lava-protocol rpcconsumer " + lavavisorServiceConfig + " --from " + fromUser + " --keyring-backend " + keyringBackend + " --chain-id " + chainID + " --geolocation " + fmt.Sprint(geoLocation) + " --log_level " + logLevel + " --node " + node + "\n"
	} else if serviceType == "provider" {
		content += "\tExecStart=lava-protocol rpcprovider " + lavavisorServiceConfig + " --from " + fromUser + " --keyring-backend " + keyringBackend + " --chain-id " + chainID + " --geolocation " + fmt.Sprint(geoLocation) + " --log_level " + logLevel + " --node " + node + "\n"
	}
	content += "\tUser=ubuntu\n"
	content += "\tRestart=always\n"
	content += "\tRestartSec=180\n"
	content += "\tLimitNOFILE=infinity\n"
	content += "\tLimitNPROC=infinity\n"
	content += "\tStandardOutput=append:" + logsDir + "/" + processName + ".log\n\n"
	content += "[Install]\n"
	content += "\tWantedBy=multi-user.target"

	filePath := servicesDir + "/" + processName + ".service"
	err = os.WriteFile(filePath, []byte(content), os.ModePerm)
	if err != nil {
		return utils.LavaFormatError("error writing to service file", err)
	}

	return nil
}

func determineServiceType(service string) (string, error) {
	if strings.Contains(service, "consumer") {
		return "consumer", nil
	} else if strings.Contains(service, "provider") {
		return "provider", nil
	}
	return "", utils.LavaFormatError("invalid service type", nil)
}
