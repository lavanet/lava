package lavavisor

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	processmanager "github.com/lavanet/lava/v2/ecosystem/lavavisor/pkg/process"
	lvutil "github.com/lavanet/lava/v2/ecosystem/lavavisor/pkg/util"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy"
	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ServiceTypeProvider = "provider"
	ServiceTypeConsumer = "consumer"
)

type ServiceParams struct {
	ServiceType               string
	ServiceConfigFile         string
	FromUser                  string
	KeyringBackend            string
	ChainID                   string
	GeoLocation               uint64
	LogLevel                  string
	Node                      string
	LavavisorServicesDir      string
	LavavisorLogsDir          string
	LavavisorServiceConfigDir string
	ParallelConnection        uint64
}

func CreateLavaVisorCreateServiceCobraCommand() *cobra.Command {
	cmdLavavisorCreateService := &cobra.Command{
		Use:   `create-service [service-type: "provider" or "consumer"] [service-config-folder]`,
		Short: "generates service files for each provider/consumer in the config.yml.",
		Long: `The 'create-service' command generates system service files for provider 
		and consumer processes. Once these service files are created, 
		the 'lavavisor start' command can utilize them to manage (enable, restart, and check the status of) 
		each service using the 'systemctl' command.
		After a service file is created, the name of the service is added to "config.yml" file inside Lavavisor directory.`,
		Args: cobra.ExactArgs(2),
		Example: `required flags: --geolocation | --from | --chain-id | --keyring-backend
			optional flags: --log-level  | --node  | --keyring-backend
			lavavisor create-service provider ./config --geolocation 1 --from alice --log-level warn
			lavavisor create-service consumer ./config --geolocation 1 --from bob --log-level info`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. read config.yml -> this will tell us what service files this command will create
			dir, _ := cmd.Flags().GetString("directory")

			binariesFetcher := processmanager.ProtocolBinaryFetcher{}
			// Build path to ./lavavisor
			lavavisorPath, err := binariesFetcher.ValidateLavavisorDir(dir)
			if err != nil {
				return err
			}
			// .lavavisor/ main services dir
			lavavisorServicesDir := lavavisorPath + "/services"
			err = os.MkdirAll(lavavisorServicesDir, 0o755)
			if err != nil {
				return utils.LavaFormatError("[Lavavisor] failed to create services directory", err)
			}

			// .lavavisor/ service logs dir
			lavavisorLogsDir := lavavisorServicesDir + "/logs"
			err = os.MkdirAll(lavavisorLogsDir, 0o755)
			if err != nil {
				return utils.LavaFormatError("[Lavavisor] failed to create service logs directory", err)
			}

			// .lavavisor/ service config dir
			lavavisorServiceConfigDir := lavavisorServicesDir + "/protocol_yml_configs"
			err = os.MkdirAll(lavavisorServiceConfigDir, 0o755)
			if err != nil {
				return utils.LavaFormatError("[Lavavisor] failed to create service logs directory", err)
			}

			// GET SERVICE PARAMS
			serviceType := args[0]
			if serviceType != ServiceTypeProvider && serviceType != ServiceTypeConsumer {
				return utils.LavaFormatError("[Lavavisor] invalid service type, must be provider or consumer", nil)
			}
			serviceConfigFile := args[1] // the path that contains provider or consumer's configuration yml file
			// from user
			fromUser, _ := cmd.Flags().GetString(flags.FlagFrom)
			// keyring backend
			keyringBackend, _ := cmd.Flags().GetString(flags.FlagKeyringBackend)
			// chainId
			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			// geolocation
			geoLocation, _ := cmd.Flags().GetUint64(lavasession.GeolocationFlag)
			// log-level
			logLevel, _ := cmd.Flags().GetString(flags.FlagLogLevel)
			// log-level
			node, _ := cmd.Flags().GetString(flags.FlagNode)

			parallelConnection, _ := cmd.Flags().GetUint(chainproxy.ParallelConnectionsFlag)

			serviceParams := &ServiceParams{
				ServiceType:               serviceType,
				ServiceConfigFile:         serviceConfigFile,
				FromUser:                  fromUser,
				KeyringBackend:            keyringBackend,
				ChainID:                   chainID,
				GeoLocation:               geoLocation,
				LogLevel:                  logLevel,
				Node:                      node,
				LavavisorServicesDir:      lavavisorServicesDir,
				LavavisorLogsDir:          lavavisorLogsDir,
				LavavisorServiceConfigDir: lavavisorServiceConfigDir,
				ParallelConnection:        uint64(parallelConnection),
			}

			utils.LavaFormatInfo("[Lavavisor] Creating the service file")

			filename := filepath.Base(serviceConfigFile)
			configName := filename[0 : len(filename)-len(filepath.Ext(filename))]
			configPath := filepath.Dir(serviceConfigFile)

			viper.SetConfigName(configName)
			viper.SetConfigType("yml")
			viper.AddConfigPath(configPath)

			// Read the configuration file
			err = viper.ReadInConfig()
			if err != nil {
				return utils.LavaFormatError("[Lavavisor] Error reading config file", err)
			}

			createLink, err := cmd.Flags().GetBool("create-link")
			if err != nil {
				return err
			}

			serviceFileName, err := CreateServiceFile(serviceParams, createLink)
			if err != nil {
				return err
			}
			// Write the name of the service into .lavavisor/config.yml
			return WriteToConfigFile(lavavisorPath, serviceFileName)
		},
	}
	flags.AddTxFlagsToCmd(cmdLavavisorCreateService)
	cmdLavavisorCreateService.Flags().Bool("create-link", false, "Creates a symbolic link to the /etc/systemd/system/ directory")
	cmdLavavisorCreateService.Flags().Uint64(common.GeolocationFlag, 0, "geolocation to run from")
	cmdLavavisorCreateService.Flags().String("directory", os.ExpandEnv("~/"), "Protocol Flags Directory")
	cmdLavavisorCreateService.Flags().String(flags.FlagLogLevel, "debug", "log level")
	cmdLavavisorCreateService.Flags().Uint(chainproxy.ParallelConnectionsFlag, chainproxy.NumberOfParallelConnections, "parallel connections")
	cmdLavavisorCreateService.MarkFlagRequired(flags.FlagFrom)
	cmdLavavisorCreateService.MarkFlagRequired(flags.FlagChainID)
	cmdLavavisorCreateService.MarkFlagRequired(flags.FlagKeyringBackend)
	cmdLavavisorCreateService.MarkFlagRequired(common.GeolocationFlag)
	return cmdLavavisorCreateService
}

func CreateServiceFile(serviceParams *ServiceParams, createLink bool) (string, error) {
	// working dir:
	out, err := exec.LookPath("lavap")
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] could not detect a linked lavap binary", err)
	}
	workingDir := strings.TrimSpace(filepath.Dir(out) + "/")

	if _, err := os.Stat(serviceParams.ServiceConfigFile); err != nil {
		return "", utils.LavaFormatError("[Lavavisor] Service config file not found", err)
	}

	configChainID := viper.GetString("endpoints.0.chain-id")
	serviceId := serviceParams.FromUser + "-" + configChainID
	configPath := serviceParams.LavavisorServiceConfigDir + "/" + filepath.Base(serviceParams.ServiceConfigFile)

	err = lvutil.Copy(serviceParams.ServiceConfigFile, configPath)
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] couldn't copy binary to system path", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] Could not get current user", err)
	}

	content := "[Unit]\n"
	content += "  Description=" + serviceId + " daemon\n"
	content += "  After=network-online.target\n"
	content += "[Service]\n"
	content += "  WorkingDirectory=" + workingDir + "\n"
	if serviceParams.ServiceType == ServiceTypeConsumer {
		content += "  ExecStart=" + workingDir + "lavap rpcconsumer "
	} else if serviceParams.ServiceType == ServiceTypeProvider {
		content += "  ExecStart=" + workingDir + "lavap rpcprovider "
	}
	content += ".lavavisor/services/protocol_yml_configs/" + filepath.Base(serviceParams.ServiceConfigFile) +
		" --from " + serviceParams.FromUser +
		" --keyring-backend " + serviceParams.KeyringBackend +
		" --parallel-connections " + fmt.Sprint(serviceParams.ParallelConnection) +
		" --chain-id " + serviceParams.ChainID +
		" --geolocation " + fmt.Sprint(serviceParams.GeoLocation) +
		" --log_level " + serviceParams.LogLevel +
		" --node " + serviceParams.Node + "\n"

	content += "  User=" + currentUser.Username + "\n"
	content += "  Restart=always\n"
	content += "  RestartSec=180\n"
	content += "  LimitNOFILE=infinity\n"
	content += "  LimitNPROC=infinity\n"
	content += "[Install]\n"
	content += "  WantedBy=multi-user.target\n"

	filePath := serviceParams.LavavisorServicesDir + "/" + serviceId + ".service"
	err = os.WriteFile(filePath, []byte(content), os.ModePerm)
	if err != nil {
		return "", utils.LavaFormatError("[Lavavisor] error writing to service file", err)
	}

	// Create a symbolic link to the systemd directory.
	if createLink {
		if err := createSystemdSymlink(filePath, serviceId+".service"); err != nil {
			return "", utils.LavaFormatError("[Lavavisor] error creating symbolic link", err)
		}
	}

	utils.LavaFormatInfo("[Lavavisor] Service file has been created successfully", utils.Attribute{Key: "Path", Value: filePath})
	// Extract filename from filePath
	filename := filepath.Base(filePath)

	return filename, nil
}

func WriteToConfigFile(lavavisorPath string, serviceFileName string) error {
	configPath := filepath.Join(lavavisorPath, "config.yml")
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return utils.LavaFormatError("[Lavavisor] error opening config.yml for appending", err)
	}
	defer file.Close()

	// Read the existing contents of the file to validate we don't have serviceFileName already there
	existingData, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	if strings.Contains(string(existingData), serviceFileName) {
		utils.LavaFormatInfo("[Lavavisor] Service already exists in ~/.lavavisor/config.yml skipping " + serviceFileName)
		return nil
	}

	// Check if the file is newly created by checking its size
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	if fileInfo.Size() == 0 {
		_, err = file.WriteString("services:\n")
		if err != nil {
			return err
		}
	}
	// Append the new service name
	_, err = file.WriteString("  - " + serviceFileName + "\n")
	if err != nil {
		return err
	}
	return nil
}

// Create a symbolic link to the /etc/systemd/system/ directory.
func createSystemdSymlink(source string, serviceName string) error {
	target := "/etc/systemd/system/" + serviceName
	// Check if the target exists
	if _, err := os.Lstat(target); err == nil || os.IsExist(err) {
		// Check if it's actually a symlink. If not, return an error.
		if _, err := os.Readlink(target); err == nil {
			// If there's a symlink exists, remove it.
			cmdRemove := exec.Command("sudo", "rm", target)
			if err := cmdRemove.Run(); err != nil {
				return utils.LavaFormatError("[Lavavisor] failed to remove existing link.", nil, utils.Attribute{Key: "Target", Value: target})
			}
			utils.LavaFormatInfo("[Lavavisor] Old service file links are removed.", utils.Attribute{Key: "Path", Value: target})
		} else {
			return utils.LavaFormatError("[Lavavisor] file exists and is not a symlink.", nil, utils.Attribute{Key: "Target", Value: target})
		}
	}
	// Create a new symbolic link pointing to the intended source.
	// Use 'sudo' to create the symlink with elevated permissions.
	cmd := exec.Command("sudo", "ln", "-s", source, target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating symbolic link: %v", err)
	}
	utils.LavaFormatInfo("[Lavavisor] Symbolic link for to root has been created successfully.", utils.Attribute{Key: "Path", Value: target})

	return nil
}
