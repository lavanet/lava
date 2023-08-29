import chalk from "chalk";

export enum LogLevel {
  NoPrints = 0,
  Error = 1,
  Warn = 2,
  Success = 3,
  Info = 4,
  Debug = 5,
}

class LoggerClass {
  private logLevel: LogLevel = LogLevel.Error; // default log level is Error

  public SetLogLevel(level: LogLevel | string | undefined) {
    if (!level) {
      return;
    }
    if (typeof level === "string") {
      let levelUpperCase = "";
      if (level.length > 0) {
        levelUpperCase = level[0].toUpperCase() + level.slice(1); // debug -> Debug to match enum keys
      }
      const enumLogLevel = LogLevel[levelUpperCase as keyof typeof LogLevel];
      if (enumLogLevel !== undefined) {
        this.logLevel = enumLogLevel;
      } else {
        console.log("Failed Setting LogLevel unknown key", level);
      }
    } else {
      this.logLevel = level;
    }
  }

  public debug(message?: any, ...optionalParams: any[]) {
    if (this.logLevel >= LogLevel.Debug) {
      console.log(chalk.cyan("[Debug]", message, optionalParams));
    }
  }

  public info(message?: any, ...optionalParams: any[]) {
    if (this.logLevel >= LogLevel.Info) {
      console.log(chalk.white("[Info]", message, ...optionalParams));
    }
  }

  public success(message?: any, ...optionalParams: any[]) {
    if (this.logLevel >= LogLevel.Success) {
      console.log(chalk.green("[Success]", message, ...optionalParams));
    }
  }

  public warn(message?: any, ...optionalParams: any[]) {
    if (this.logLevel >= LogLevel.Warn) {
      console.log(chalk.yellow("[Warning]", message, ...optionalParams));
    }
  }

  public error(message?: any, ...optionalParams: any[]) {
    if (this.logLevel >= LogLevel.Error) {
      console.log(chalk.red("[Error]", message, ...optionalParams));
    }
  }

  public emptyLine() {
    console.log();
  }
}

export const Logger = new LoggerClass();
